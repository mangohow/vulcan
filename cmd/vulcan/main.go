package main

import (
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/generator/dbgenerator"
	parser2 "github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/log"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
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
			options := parseCommandArgs(cmd)
			if options.DDLGen {
				if err := generateMapperBySqlDDL(options); err != nil {
					log.Fatalf("%v", err)
				}
				return
			}

			if options.StructGen {
				if err := generateMapperByStructModel(options, nil); err != nil {
					log.Fatalf("%v", err)
				}
				return
			}

			if err := generateMapper(options.File); err != nil {
				log.Fatalf("%v", err)
			}
		},
	}
)

func generateMapperByStructModel(options *command.CommandOptions, modelSpecs []*types.TypeSpec) (err error) {
	if modelSpecs == nil {
		// 解析model struct
		modelSpecs, err = dbparser.ParseModelFile()
		if err != nil {
			return err
		}
	}

	// 生成中间代码
	source, err := dbgenerator.GenerateCRUDFuncsByModel(modelSpecs)
	if err != nil {
		return err
	}

	// TODO  写入package、 go generate等
	sourcePath := options.OutPutPath
	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "get abs output path failed")
	}
	exist, err := utils.IsDirExists(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "stat %s failed", sourcePath)
	}
	if !exist {
		if err = os.MkdirAll(sourcePath, 0644); err != nil {
			return errors.Wrapf(err, "mkdir %s error", sourcePath)
		}
	}

	filename := filepath.Base(options.File)
	index := strings.LastIndex(filename, ".")
	if index != -1 {
		filename = filename[:index]
	}
	filename = filepath.Join(sourcePath, filename+".go")
	if err = os.WriteFile(filename, source, 0644); err != nil {
		return errors.Wrapf(err, "write source to %s failed", filename)
	}

	// 如果不需要保留中间代码, 则需要删除
	if !options.IntermediateCode {
		defer os.Remove(filename)
	}

	// 根据中间代码生成最终代码
	return generateMapper(filename)
}

func generateMapperBySqlDDL(options *command.CommandOptions) error {
	if options.File == "" {
		return errors.Errorf("source file is not specified")
	}

	if options.OutPutPath == "" {
		return errors.Errorf("output path is not specified")
	}

	if options.ModelOutputPath == "" {
		return errors.Errorf("model struct path is not specified")
	}

	// 解析ddl语句
	ddlSpecs, err := dbparser.ParseSqlFile(options.File)
	if err != nil {
		return err
	}

	tags := strings.Split(options.Tags, ",")
	tags = stream.Map(tags, func(t string) string {
		return strings.TrimSpace(t)
	})

	genOptions := &dbgenerator.ModelGenOptions{
		TablePrefix: options.TablePrefix,
		ModelSuffix: options.ModelSuffix,
		RepoSuffix:  options.RepoSuffix,
		UseNull:     options.UseNullable,
		TagKeys:     tags,
	}
	// 生成model文件
	modelTypes, err := dbgenerator.GenerateGoModelStructList(ddlSpecs, options, genOptions)
	if err != nil {
		return err
	}

	return generateMapperByStructModel(options, modelTypes)
}

func init() {
	if err := command.BindCommand(rootCmd, &command.CommandOptions{}); err != nil {
		log.Fatalf("bind command error, err: %v", err)
	}
}

func parseCommandArgs(cmd *cobra.Command) *command.CommandOptions {
	options := &command.CommandOptions{}
	err := command.BindOptions(cmd, options)
	if err != nil {
		log.Fatalf("bind options error, err: %v", err)
	}

	if options.File == "" {
		// 获取go generate传入的参数
		goSourceFile := os.Getenv("GOFILE")
		dir, err := os.Getwd()
		if err != nil {
			log.Fatalf("get cwd: %v", err)
		}
		options.File = filepath.Join(dir, goSourceFile)
	}

	return options
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
