package main

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"

	"github.com/mangohow/gowlb/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/generator/dbgenerator"
	parser2 "github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "vulcan",
		Short: "vulcan is a cli tool for generating go codes",
		Long:  "vulcan is a cli tool for generating go codes",
		Example: `1.gen database access object:
    vulcan -f example/repo/userrepo.go
2.gen database access object by models:
    vulcan --struct-gen -f example/model/user.go -o example/repo --table-prefix t --repo-suffix repo
    or
    vulcan --struct-gen -f example/model/user.go -o example/repo/userrepo.go --table-prefix t
3.gen model and database access object by sql ddl:
    vulcan --ddl-gen -f example/db/db.sql -o example/repo --model-output example/model --table-prefix t --repo-suffix repo --tags json
`,
		Run: func(cmd *cobra.Command, args []string) {
			options := parseCommandArgs(cmd)
			if options.DDLGen {
				// 生成结构体
				if err := generateModelStructBySqlDDL(options); err != nil {
					log.Fatalf("%v", err)
				}

				// 生成mapper代码
				if err := generateMapperByStructModel(options); err != nil {
					log.Fatalf("%v", err)
				}
				return
			}

			if options.StructGen {
				if err := generateMapperByStructModel(options); err != nil {
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

func generateMapperByStructModel(options *command.CommandOptions) error {
	fst := token.NewFileSet()
	dependencyManager := parser2.NewDependencyManager(fst)
	parser := dbparser.NewModelStructParser(fst, dependencyManager, options)
	// 解析model struct
	modelSpecs, err := parser.Parse()
	if err != nil {
		return err
	}

	// 生成中间代码
	generator := dbgenerator.NewCRUDGenerator(options, modelSpecs)
	files, err := generator.Execute()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	// 删除中间代码
	if !options.IntermediateCode {
		defer func() {
			for _, file := range files {
				_ = os.Remove(file)
			}
		}()
	} else {
		utils.FormatGoSourceDir(filepath.Dir(files[0]))
	}

	// 生成最终mapper代码
	for _, file := range files {
		if err := generateMapper(file); err != nil {
			return err
		}
	}

	return nil
}

func generateModelStructBySqlDDL(options *command.CommandOptions) error {
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
	err = dbgenerator.GenerateGoModelStructList(ddlSpecs, options, genOptions)
	if err != nil {
		return err
	}

	return nil
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

	// 将路径都转换为绝对路径
	options.Dir = absPathHelper(options.Dir)
	options.OutPutPath = absPathHelper(options.OutPutPath)
	options.ModelOutputPath = absPathHelper(options.ModelOutputPath)

	return options
}

func absPathHelper(path string) string {
	if path == "" {
		return ""
	}
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("get abs path error, err: %v", err)
	}

	return path
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

	idx := strings.LastIndex(path, ".")
	newFileName := path[:idx] + "_gen" + path[idx:]
	generator := dbgenerator.NewFileGenerator(parsedFile)
	if err := generator.Execute(newFileName); err != nil {
		return errors.Wrapf(err, "generate file %s failed", path)
	}

	return nil
}
