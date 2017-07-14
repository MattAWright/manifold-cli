package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/manifoldco/go-manifold"
	"github.com/urfave/cli"

	"github.com/manifoldco/manifold-cli/clients"
	"github.com/manifoldco/manifold-cli/config"
	"github.com/manifoldco/manifold-cli/session"

	"github.com/manifoldco/manifold-cli/generated/marketplace/client/resource"
)

func init() {
	runCmd := cli.Command{
		Name:   "run",
		Usage:  "Run a process and inject secrets into its environment",
		Action: run,
		Flags: []cli.Flag{
			appFlag(),
		},
	}

	cmds = append(cmds, runCmd)
}

func run(cliCtx *cli.Context) error {
	ctx := context.Background()
	args := cliCtx.Args()

	if len(args) == 0 {
		return newUsageExitError(cliCtx, fmt.Errorf("A command is required"))
	} else if len(args) == 1 { //only one arg, maybe it was quoted
		args = strings.Split(args[0], " ")
	}

	appName := cliCtx.String("app")
	if appName != "" {
		name := manifold.Name(appName)
		if err := name.Validate(nil); err != nil {
			return newUsageExitError(cliCtx, errInvalidAppName)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return cli.NewExitError("Could not load config: "+err.Error(), -1)
	}

	s, err := session.Retrieve(ctx, cfg)
	if err != nil {
		return cli.NewExitError("Could not retrieve session: "+err.Error(), -1)
	}

	if !s.Authenticated() {
		return errMustLogin
	}

	marketplace, err := clients.NewMarketplace(cfg)
	if err != nil {
		return cli.NewExitError("Could not create marketplace client: "+err.Error(), -1)
	}

	p := resource.NewGetResourcesParamsWithContext(ctx)
	r, err := marketplace.Resource.GetResources(p, nil)
	if err != nil {
		return cli.NewExitError("Could not retrieve resources: "+err.Error(), -1)
	}

	resources := filterResourcesByAppName(r.Payload, appName)
	cMap, err := fetchCredentials(ctx, marketplace, resources)
	if err != nil {
		return cli.NewExitError("Could not retrieve credentials: "+err.Error(), -1)
	}

	credentials, err := flattenCMap(cMap)
	if err != nil {
		return cli.NewExitError("Could not flatten credential map: "+err.Error(), -1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = filterEnv()

	for name, value := range credentials {
		cmd.Env = append(cmd.Env, name+"="+value)
	}

	err = cmd.Start()
	if err != nil {
		return cli.NewExitError("Could not execute command: "+err.Error(), -1)
	}

	done := make(chan bool)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)

		select {
		case s := <-c:
			cmd.Process.Signal(s)
		case <-done:
			signal.Stop(c)
			return
		}
	}()

	err = cmd.Wait()
	close(done)
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
				return nil
			}

			return err
		}
	}

	return nil
}

func filterEnv() []string {
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, session.EnvManifoldUser+"=") || strings.HasPrefix(e, session.EnvManifoldPass+"=") {
			continue
		}

		env = append(env, e)
	}

	return env
}
