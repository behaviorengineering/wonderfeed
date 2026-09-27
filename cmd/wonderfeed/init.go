package main

import (
	"fmt"
	"io"

	"github.com/behaviorengineering/wonderfeed/internal/config"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	opts := config.InitOptions{}
	for _, a := range args {
		switch a {
		case "-h", "--help", "help":
			fmt.Fprint(stdout, `Usage: wonderfeed init [--force]

Creates ~/.config/wonderfeed/config.yaml when missing (mode 0600) and refreshes
config.yaml.example. Lists secrets in YAML for env, OS credential store, or
optional secrets.enc.yaml (SOPS). Field values use ${VAR} placeholders only.

`)
			return 0
		case "--force":
			opts.Force = true
		default:
			fmt.Fprintf(stderr, "unknown init flag: %s\n", a)
			return 2
		}
	}
	result, err := config.InitUserConfigFiles(opts)
	if err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Config directory: %s\n", result.ConfigDir)
	for _, p := range result.Created {
		fmt.Fprintf(stdout, "Created: %s\n", p)
	}
	for _, p := range result.Skipped {
		fmt.Fprintf(stdout, "Skipped (already exists): %s\n", p)
	}
	if len(result.Created) == 0 && len(result.Skipped) == 0 {
		fmt.Fprintln(stdout, "No files changed.")
	}
	return 0
}
