// Package main is the Wonderfeed host operator CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/backup"
	"github.com/behaviorengineering/wonderfeed/internal/provider/ytzero"
)

// version is overridden at link time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		printUsage(stderr)
		return 2
	}
	switch args[1] {
	case "version":
		fmt.Fprintf(stdout, "wonderfeed %s\n", version)
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "provider":
		return runProvider(args[2:], stdout, stderr)
	case "backup":
		return runBackup(args[2:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[1])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: wonderfeed <command>

Commands:
  version              Print binary version
  help                 Show this help
  provider status      Show submodule pin and compose state
  provider prepare     Ensure data dirs, .env, and Docker (no start)
  provider up          Start PostgreSQL + YT Zero detached via compose
  provider down        Stop the stack without removing volumes
  provider health      GET /api/health on the local provider
  provider open        Print the local provider UI URL
  backup create        Encrypted Postgres + portable state backup
  backup list          List local (and optional S3) backups
  backup restore <id>  Restore a backup (requires --confirm)

Interactive stack: make serve (process-compose). Detached: wonderfeed provider up.

`)
}

func runProvider(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: wonderfeed provider <status|prepare|up|down|health|open>")
		return 2
	}
	paths, err := ytzero.ResolvePaths("")
	if err != nil {
		fmt.Fprintf(stderr, "provider: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch args[0] {
	case "status":
		report, err := ytzero.CollectStatus(ctx, paths, ytzero.DefaultBaseURL)
		if err != nil {
			fmt.Fprintf(stderr, "provider status: %v\n", err)
			return 1
		}
		out, err := ytzero.FormatStatus(report)
		if err != nil {
			fmt.Fprintf(stderr, "provider status: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, out)
		return 0
	case "prepare":
		if err := ytzero.Prepare(ctx, paths); err != nil {
			fmt.Fprintf(stderr, "provider prepare: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "Provider data dirs and env ready.")
		return 0
	case "up":
		upCtx, upCancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer upCancel()
		if err := ytzero.Up(upCtx, paths); err != nil {
			fmt.Fprintf(stderr, "provider up: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "YT Zero + PostgreSQL starting. UI: %s\n", ytzero.DefaultBaseURL)
		return 0
	case "down":
		if err := ytzero.Down(ctx, paths); err != nil {
			fmt.Fprintf(stderr, "provider down: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "Provider stack stopped (volumes kept).")
		return 0
	case "health":
		client := ytzero.NewHealthClient(ytzero.DefaultBaseURL)
		result, err := client.CheckGET(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "provider health: %v\n", err)
			if result.Raw != "" {
				fmt.Fprintln(stderr, result.Raw)
			}
			return 1
		}
		if result.Body != nil {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(result.Body); err != nil {
				fmt.Fprintf(stderr, "provider health: encode: %v\n", err)
				return 1
			}
			return 0
		}
		fmt.Fprintln(stdout, strings.TrimSpace(result.Raw))
		return 0
	case "open":
		fmt.Fprintln(stdout, ytzero.DefaultBaseURL)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown provider command: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: wonderfeed provider <status|prepare|up|down|health|open>")
		return 2
	}
}

func runBackup(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: wonderfeed backup <create|list|restore>")
		return 2
	}
	paths, err := ytzero.ResolvePaths("")
	if err != nil {
		fmt.Fprintf(stderr, "backup: %v\n", err)
		return 1
	}
	cfg, err := backup.CreateConfig(paths, nil)
	if err != nil {
		fmt.Fprintf(stderr, "backup: %v\n", err)
		return 1
	}

	switch args[0] {
	case "create":
		for _, a := range args[1:] {
			if a == "--local-only" {
				cfg.LocalOnly = true
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		result, err := backup.Create(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "backup create: %v\n", err)
			return 1
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(stderr, "backup create: encode: %v\n", err)
			return 1
		}
		return 0
	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		items, err := backup.List(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "backup list: %v\n", err)
			return 1
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(items); err != nil {
			fmt.Fprintf(stderr, "backup list: encode: %v\n", err)
			return 1
		}
		return 0
	case "restore":
		if len(args) < 2 {
			fmt.Fprintln(stderr, "usage: wonderfeed backup restore <id> --confirm")
			return 2
		}
		opts := backup.RestoreOptions{ID: args[1]}
		for _, a := range args[2:] {
			if a == "--confirm" {
				opts.Confirm = true
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		result, err := backup.Restore(ctx, cfg, opts)
		if err != nil {
			fmt.Fprintf(stderr, "backup restore: %v\n", err)
			return 1
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(stderr, "backup restore: encode: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown backup command: %s\n", args[0])
		fmt.Fprintln(stderr, "usage: wonderfeed backup <create|list|restore>")
		return 2
	}
}
