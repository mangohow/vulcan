package gencmd

import "github.com/spf13/cobra"

var StructCopyCmd = &cobra.Command{
	Use:   "copy",
	Short: "copy between structs",
	Long:  "copy between structs",
	Run: func(cmd *cobra.Command, args []string) {

	},
}
