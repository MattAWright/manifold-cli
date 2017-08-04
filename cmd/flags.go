package main

import (
	"github.com/urfave/cli"

	"github.com/manifoldco/manifold-cli/placeholder"
)

func formatFlag(defaultValue, description string) cli.Flag {
	return placeholder.New("format, f", "FORMAT", description, defaultValue, "MANIFOLD_FORMAT", false)
}

func appFlag() cli.Flag {
	return cli.StringFlag{
		Name:   "app, a",
		Usage:  "Filter output to only items related to the specified App.",
		Value:  "",
		EnvVar: "MANIFOLD_APP",
	}
}

func planFlag() cli.Flag {
	return cli.StringFlag{
		Name:   "plan, p",
		Usage:  "Specify a plan",
		Value:  "",
		EnvVar: "MANIFOLD_PLAN",
	}
}

func skipFlag() cli.Flag {
	return cli.BoolFlag{
		Name:   "no-wait, w",
		Usage:  "Do not wait when creating, updating, or deleting a resource",
		EnvVar: "MANIFOLD_DONT_WAIT",
	}
}