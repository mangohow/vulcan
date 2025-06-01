package gencmd

import (
	"github.com/mangohow/vulcan/internal/ast/generator/dbgenerator"
	parser2 "github.com/mangohow/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/internal/log"
	"github.com/spf13/cobra"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "gen db mapper",
	Long:  "gen db mapper",
	Run: func(cmd *cobra.Command, args []string) {
		goSourceFile := os.Getenv("GOFILE")
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("get cwd: %v", err)
		}
		log.Infof("%s %s", dir, goSourceFile)

		asbPath := filepath.Join(dir, goSourceFile)
		if err := generate(asbPath); err != nil {
			log.Fatalf("parse db files: %v", err)
			os.Exit(1)
		}
	},
}

func generate(path string) error {
	fst := token.NewFileSet()
	dependencyManager := parser2.NewDependencyManager(fst)
	parser := dbparser.NewFileParser(fst, dependencyManager)
	parsedFile, err := parser.Parse(path)
	if err != nil {
		return err
	}

	idx := strings.Index(path, ".")
	newFileName := path[:idx] + "_gen" + path[idx:]
	generator := dbgenerator.NewFileGenerator(parsedFile)
	if err := generator.Execute(newFileName); err != nil {
		return err
	}

	return nil
}
