package dbgenerator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mangohow/vulcan/cmd/vulcan/internal/log"

	"github.com/mangohow/gowlb/tools/collection"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils/stringutils"
)

var (
	sqlColumnToGoTypeMapping = map[string]string{
		// int 类型
		"bigint":    "int64",
		"int":       "int32",
		"mediumint": "int64",
		"smallint":  "int16",
		"tinyint":   "int8",

		// uint 类型
		"bigint unsigned":    "uint64",
		"int unsigned":       "uint32",
		"mediumint unsigned": "uint64",
		"smallint unsigned":  "uint16",
		"tinyint unsigned":   "uint8",

		// float类型
		"float":  "float32",
		"double": "float64",

		// string类型
		"char":       "string",
		"varchar":    "string",
		"text":       "string",
		"tinytext":   "string",
		"mediumtext": "string",
		"longtext":   "string",

		// []byte类型
		"binary":     "[]byte",
		"varbinary":  "[]byte",
		"blob":       "[]byte",
		"tinyblob":   "[]byte",
		"mediumblob": "[]byte",
		"longblob":   "[]byte",

		// 时间日期类型
		"date":      "time.Time",
		"time":      "time.Time",
		"datetime":  "time.Time",
		"timestamp": "int64",
		"year":      "int8",
	}

	sqlColumnToGoTypeMappingUseNull = map[string]string{
		// int 类型
		"bigint":    "NullInt64",
		"int":       "NullInt32",
		"mediumint": "NullInt64",
		"smallint":  "NullInt16",
		"tinyint":   "NullInt16",

		// uint 类型
		"bigint unsigned":    "uint64",
		"int unsigned":       "uint32",
		"mediumint unsigned": "uint64",
		"smallint unsigned":  "uint16",
		"tinyint unsigned":   "uint8",

		// float类型
		"float":  "NullFloat64",
		"double": "NullFloat64",

		// string类型
		"char":       "NullString",
		"varchar":    "NullString",
		"text":       "NullString",
		"tinytext":   "NullString",
		"mediumtext": "NullString",
		"longtext":   "NullString",

		// []byte类型
		"binary":     "NullByte",
		"varbinary":  "NullByte",
		"blob":       "NullByte",
		"tinyblob":   "NullByte",
		"mediumblob": "NullByte",
		"longblob":   "NullByte",

		// 时间日期类型
		"date":      "NullTime",
		"time":      "NullTime",
		"datetime":  "NullTime",
		"timestamp": "NullInt64",
		"year":      "NullInt16",
	}
)

const (
	primaryTag            = "pk"
	autoincrementTag      = "auto_incr"
	tablePropertyTypeName = "annotation.TableProperty"
)

func getGoTypeFromSqlType(sqlType string, useNull bool) (string, error) {
	var (
		res string
		ok  bool
	)

	// go版本>=1.22时, 可以使用sql.Null
	version := utils.GetSystemGoSubVersion()
	if useNull && version >= 22 {
		res, ok = sqlColumnToGoTypeMapping[sqlType]
		if !ok {
			return "", errors.Errorf("unsupport sql type %s", sqlType)
		}
		return fmt.Sprintf("sql.Null[%s]", res), nil
	} else if useNull {
		res, ok = sqlColumnToGoTypeMappingUseNull[sqlType]
		res = "sql." + res
	} else {
		res, ok = sqlColumnToGoTypeMapping[sqlType]
	}
	if !ok {
		return "", errors.Errorf("unsupport sql type %s", sqlType)
	}

	return res, nil
}

type ModelSpec struct {
	ModelStructName string
	Fields          []*ModelFieldSpec
}

func (m *ModelSpec) GetImports() []string {
	set := collection.NewSet[string]()
	for _, field := range m.Fields {
		set.Adds(field.Imports...)
	}

	return set.Values()
}

type ModelFieldSpec struct {
	Name    string
	Type    string
	Tags    []KVPair
	Imports []string
}

type KVPair struct {
	Key string
	Val string
}

func (m *ModelFieldSpec) AddImport(path string) {
	m.Imports = append(m.Imports, path)
}

func (m *ModelFieldSpec) AddTag(key, value string) {
	m.Tags = append(m.Tags, KVPair{
		Key: key,
		Val: value,
	})
}

func (m *ModelFieldSpec) Tag() string {
	builder := strings.Builder{}
	builder.Grow(32)
	builder.WriteString("`")
	i := 0
	for _, tag := range m.Tags {
		builder.WriteString(tag.Key)
		builder.WriteString(":")
		builder.WriteString("\"")
		builder.WriteString(tag.Val)
		builder.WriteString("\"")
		i++
		if i != len(m.Tags) {
			builder.WriteString(" ")
		}
	}
	builder.WriteString("`")

	return builder.String()
}

type ModelGenOptions struct {
	TablePrefix string   // 表名包含的前缀, 比如t_user中前缀为t
	UseNull     bool     // 当字段为nullable时, 使用使用sql.NullValue来作为字段类型
	RepoSuffix  string   // 生成的DAO对象的名称后缀
	ModelSuffix string   // 生成的model结构体名称后缀
	TagKeys     []string // 需要附加的结构体tag
}

func GenerateGoModelStructList(specList []*dbparser.TableSpec, options *command.CommandOptions, modelGenOptions *ModelGenOptions) error {
	srcBuilder := strings.Builder{}
	srcBuilder.Grow(4 << 10)
	importSet := collection.NewSet[string]()
	importSet.Add("github.com/mangohow/vulcan/annotation")
	for _, spec := range specList {
		modelDetails, err := GenerateGoModelStruct(spec, modelGenOptions)
		if err != nil {
			return err
		}
		srcBuilder.WriteString(modelDetails.Source)
		srcBuilder.WriteByte('\n')
		importSet.Adds(modelDetails.Imports...)
	}

	modelOutputPath, err := filepath.Abs(options.ModelOutputPath)
	if err != nil {
		return errors.Wrapf(err, "get model abs path failed")
	}
	var (
		modelFileName = modelOutputPath
		modelFilePath = modelOutputPath
		packageName   string
	)

	if !strings.HasSuffix(modelFileName, ".go") {
		base := filepath.Base(options.File)
		index := strings.LastIndex(base, ".")
		if index != -1 {
			base = base[:index]
		}
		modelFileName = filepath.Join(modelFilePath, strings.ToLower(base)+"_gen.go")
	} else {
		modelFilePath = filepath.Dir(modelFilePath)
	}
	exists, err := utils.IsDirExists(modelFilePath)
	if err != nil {
		return errors.Wrapf(err, "stat %s error", modelFilePath)
	}
	if !exists {
		if err = os.MkdirAll(modelFilePath, 0644); err != nil {
			return errors.Wrapf(err, "mkdir %s error", modelFilePath)
		}
	} else {
		// 获取包名
		packageName, err = utils.GetPackageNameByDir(modelFilePath)
		if err != nil {
			return errors.Wrapf(err, "get model package name failed")
		}
	}

	options.File = modelFileName
	// 写入生成的代码
	file, err := os.OpenFile(modelFileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrapf(err, "open file %s failed", modelFileName)
	}
	defer file.Close()

	fmt.Fprintf(file, fileHeaderComment)
	fmt.Fprintf(file, fmt.Sprintf("package %s\n\n", packageName))
	if importSet.Len() > 0 {
		fmt.Fprintf(file, "import (\n")
		importSet.ForEach(func(v string) {
			fmt.Fprintf(file, fmt.Sprintf("\t%q\n", v))
		})
		fmt.Fprintf(file, ")\n\n")
	}
	fmt.Fprintf(file, srcBuilder.String())

	log.Infof("Generate go file %s!", modelFileName)
	// 格式化代码
	output, err := exec.Command("go", "fmt", modelFileName).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "format source file %s failed, error info: %s", modelFileName, output)
	}

	return nil
}

type ModelStructDetails struct {
	Source  string
	Imports []string
}

func GenerateGoModelStruct(spec *dbparser.TableSpec, modelGenOptions *ModelGenOptions) (*ModelStructDetails, error) {
	// 先将数据库的表描述转换为go模型结构体描述
	modelSpec, err := convertToModelSpec(spec, modelGenOptions)
	if err != nil {
		return nil, err
	}
	// 生成结构体代码
	builder := strings.Builder{}
	builder.Grow(256)
	builder.WriteString("type ")
	builder.WriteString(modelSpec.ModelStructName)
	builder.WriteString(" struct {\n")
	// 对genFuncs校验
	if err := validateGenFuncList(spec.GenFuncList); err != nil {
		return nil, err
	}
	// 写入TableProperty
	if len(spec.GenFuncList) > 0 {
		builder.WriteString(fmt.Sprintf("\t%s `tableName:\"%s\" gen:\"%s\"`\n\n", tablePropertyTypeName, spec.TableName, strings.Join(spec.GenFuncList, "|")))
	} else {
		builder.WriteString(fmt.Sprintf("\t%s `tableName:\"%s\"`\n\n", tablePropertyTypeName, spec.TableName))
	}

	for _, field := range modelSpec.Fields {
		builder.WriteString("\t")
		builder.WriteString(field.Name)
		builder.WriteString(" ")
		builder.WriteString(field.Type)
		builder.WriteString(" ")
		builder.WriteString(field.Tag())
		builder.WriteString("\n")
	}
	builder.WriteString("}\n")

	return &ModelStructDetails{
		//Type:    typeSpec,
		Source:  builder.String(),
		Imports: modelSpec.GetImports(),
	}, nil
}

func convertToModelSpec(spec *dbparser.TableSpec, modelGenOptions *ModelGenOptions) (*ModelSpec, error) {
	tablePrefix := strings.Trim(modelGenOptions.TablePrefix, "_")
	parts := strings.Split(spec.TableName, "_")
	if len(parts) > 0 && parts[0] == tablePrefix {
		parts = parts[1:]
	}
	modelSpec := &ModelSpec{
		ModelStructName: stringutils.ToPascalCaseByList(parts) + stringutils.UpperFirstLittle(modelGenOptions.ModelSuffix),
	}

	for _, col := range spec.Columns {
		modelFieldSpec := &ModelFieldSpec{}
		goType, err := getGoTypeFromSqlType(col.Type, modelGenOptions.UseNull && !col.NotNull && !col.IsPrimaryKey)
		if err != nil {
			return nil, err
		}
		if strings.Contains(goType, "sql.") {
			modelFieldSpec.AddImport("database/sql")
		}
		if strings.Contains(goType, "time.") {
			modelFieldSpec.AddImport("time")
		}

		modelFieldSpec.Type = goType
		modelFieldSpec.Name = stringutils.ToPascalCase(col.Name)

		dbTagVal := col.Name
		if col.IsPrimaryKey {
			dbTagVal += ("," + primaryTag)
		}
		if col.IsAutoIncrement {
			dbTagVal += ("," + autoincrementTag)
		}
		modelFieldSpec.AddTag("db", dbTagVal)

		for _, key := range modelGenOptions.TagKeys {
			if key != "" {
				modelFieldSpec.AddTag(key, stringutils.LowerFirstLittle(modelFieldSpec.Name))
			}
		}

		modelSpec.Fields = append(modelSpec.Fields, modelFieldSpec)
	}

	return modelSpec, nil
}

// TODO
func validateGenFuncList(fns []string) error {
	return nil
}
