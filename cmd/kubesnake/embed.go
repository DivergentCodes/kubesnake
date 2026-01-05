// Package embed is the embed command for the kubesnake CLI.
package main

import (
	"fmt"

	"github.com/DivergentCodes/kubesnake/internal/config"
	"github.com/spf13/cobra"
)

// newEmbedCmd returns the embed command for the kubesnake CLI.
func newEmbedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "embed <config.json>",
		Short: "Embed config into this kubesnake binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			exePath, err := config.EmbedConfigFileIntoSelf(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("embedded config into %s\n", exePath)
			return nil
		},
	}
}
