package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("freeapi %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
