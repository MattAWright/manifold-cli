package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/juju/ansiterm"
	"github.com/manifoldco/go-manifold"
	"github.com/urfave/cli"

	"strings"

	"github.com/manifoldco/manifold-cli/api"
	"github.com/manifoldco/manifold-cli/clients"
	"github.com/manifoldco/manifold-cli/color"
	"github.com/manifoldco/manifold-cli/errs"
	"github.com/manifoldco/manifold-cli/generated/identity/client"
	inviteClient "github.com/manifoldco/manifold-cli/generated/identity/client/invite"
	teamClient "github.com/manifoldco/manifold-cli/generated/identity/client/team"
	"github.com/manifoldco/manifold-cli/generated/identity/models"
	iModels "github.com/manifoldco/manifold-cli/generated/identity/models"
	"github.com/manifoldco/manifold-cli/middleware"
	"github.com/manifoldco/manifold-cli/prompts"
)

func init() {
	appCmd := cli.Command{
		Name:     "teams",
		Usage:    "Manage your teams",
		Category: "ADMINISTRATIVE",
		Subcommands: []cli.Command{
			{
				Name:      "create",
				Usage:     "Create a new team",
				ArgsUsage: "[name]",
				Action:    middleware.Chain(middleware.EnsureSession, createTeamCmd),
			},
			{
				Name:      "update",
				Usage:     "Update an existing team",
				ArgsUsage: "[label]",
				Flags: []cli.Flag{
					nameFlag(),
				},
				Action: middleware.Chain(middleware.EnsureSession, updateTeamCmd),
			},
			{
				Name:      "invite",
				ArgsUsage: "[email] [name]",
				Usage:     "Invite a user to join a team",
				Flags:     teamFlags,
				Action: middleware.Chain(middleware.LoadDirPrefs, middleware.EnsureSession,
					middleware.LoadTeamPrefs, inviteToTeamCmd),
			},
			{
				Name:  "members",
				Usage: "List members of a team",
				Flags: teamFlags,
				Action: middleware.Chain(middleware.LoadDirPrefs, middleware.EnsureSession,
					middleware.LoadTeamPrefs, membersTeamCmd),
			},
			{
				Name:   "list",
				Usage:  "List all your teams",
				Action: middleware.Chain(middleware.EnsureSession, listTeamCmd),
			},
			{
				Name:      "leave",
				ArgsUsage: "[name]",
				Usage:     "Remove yourself from a team",
				Action:    middleware.Chain(middleware.EnsureSession, leaveTeamCmd),
			},
			{
				Name:      "set-role",
				ArgsUsage: "[email] [role]",
				Usage:     "Change the role of an existing member",
				Flags:     teamFlags,
				Action: middleware.Chain(middleware.LoadDirPrefs, middleware.EnsureSession,
					middleware.LoadTeamPrefs, setRoleCmd),
			},
		},
	}

	cmds = append(cmds, appCmd)
}

func createTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	if err := maxOptionalArgsLength(cliCtx, 1); err != nil {
		return err
	}

	teamName, err := optionalArgLabel(cliCtx, 0, "team")
	if err != nil {
		return err
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	autoSelect := teamName != ""
	teamName, err = prompts.TeamName(teamName, autoSelect)
	if err != nil {
		return prompts.HandleSelectError(err, "Failed to name team")
	}

	if err := createTeam(ctx, teamName, client.Identity); err != nil {
		return cli.NewExitError(fmt.Sprintf("Could not create team: %s", err), -1)
	}

	fmt.Printf("Your team '%s' has been created\n", teamName)
	return nil
}

func updateTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	if err := maxOptionalArgsLength(cliCtx, 1); err != nil {
		return err
	}

	teamName, err := optionalArgLabel(cliCtx, 0, "team")
	if err != nil {
		return err
	}

	newTeamName, err := validateName(cliCtx, "name", "team")
	if err != nil {
		return err
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	team, err := selectTeam(ctx, teamName, client.Identity)
	if err != nil {
		return err
	}

	autoSelect := newTeamName != ""
	newTeamName, err = prompts.TeamName(newTeamName, autoSelect)
	if err != nil {
		return prompts.HandleSelectError(err, "Could not validate name")
	}

	if err := updateTeam(ctx, team, newTeamName, client.Identity); err != nil {
		return cli.NewExitError(fmt.Sprintf("Could not update team: %s", err), -1)
	}

	fmt.Printf("Your team \"%s\" has been updated\n", newTeamName)
	return nil
}

func inviteToTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	email, err := optionalArgEmail(cliCtx, 0, "user")
	if err != nil {
		return err
	}

	args := cliCtx.Args().Tail()
	name := strings.Join(args, " ")

	// read team as an optional flag
	teamName, err := validateName(cliCtx, "team")
	if err != nil {
		return err
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	team, err := selectTeam(ctx, teamName, client.Identity)
	if err != nil {
		return err
	}

	if name == "" {
		name, err = prompts.FullName("")
		if err != nil {
			return err
		}
	}

	if email == "" {
		email, err = prompts.Email("")
		if err != nil {
			return err
		}
	}

	role, err := prompts.SelectRole()
	if err != nil {
		return prompts.HandleSelectError(err, "Could not select role")
	}

	if err := inviteToTeam(ctx, team, email, name, role, client.Identity); err != nil {
		return cli.NewExitError(fmt.Sprintf("Could not invite to team: %s", err), -1)
	}

	fmt.Printf("An invite has been sent to %s <%s>\n", name, email)
	return nil
}

func membersTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	teamID, err := validateTeamID(cliCtx)
	if err != nil {
		return err
	}
	if teamID == nil {
		return cli.NewExitError("Can't view members for a non-team. Use `manifold switch` to select a team.", -1)
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	prompts.SpinStart("Fetching Team Members")
	members, err := clients.FetchTeamMembers(ctx, teamID.String(), client.Identity)
	prompts.SpinStop()
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to fetch list of teams: %s", err), -1)
	}

	prompts.SpinStart("Fetching Invites")
	invites, err := clients.FetchInvites(ctx, teamID.String(), client.Identity)
	prompts.SpinStop()
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to fetch list of invites: %s", err), -1)
	}

	fmt.Printf("%d members and %d invites\n", len(members), len(invites))
	fmt.Println("Use `manifold switch` to change to a different team")
	fmt.Println()

	w := ansiterm.NewTabWriter(os.Stdout, 0, 0, 8, ' ', 0)

	w.SetStyle(ansiterm.Bold)
	w.SetForeground(ansiterm.Gray)
	fmt.Fprintln(w, "Name\tEmail\tRole\tStatus")
	w.ClearStyle(ansiterm.Bold)
	w.Reset()
	for _, m := range members {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Name, m.Email, m.Role, "active")
	}
	w.SetStyle(ansiterm.Faint)
	for _, i := range invites {
		role := i.Body.Role
		if role == "" {
			role = "admin"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", i.Body.Name, i.Body.Email, role, "pending")
	}
	w.ClearStyle(ansiterm.Faint)
	return w.Flush()
}

func listTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	prompts.SpinStart("Fetching Team Members")
	teams, err := clients.FetchTeamsMembersCount(ctx, client.Identity)
	prompts.SpinStop()
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to fetch list of teams: %s", err), -1)
	}

	w := ansiterm.NewTabWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(w, "%s\t%s\n", color.Bold("Name"), color.Bold("Members"))

	sort.Slice(teams, func(i int, j int) bool {
		a := strings.ToLower(teams[i].Name)
		b := strings.ToLower(teams[j].Name)
		return b > a
	})

	for _, team := range teams {
		fmt.Fprintf(w, "%s\t%d\n", team.Name, team.Members)
	}
	return w.Flush()
}

func leaveTeamCmd(cliCtx *cli.Context) error {
	ctx := context.Background()

	if err := maxOptionalArgsLength(cliCtx, 1); err != nil {
		return err
	}

	teamName, err := optionalArgLabel(cliCtx, 0, "team")
	if err != nil {
		return err
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	team, err := selectTeam(ctx, teamName, client.Identity)
	if err != nil {
		return err
	}

	memberships, err := clients.FetchMemberships(ctx, client.Identity)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to fetch user memberships: %s", err), -1)
	}

	var membershipID manifold.ID

	for _, m := range memberships {
		if m.Body.TeamID == team.ID {
			membershipID = m.ID
		}
	}

	if membershipID.IsEmpty() {
		return cli.NewExitError("No memberships found", -1)
	}

	if err := leaveTeam(ctx, membershipID, client.Identity); err != nil {
		return cli.NewExitError(fmt.Sprintf("Could not leave team: %s", err), -1)
	}

	fmt.Printf("You have left the team \"%s\"\n", team.Body.Name)
	return nil
}

func createTeam(ctx context.Context, teamName string, identityClient *client.Identity) error {
	createTeam := &models.CreateTeam{
		Body: &models.CreateTeamBody{
			Name:  manifold.Name(teamName),
			Label: generateLabel(teamName),
		},
	}

	c := teamClient.NewPostTeamsParamsWithContext(ctx)
	c.SetBody(createTeam)

	_, err := identityClient.Team.PostTeams(c, nil)
	if err != nil {

		switch e := err.(type) {
		case *teamClient.PostTeamsBadRequest:
			return e.Payload
		case *teamClient.PostTeamsUnauthorized:
			return e.Payload
		case *teamClient.PostTeamsConflict:
			return e.Payload
		case *teamClient.PostTeamsInternalServerError:
			return errs.ErrSomethingWentHorriblyWrong
		default:
			return err
		}
	}

	return nil
}

func updateTeam(ctx context.Context, team *models.Team, teamName string, identityClient *client.Identity) error {
	updateTeam := &models.UpdateTeam{
		Body: &models.UpdateTeamBody{
			Name:  manifold.Name(teamName),
			Label: generateLabel(teamName),
		},
	}

	c := teamClient.NewPatchTeamsIDParamsWithContext(ctx)
	c.SetBody(updateTeam)
	c.SetID(team.ID.String())

	_, err := identityClient.Team.PatchTeamsID(c, nil)
	if err != nil {
		switch e := err.(type) {
		case *teamClient.PatchTeamsIDBadRequest:
			return e.Payload
		case *teamClient.PatchTeamsIDInternalServerError:
			return errs.ErrSomethingWentHorriblyWrong
		default:
			return err
		}
	}

	return nil
}

func inviteToTeam(ctx context.Context, team *models.Team, email,
	name, role string, identityClient *client.Identity) error {
	c := inviteClient.NewPostInvitesParamsWithContext(ctx)

	params := &iModels.CreateInvite{
		Body: &iModels.CreateInviteBody{
			Email:  manifold.Email(email),
			Name:   iModels.UserDisplayName(name),
			TeamID: team.ID,
			Role:   models.RoleLabel(role),
		},
	}

	c.SetBody(params)

	_, _, err := identityClient.Invite.PostInvites(c, nil)
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *teamClient.PostTeamsUnauthorized:
		return e.Payload
	case *teamClient.PostTeamsInternalServerError:
		return errs.ErrSomethingWentHorriblyWrong
	default:
		return err
	}
}

func leaveTeam(ctx context.Context, membershipID manifold.ID,
	identityClient *client.Identity) error {
	c := teamClient.NewDeleteMembershipsIDParamsWithContext(ctx)
	c.SetID(membershipID.String())

	_, err := identityClient.Team.DeleteMembershipsID(c, nil)

	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *teamClient.DeleteMembershipsIDUnauthorized:
		return e.Payload
	case *teamClient.DeleteMembershipsIDInternalServerError:
		return errs.ErrSomethingWentHorriblyWrong
	default:
		return err
	}
}

// fetchTeams retrieves all user's team and prompt to select which team the cmd
// will be applied to.
func selectTeam(ctx context.Context, teamName string, identityClient *client.Identity) (*iModels.Team, error) {
	teams, err := clients.FetchTeams(ctx, identityClient)
	if err != nil {
		return nil, cli.NewExitError(fmt.Sprintf("Failed to fetch list of teams: %s", err), -1)
	}

	if len(teams) == 0 {
		return nil, errs.ErrNoTeams
	}

	idx, _, err := prompts.SelectTeam(teams, teamName, nil)
	if err != nil {
		return nil, prompts.HandleSelectError(err, "Could not select team")
	}

	team := teams[idx]

	return team, nil
}

func setRoleCmd(cliCtx *cli.Context) error {
	ctx := context.Background()
	args := cliCtx.Args()

	// Only allow args if exactly two are supplied
	// otherwise prompt for both values
	var roleLabel, email string
	var err error
	if len(args) > 0 {
		email, err = optionalArgEmail(cliCtx, 0, "email")
		if err != nil {
			return err
		}
		roleLabel, err = optionalArgLabel(cliCtx, 1, "role")
		if err != nil {
			return err
		}
	} else {
		email, err = prompts.Email("")
		if err != nil {
			return err
		}

		roleLabel, err = prompts.SelectRole()
		if err != nil {
			return prompts.HandleSelectError(err, "Could not select role")
		}
	}

	// Use the current context to determine the team to use
	teamID, err := validateTeamID(cliCtx)
	if err != nil {
		return err
	}
	if teamID == nil {
		return cli.NewExitError("Can't view members for a non-team. Use `manifold switch` to select a team.", -1)
	}

	client, err := api.New(api.Identity)
	if err != nil {
		return err
	}

	// Ensure the supplied email is a member
	prompts.SpinStart("Verifying team member")
	members, err := clients.FetchTeamMembers(ctx, teamID.String(), client.Identity)
	prompts.SpinStop()
	if err != nil {
		return err
	}
	var membershipID *manifold.ID
	var name string
	for _, m := range members {
		if string(m.Email) == email {
			name = string(m.Name)
			membershipID = &m.MembershipID
		}
	}
	if membershipID == nil {
		prompts.SpinStart("Checking invites")
		invites, err := clients.FetchInvites(ctx, teamID.String(), client.Identity)
		prompts.SpinStop()
		if err == nil {
			for _, i := range invites {
				if string(i.Body.Email) == email {
					return cli.NewExitError("Cannot modify role for pending invite.", -1)
				}
			}
		}
		return cli.NewExitError("Could not find team member for the email supplied.", -1)
	}

	// Update the membership row
	prompts.SpinStart(fmt.Sprintf("Updating role for %s <%s>", name, email))
	params := teamClient.NewPatchMembershipsIDParamsWithContext(ctx)
	params.SetID(membershipID.String())
	_, err = client.Identity.Team.PatchMembershipsID(params, nil)
	prompts.SpinStop()
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Could not update role: %s", err), -1)
	}

	fmt.Println("")
	fmt.Printf("%s <%s> now has the role of `%s`", name, email, roleLabel)
	return nil
}
