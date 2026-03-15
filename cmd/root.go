package cmd

import (
	"fmt"
	"os"
)

const version = "0.1.0-dev"

func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	switch os.Args[1] {
	case "serve":
		return runServe()
	case "init":
		return runInit()
	case "site":
		return runSite()
	case "keys":
		return runKeys()
	case "send":
		return runSend()
	case "doctor":
		return runDoctor()
	case "migrate":
		return runMigrate()
	case "version":
		fmt.Printf("dispatch %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func printUsage() {
	fmt.Printf(`dispatch %s — self-hosted email orchestration

Usage:
  dispatch <command> [options]

Commands:
  serve       Start the Dispatch server
  init        Generate default config files
  site        Manage sites (add, list, remove)
  keys        Manage API keys (create, list, revoke)
  send        Send a test email
  doctor      Check config, backends, DNS
  migrate     Run database migrations
  version     Print version

Run 'dispatch <command> --help' for details.
`, version)
}
