package dbgenerator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
)

type CRUDGenFunc func(spec *types.ModelSpec, options *CommonOptions) (string, error)

var ( // TODO
	crudGenFuncMapping = map[string]CRUDGenFunc{
		"Add": func(modelSpec *types.ModelSpec, options *CommonOptions) (string, error) {
			return "", nil
		},
	}
)

type CRUDGenerator struct {
	options        *command.CommandOptions
	modelSpecs     []*types.ModelSpec
	importPackages []string
}

func NewCRUDGenerator(options *command.CommandOptions, modelSpecs []*types.ModelSpec) *CRUDGenerator {
	return &CRUDGenerator{
		options:    options,
		modelSpecs: modelSpecs,
		importPackages: []string{
			`"database/sql"`,
			`"github.com/mangohow/vulcan"`,
			`. "github.com/mangohow/vulcan/annotation"`,
		},
	}
}

func (g *CRUDGenerator) Execute() error {
	return g.generateCRUDFuncsByModel()
}

func (g *CRUDGenerator) generateCRUDFunc(modelSpec *types.ModelSpec, funcSpec *types.GenFuncSpec, commonOptions *CommonOptions) (string, error) {
	fn, ok := crudGenFuncMapping[funcSpec.KeyFuncName]
	if !ok {
		return "", errors.Errorf("gen func %s is invalid", funcSpec.FuncName)
	}

	return fn(modelSpec, commonOptions)
}

// 根据model生成中间代码
func (g *CRUDGenerator) generateCRUDFuncsByModel() error {
	// 获取mapper的包名
	outputPath := g.options.OutPutPath
	if strings.HasSuffix(outputPath, ".go") {
		outputPath = filepath.Dir(outputPath)
	}
	packageName, err := utils.GetPackageNameByDir(outputPath)
	if err != nil {
		return errors.Wrapf(err, "get mapper package name failed")
	}
	for _, modelSpec := range g.modelSpecs {
		modelObjName := strings.ToLower(modelSpec.ModelName[:1]) + modelSpec.ModelName[1:]
		modelTypeName := modelSpec.PackageName + "." + modelSpec.ModelName
		commonOptions := &CommonOptions{
			MapperName:    strings.ToUpper(modelSpec.ModelName[:1]) + modelSpec.ModelName[1:] + g.options.RepoSuffix,
			ReceiverName:  strings.ToLower(modelSpec.ModelName[:1]),
			ModelObjName:  modelObjName,
			ModelTypeName: modelTypeName,
			TableName:     modelSpec.TableName,
		}

		buffer := bytes.NewBuffer(nil)
		buffer.Grow(8 << 10)
		// 写入编译选项
		buffer.WriteString("//go:build vulcan\n\n")
		// 写入go generate
		buffer.WriteString("//go:generate ${GOPATH}/bin/vulcan gen db\n")
		// 写入package
		buffer.WriteString("package ")
		buffer.WriteString(packageName)
		buffer.WriteString("\n")
		// 写入import
		buffer.WriteString("import (\n")
		for _, imp := range g.importPackages {
			buffer.WriteString(imp)
			buffer.WriteString("\n")
		}
		buffer.WriteString(fmt.Sprintf("%q\n)\n\n", modelSpec.ImportPath))
		// 写入结构体声明
		buffer.WriteString(fmt.Sprintf("type %s struct {\n\tdb *sql.DB\n}\n\n", commonOptions.MapperName))
		// 写入结构体构造函数
		buffer.WriteString(fmt.Sprintf("func New%s(db *sql.DB) *%s {\n\treturn &%s{\n\t\tdb: db,\n\t}\n}\n\n", commonOptions.MapperName, commonOptions.MapperName, commonOptions.MapperName))

		for _, curdFnSpec := range modelSpec.FuncSpecs {
			source, err := g.generateCRUDFunc(modelSpec, curdFnSpec, commonOptions)
			if err != nil {
				return err
			}

			buffer.WriteString(source)
			buffer.WriteString("\n")
		}

		outputPath = g.options.OutPutPath
		if !strings.HasSuffix(outputPath, ".go") {
			outputPath = filepath.Join(outputPath, strings.ToLower(commonOptions.MapperName)+".go")
		}
		if err := g.writeSource(buffer, outputPath); err != nil {
			return errors.Wrapf(err, "write source failed")
		}
	}

	return nil
}

func (g *CRUDGenerator) writeSource(reader io.Reader, filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return errors.Wrapf(err, "open %s failed", filename)
	}
	defer file.Close()
	if _, err = fmt.Fprintf(file, fileHeaderComment); err != nil {
		return errors.Wrapf(err, "copy file header to %s failed", filename)
	}
	if _, err = io.Copy(file, reader); err != nil {
		return errors.Wrapf(err, "copy source to %s failed", filename)
	}

	return nil
}
