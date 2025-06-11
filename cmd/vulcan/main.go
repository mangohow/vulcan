package main

import (
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/generator/dbgenerator"
	parser2 "github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/log"
	"github.com/spf13/cobra"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var (
	rootCmd = &cobra.Command{
		Use:   "vulcan",
		Short: "vulcan is a cli tool for generating go codes",
		Long:  "vulcan is a cli tool for generating go codes",
		Run: func(cmd *cobra.Command, args []string) {
			cmdArgs := parseCommandArgs(cmd)
			if err := generateMapper(cmdArgs.file); err != nil {
				log.Fatalf("%v", err)
			}
		},
	}
)

func init() {
	rootCmd.Flags().StringP("file", "f", "", "Specify the file to generate the code")
	rootCmd.Flags().StringP("dir", "d", "", "Specify the directory to generate the code")
	rootCmd.Flags().StringP("mode", "m", "mapper", "Specify the generation mode: [mapper, copy]")
}

type commandArgs struct {
	file string
	dir  string
	mode string
}

func parseCommandArgs(cmd *cobra.Command) commandArgs {
	var (
		args commandArgs
		err  error
	)

	args.file, err = cmd.Flags().GetString("file")
	errExit(err, "file")
	args.dir, err = cmd.Flags().GetString("dir")
	errExit(err, "dir")
	args.mode, err = cmd.Flags().GetString("mode")
	errExit(err, "mode")

	if args.file == "" {
		// 获取go generate传入的参数
		goSourceFile := os.Getenv("GOFILE")
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("get cwd: %v", err)
		}
		args.file = filepath.Join(dir, goSourceFile)
	}

	return args
}

func errExit(err error, name string) {
	if err != nil {
		log.Fatalf("Get commcand line arg %s error: %v\n", name, err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%v\n", err)
	}
}

func generateMapper(path string) error {
	fst := token.NewFileSet()
	dependencyManager := parser2.NewDependencyManager(fst)
	parser := dbparser.NewFileParser(fst, dependencyManager)
	parsedFile, err := parser.Parse(path)
	if err != nil {
		return errors.Wrapf(err, "parse file %s failed", path)
	}

	idx := strings.Index(path, ".")
	newFileName := path[:idx] + "_gen" + path[idx:]
	generator := dbgenerator.NewFileGenerator(parsedFile)
	if err := generator.Execute(newFileName); err != nil {
		return errors.Wrapf(err, "generate file %s failed", path)
	}

	return nil
}
