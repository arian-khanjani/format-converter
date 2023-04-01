package csv

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
)

var version = "0.0.2"

var rootCmd = &cobra.Command{
	Use:     "convert",
	Version: version,
	Short:   "convert - a simple CLI to convert files to the desired format",
	Long:    "convert is a CLI tool to convert different data formats to the desired and supported formats",
	Run:     func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "CLI could not start: %e", err)
		if err != nil {
			log.Fatalln(err)
		}
		os.Exit(1)
	}
}
