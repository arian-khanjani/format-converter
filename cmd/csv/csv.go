package csv

import (
	"github.com/arian-khanjani/format-converter/pkg/csv"
	"github.com/spf13/cobra"
)

var pretty bool
var separator string

var csvCmd = &cobra.Command{
	Use:     "csv",
	Aliases: []string{"CSV"},
	Short:   "convert CSV file",
	Args:    cobra.ExactArgs(1),
	Run:     run,
}

func init() {
	csvCmd.Flags().BoolVarP(&pretty, "pretty", "p", true, "make the JSON pretty")
	csvCmd.Flags().StringVarP(&separator, "separator", "s", "comma", "change JSON separator")
	rootCmd.AddCommand(csvCmd)
}

func run(cmd *cobra.Command, args []string) {
	fileData, err := csv.GetFileData(args, pretty, separator)
	if err != nil {
		csv.ExitGracefully(err)
	}

	if _, err := csv.IsValidFile(fileData.Filepath); err != nil {
		csv.ExitGracefully(err)
	}

	writerChannel := make(chan map[string]interface{})
	done := make(chan bool)

	go csv.ProcessCSVFile(*fileData, writerChannel)
	go csv.WriteJSONFile(fileData.Filepath, writerChannel, done, fileData.Pretty)

	<-done
}
