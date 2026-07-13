package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var gendocsCmd = &cobra.Command{
	Use:    "gendocs",
	Short:  "Generate man docs for the CLI",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		path := "./man"

		if err := os.MkdirAll(path, 0o750); err != nil {
			log.Fatal(err)
		}

		header := &doc.GenManHeader{
			Title:   "mtracer CLI",
			Section: "1",
			Source:  "mtracer development team",
		}

		if err := doc.GenManTree(rootCmd, header, path); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Man pages generated with success at", path)
	},
}

func init() {
	rootCmd.AddCommand(gendocsCmd)
}
