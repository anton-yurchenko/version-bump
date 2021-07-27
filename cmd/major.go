package cmd

import (
	"version-bump/bump"

	"github.com/spf13/cobra"
)

var majorCmd = &cobra.Command{
	Use:   "major",
	Short: "Increment a major version",
	Run: func(cmd *cobra.Command, args []string) {
		run(bump.Major)
	},
}

func init() {
	rootCmd.AddCommand(majorCmd)
}
