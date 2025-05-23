package main

import (
	"github.com/mangohow/vulcan/cmd/gencmd"
	"github.com/spf13/cobra"
	"log"
)

var (
	rootCmd = &cobra.Command{
		Use:   "vulcan",
		Short: "vulcan is a cli tool for generating go codes",
		Long:  "vulcan is a cli tool for generating go codes",
	}

	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "generate go codes",
		Long:  "generate go codes, support db mapper and struct copy",
	}
)

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.AddCommand(gencmd.DbCmd)
	genCmd.AddCommand(gencmd.StructCopyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
