// Package main is the entry point for the kubesnake CLI.
package main

import (
	"fmt"
	"os"

	"github.com/DivergentCodes/kubesnake/internal/app"
	"github.com/spf13/cobra"
)

// main is the entry point for the kubesnake CLI.
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// banner returns the banner for the kubesnake CLI.
func banner() string {
	return "KubeSnake ( https://github.com/DivergentCodes/kubesnake )"
}

// newRootCmd returns the root command for the kubesnake CLI.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "kubesnake",
		Short:         banner(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, args []string) error { return app.Run() },
	}

	rootCmd.AddCommand(newEmbedCmd())
	return rootCmd
}
