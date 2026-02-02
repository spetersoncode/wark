package main

import (
	"fmt"
	"os"

	"github.com/diogenes-ai-code/wark/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cli.FormatErrorMessage(err))
		os.Exit(cli.ExitCode(err))
	}
}
