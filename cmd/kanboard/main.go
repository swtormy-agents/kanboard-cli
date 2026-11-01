package main

import (
	"os"

	"github.com/swtormy-agents/kanboard-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))
}
