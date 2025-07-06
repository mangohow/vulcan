package dbgenerator

import (
	"fmt"
	"github.com/mangohow/mangokit/tools/collection"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/dbparser"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/command"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils/stringutils"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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

	goTypeToReflectKindMapping = map[string]reflect.Kind{
		"int":   reflect.Int,
		"int8":  reflect.Int8,
		"int16": reflect.Int16,
		"int32": reflect.Int32,
		"int64": reflect.Int64,

		"uint":   reflect.Uint,
		"uint8":  reflect.Uint8,
		"uint16": reflect.Uint16,
		"uint32": reflect.Uint32,

		"float32": reflect.Float32,
		"float64": reflect.Float64,

		"string": reflect.String,
		"[]byte": reflect.Slice,

		"NullInt64":   reflect.Struct,
		"NullInt32":   reflect.Struct,
		"NullInt16":   reflect.Struct,
		"NullFloat64": reflect.Struct,
		"NullString":  reflect.Struct,
		"NullByte":    reflect.Struct,
		"NullTime":    reflect.Struct,
		"NullBool":    reflect.Struct,
	}
)

func getGoTypeFromSqlType(sqlType string, useNull bool) (string, error) {
	var (
		res string
		ok  bool
	)

	if useNull {
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
		for _, path := range field.Imports {
			set.Add(path)
		}
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

func GenerateGoModelStructList(specList []*dbparser.TableSpec, options *command.CommandOptions, modelGenOptions *ModelGenOptions) ([]*types.TypeSpec, error) {
	srcBuilder := strings.Builder{}
	srcBuilder.Grow(4 << 10)
	importSet := collection.NewSet[string]()
	importSet.Add("github.com/mangohow/vulcan")
	res := make([]*types.TypeSpec, 0, len(specList))
	for _, spec := range specList {
		modelDetails, err := GenerateGoModelStruct(spec, modelGenOptions)
		if err != nil {
			return nil, err
		}
		srcBuilder.WriteString(modelDetails.Source)
		srcBuilder.WriteByte('\n')
		for _, importPath := range modelDetails.Imports {
			importSet.Add(importPath)
		}
		res = append(res, modelDetails.Type)
	}

	modelOutputPath, err := filepath.Abs(options.ModelOutputPath)
	if err != nil {
		return nil, errors.Wrapf(err, "get model abs path failed")
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
		return nil, errors.Wrapf(err, "stat %s error", modelFilePath)
	}
	if !exists {
		if err = os.MkdirAll(modelFilePath, 0644); err != nil {
			return nil, errors.Wrapf(err, "mkdir %s error", modelFilePath)
		}
	} else {
		// 获取包名
		packageName, err = utils.GetPackageNameByDir(modelFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "get model package name failed")
		}
	}

	// 写入生成的代码
	file, err := os.OpenFile(modelFileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, "open file %s failed", modelFileName)
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

	// 格式化代码
	exec.Command("go", "fmt", modelFileName).Run()

	return res, nil
}

func GenerateTypeSpecs(modelSpec *ModelSpec) (*types.TypeSpec, error) {
	typeSpec := &types.TypeSpec{
		Name: modelSpec.ModelStructName,
		Kind: reflect.Struct,
		Fields: []*types.Param{
			{
				Type: types.TypeSpec{
					Name: "TableProperty",
					Package: &types.PackageInfo{
						PackageName: corePackageName,
						PackagePath: corePackagePath,
					},
					Tag:  reflect.StructTag(modelSpec.Fields[0].Tag()),
					Kind: reflect.Struct,
				},
			},
		},
	}

	for _, field := range modelSpec.Fields {
		sf := &types.Param{
			Name: field.Name,
			Type: types.TypeSpec{
				Name:    field.Type,
				Package: nil,
				Tag:     reflect.StructTag(field.Tag()),
				Kind:    0,
			},
		}
		if field.Type == "time.Time" {
			sf.Type.Kind = reflect.Struct
			sf.Type.Package = &types.PackageInfo{
				PackageName: "time",
				PackagePath: "time",
			}
		} else if strings.HasPrefix(field.Type, "sql.") {
			sf.Type.Kind = reflect.Struct
			sf.Type.Package = &types.PackageInfo{
				PackageName: "sql",
			}
		} else {
			sf.Type.Kind = goTypeToReflectKindMapping[field.Type]
		}

		typeSpec.Fields = append(typeSpec.Fields, sf)
	}

	return typeSpec, nil
}

type ModelStructDetails struct {
	Type    *types.TypeSpec
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
	hasPrimaryKey := false
	for _, col := range spec.Columns {
		if col.IsPrimaryKey {
			hasPrimaryKey = true
			break
		}
	}
	genFuncs := []string{"Add", "BatchAdd", "SelectPage(|true)"}
	if hasPrimaryKey {
		genFuncs = append(genFuncs, "DeleteById", "GetById", "SelectListByIds", "UpdateById(|true)")
	}
	// 写入TableProperty
	builder.WriteString(fmt.Sprintf("\tvulcan.TableProperty `tableName:\"%s\" gen:\"%s\"`\n\n", spec.TableName, strings.Join(genFuncs, ",")))
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

	typeSpec, err := GenerateTypeSpecs(modelSpec)
	if err != nil {
		return nil, err
	}

	return &ModelStructDetails{
		Type:    typeSpec,
		Source:  builder.String(),
		Imports: modelSpec.GetImports(),
	}, nil
}

func convertToModelSpec(spec *dbparser.TableSpec, modelGenOptions *ModelGenOptions) (*ModelSpec, error) {
	parts := strings.Split(spec.TableName, "_")
	if len(parts) > 0 && parts[0] == modelGenOptions.TablePrefix {
		parts = parts[1:]
	}
	modelSpec := &ModelSpec{
		ModelStructName: stringutils.ToPascalCaseByList(parts) + modelGenOptions.ModelSuffix,
	}

	for _, col := range spec.Columns {
		modelFieldSpec := &ModelFieldSpec{}
		goType, err := getGoTypeFromSqlType(col.Type, modelGenOptions.UseNull && !col.NotNull && !col.IsPrimaryKey)
		if err != nil {
			return nil, err
		}
		if strings.Contains(goType, "sql.") {
			modelFieldSpec.AddImport("database/sql")
		} else if strings.Contains(goType, "time.") {
			modelFieldSpec.AddImport("time")
		}

		modelFieldSpec.Type = goType
		modelFieldSpec.Name = stringutils.ToPascalCase(col.Name)

		dbTagVal := col.Name
		if col.IsPrimaryKey {
			dbTagVal += ",pk"
		}
		if col.IsAutoIncrement {
			dbTagVal += ",auto_incr"
		}
		modelFieldSpec.AddTag("db", dbTagVal)

		for _, key := range modelGenOptions.TagKeys {
			if key != "" {
				modelFieldSpec.AddTag(key, modelFieldSpec.Name)
			}
		}

		modelSpec.Fields = append(modelSpec.Fields, modelFieldSpec)
	}

	return modelSpec, nil
}
