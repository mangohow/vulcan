package dbgenerator

import (
	"fmt"
	"strings"

	"github.com/mangohow/gowlb/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/log"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils"
)

var (
	AddFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) Add({{ .ModelObjName }} *{{ .ModelTypeName }}) {
	Insert("INSERT INTO {{ .TableName }} ({{ .TableFields }}) VALUES ({{ .StructFields }})")
}`

	AddBatchFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) BatchAdd({{ .EntityListName }} []*{{ .ModelTypeName }}) {
	Insert(SQL().
		Stmt("INSERT INTO {{.TableName}} ({{.TableFields}}) VALUES").
		Foreach("{{ .EntityListName }}", "{{ .EntityObjName }}", "", "", "", "({{ .StructFields }})").
		Build())
	return nil
}`
	DeleteByIdFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) DeleteById({{ .QueryKeyName }} {{ .QueryKeyType }}) {
	Delete("DELETE FROM {{ .TableName}} WHERE {{ .PrimaryKey }} = {{ .QueryKeyNameRef }}")
}`

	DeleteBatchIdsFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) DeleteBatchIds({{ .QueryListName }} []{{ .QueryKeyType }}) {
	Delete(SQL().
		Stmt("DELETE FROM {{ .TableName }} WHERE {{ .PrimaryKey }} IN").
		Foreach("{{ .QueryListName }}", "{{ .QueryKeyName }}", ", ", "(", ")", "{{ .QueryKeyNameRef }}").
		Build())
}`

	SelectByIdFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) SelectById({{ .QueryKeyName }} {{ .QueryKeyType }}) *{{ .ModelTypeName }} {
	Select("SELECT {{ .SelectColumns }} FROM {{ .TableName }} WHERE {{ .PrimaryKey }} = {{ .QueryKeyNameRef }}")
	return nil
}`

	SelectBatchIdsFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) SelectBatchIds({{ .QueryListName }} []{{ .QueryKeyType }}) []*{{ .ModelTypeName }} {
	Select(SQL().
		Stmt("SELECT {{ .SelectColumns }} FROM {{ .TableName }} WHERE {{ .PrimaryKey }} IN").
		Foreach("{{ .QueryListName }}", "{{ .QueryKeyName }}", ", ", "(", ")", "{{ .QueryKeyNameRef }}").
		Build())
	return nil
}`

	SelectAllFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) SelectAll() []*{{ .ModelTypeName }} {
	Select("SELECT {{ .TableFields }} FROM {{ .TableName }}")
}`

	SelectCountFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) SelectCount() int {
	Select("SELECT COUNT(*) FROM {{ .TableName }}")
}`

	DeleteByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) DeleteBy({{ .ModelObjName }} *{{ .ModelTypeName }}) {
	Delete("DELETE FROM {{ .TableName }} WHERE 1=1{{ .WhereQuery }}")
}`

	UpdateByIdFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) UpdateById({{ .ModelObjName }} *{{ .ModelTypeName }}) {
	Update(SQL().
		{{ if .ValidateEmpty -}}
		Stmt("UPDATE {{ .TableName }}").
		Set(%s).
		{{ else -}}
		Stmt("UPDATE {{ .TableName }} SET {{ .NoValidateSetStmt }}").
		{{ end -}}
		Stmt("WHERE {{ .PrimaryKey }} = {{ .QueryKeyNameRef }}").
		Build())
}`

	UpdateByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) {
	Update(SQL().
		{{ if .ValidateSetField }}
		Stmt("UPDATE {{ .TableName }}").
		Set(%s).
		{{ else -}}
		Stmt("UPDATE {{ .TableName }} SET {{ .NoValidateSetStmt }}").
		{{ end -}}
		{{ if .ValidateWhereField -}}
		Where(%s).
		{{ else -}}
		Stmt("WHERE 1=1{{ .WhereQuery }}").
		{{ end -}}
		Build())
}`

	SelectOneByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) *{{ .ModelTypeName }} {
	Select(SQL().
		Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }} WHERE 1=1{{ .WhereQuery }} LIMIT 1).
		Build())
	return nil
}`
	SelectOneByFuncTemplateArgsNullable = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) *{{ .ModelTypeName }} {
	Select(SQL().Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }}).
		Where(%s).
		Stmt("Limit 1").
		Build())
	return nil
}`

	SelectListByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) []*{{ .ModelTypeName }} {
	Select(SQL().
		Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }} WHERE 1=1{{ .WhereQuery }}).
		Build())
	return nil
}`

	SelectListByFuncTemplateArgsNullable = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) []*{{ .ModelTypeName }} {
	Select(SQL().
		Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }}).
		Where(%s).
		Build())
	return nil
}`

	SelectCountByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) int {
	Select(SQL().
		Stmt(SELECT COUNT(*) FROM {{ .TableName }} WHERE 1=1{{ .WhereQuery }}).
		Build())
	return 0
}`

	SelectCountByFuncTemplateArgsNullable = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}({{ .ModelObjName }} *{{ .ModelTypeName }}) int {
	Select(SQL().
		Stmt(SELECT COUNT(*) FROM {{ .TableName }}).
		Where(%s).
		Build())
	return 0
}`

	SelectPageByFuncTemplate = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}(page vulcan.Page, {{ .ModelObjName }} *{{ .ModelTypeName }}) []*{{ .ModelTypeName }} {
	Select(SQL().
		Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }} WHERE 1=1{{ .WhereQuery }}).
		Build())
	return nil
}`

	SelectPageByFuncTemplateArgsNullable = `func ({{ .ReceiverName }} *{{ .MapperName }}) {{.FuncName}}(page vulcan.Page, {{ .ModelObjName }} *{{ .ModelTypeName }}) []*{{ .ModelTypeName }} {
	Select(SQL().
		Stmt(SELECT {{ .SelectColumns }} FROM {{ .TableName }}).
		Where(%s)
		Build())
	return nil
}`
)

type CRUDGenFunc func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error)

var (
	getTableFields = func(fields []*types.ModelField, ignorePrimary bool) string {
		fieldList := make([]string, 0, len(fields))
		for _, field := range fields {
			if ignorePrimary && field.IsPrimaryKey && field.IsAutoIncrement {
				continue
			}
			fieldList = append(fieldList, "`"+field.ColumnName+"`")
		}
		return strings.Join(fieldList, ", ")
	}

	getAddFields = func(objName string, fields []*types.ModelField, ignorePrimary bool) string {
		structList := make([]string, 0, len(fields))
		for _, field := range fields {
			if ignorePrimary && field.IsPrimaryKey && field.IsAutoIncrement {
				continue
			}
			structList = append(structList, fmt.Sprintf("#{%s.%s}", objName, field.Name))
		}
		return strings.Join(structList, ", ")
	}

	getStructFieldName = func(column string, fields []*types.ModelField) string {
		for _, field := range fields {
			if field.ColumnName == column {
				return field.Name
			}
		}

		log.Fatalf("column %s not found", column)
		return ""
	}

	getStructField = func(column string, fields []*types.ModelField) *types.ModelField {
		for _, field := range fields {
			if field.ColumnName == column {
				return field
			}
		}

		log.Fatalf("column %s not found", column)
		return nil
	}

	validateNullableFields = func(columns []string, fields []*types.ModelField) error {
		for _, field := range fields {
			if !utils.Contains(columns, field.ColumnName) {
				continue
			}

			if !types.IsNullableType(field.Type) {
				return errors.Errorf("field %s is not a nullable type", field.Name)
			}
		}

		return nil
	}

	getNullableTypeValidator = func(objName, fieldName, typeName string) string {
		fieldRef := objName + "." + fieldName
		if typeName == "sql.RawBytes" {
			return fmt.Sprintf("len(%s) != 0", fieldRef)
		}

		return fieldRef + "." + "Valid"
	}

	getNullableTypeValueName = func(objName, fieldName, typeName string) string {
		fieldRef := objName + "." + fieldName
		if strings.HasPrefix(typeName, "sql.Null[") {
			return fieldRef + "." + "V"
		}

		if typeName == "sql.RawBytes" {
			return fieldRef
		}

		_, name, found := strings.Cut(typeName, "sql.Null")
		if !found {
			log.Fatalf("invalid type %s", typeName)
		}

		return fieldRef + "." + name
	}

	getTypeValueName = func(objName, fieldName, typeName string) string {
		if types.IsNullableType(typeName) {
			return getNullableTypeValueName(objName, fieldName, typeName)
		}

		return objName + "." + fieldName
	}

	genWhereQuery = func(whereColumns []types.Pair[string, string], fields []*types.ModelField, objName string) string {
		builder := strings.Builder{}
		for _, p := range whereColumns {
			field := getStructField(p.Val, fields)
			builder.WriteString(fmt.Sprintf(" %s %s=#{%s}", p.Key, p.Val, getTypeValueName(objName, field.Name, field.Type)))
		}
		return builder.String()
	}

	genIfOfWhereAnnotation = func(whereColumns []types.Pair[string, string], fields []*types.ModelField, objName string) string {
		builder := strings.Builder{}
		for i, p := range whereColumns {
			field := getStructField(p.Val, fields)
			validator := getNullableTypeValidator(objName, field.Name, field.Type)
			valueName := getNullableTypeValueName(objName, field.Name, field.Type)
			if i > 0 {
				builder.WriteString(".\n\t\t\t")
			}
			builder.WriteString(fmt.Sprintf("If(%s, \"%s %s=#{%s}\")", validator, p.Key, p.Val, valueName))
		}

		return builder.String()
	}

	selectXXXFunc = func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions, tmpl, nullableTmpl, genFuncName, tmplName string) (string, error) {
		data := &SelectTemplateOptions{
			CommonOptions: options,
			SelectColumns: strings.Join(stream.Map(funcSpec.SelectColumnNames, func(s string) string {
				return "`" + s + "`"
			}), ","),
			FuncName: genFuncName,
		}
		funcTmpl := tmpl
		if !funcSpec.SelectValidateEmpty {
			data.WhereQuery = genWhereQuery(funcSpec.WhereColumnNames, spec.ModelFields, options.ModelObjName)
		} else {
			funcTmpl = nullableTmpl
			if err := validateNullableFields(stream.Map(funcSpec.WhereColumnNames, func(s types.Pair[string, string]) string {
				return s.Val
			}), spec.ModelFields); err != nil {
				return "", err
			}
			annotations := genIfOfWhereAnnotation(funcSpec.WhereColumnNames, spec.ModelFields, options.ModelObjName)
			funcTmpl = fmt.Sprintf(funcTmpl, annotations)
		}

		return utils.ExecuteTemplate(tmplName, funcTmpl, data)
	}

	genIfOfSetAnnotation = func(setColumns []string, fields []*types.ModelField, objName string) string {
		args := make([]types.Pair[string, string], 0, len(setColumns))
		for _, column := range setColumns {
			field := getStructField(column, fields)
			args = append(args, types.Pair[string, string]{
				Key: getNullableTypeValidator(objName, field.Name, field.Type),
				Val: getNullableTypeValueName(objName, field.Name, field.Type),
			})
		}
		builder := strings.Builder{}
		for i := 0; i < len(args); i++ {
			if i > 0 {
				builder.WriteString(".\n\t\t\t")
			}
			builder.WriteString(fmt.Sprintf("If(%s, %s)", args[i].Key, args[i].Val))
		}

		return builder.String()
	}

	// TODO
	crudGenFuncMapping = map[string]CRUDGenFunc{
		"Add": func(modelSpec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			tableFields := getTableFields(modelSpec.ModelFields, true)
			structFields := getAddFields(options.ModelObjName, modelSpec.ModelFields, true)
			data := &AddFuncTemplateOptions{
				CommonOptions: options,
				TableFields:   tableFields,
				StructFields:  structFields,
			}

			return utils.ExecuteTemplate("Add", AddFuncTemplate, data)
		},
		"AddBatch": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			tableFields := getTableFields(spec.ModelFields, true)
			entityListName := options.ModelObjName + "List"
			entityObjName := options.ModelObjName
			structFields := getAddFields(entityObjName, spec.ModelFields, true)
			data := &AddBatchTemplateOptions{
				CommonOptions:  options,
				EntityListName: entityListName,
				EntityObjName:  entityObjName,
				TableFields:    tableFields,
				StructFields:   structFields,
			}
			return utils.ExecuteTemplate("AddBatch", AddBatchFuncTemplate, data)
		},
		"DeleteById": func(modelSpec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &DeleteByIdTemplateOptions{
				CommonOptions:   options,
				QueryKeyName:    "id",
				QueryKeyType:    modelSpec.PrimaryKey.Type,
				QueryKeyNameRef: "#{id}",
			}
			return utils.ExecuteTemplate("DeleteById", DeleteByIdFuncTemplate, data)
		},
		"DeleteBatchIds": func(modelSpec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &DeleteBatchIdsTemplateOptions{
				CommonOptions:   options,
				QueryListName:   "ids",
				QueryKeyType:    modelSpec.PrimaryKey.Type,
				QueryKeyName:    "id",
				QueryKeyNameRef: "#{id}",
			}
			return utils.ExecuteTemplate("DeleteBatchIds", DeleteBatchIdsFuncTemplate, data)
		},
		"SelectById": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &SelectByIdTemplateOptions{
				CommonOptions:   options,
				QueryKeyName:    "id",
				QueryKeyType:    spec.PrimaryKey.Type,
				SelectColumns:   getTableFields(spec.ModelFields, false),
				QueryKeyNameRef: "#{id}",
			}
			return utils.ExecuteTemplate("SelectById", SelectByIdFuncTemplate, data)
		},
		"SelectBatchIds": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &SelectBatchIdsTemplateOptions{
				CommonOptions:   options,
				QueryListName:   "ids",
				QueryKeyName:    "id",
				QueryKeyType:    spec.PrimaryKey.Type,
				SelectColumns:   getTableFields(spec.ModelFields, false),
				QueryKeyNameRef: "#{id}",
			}
			return utils.ExecuteTemplate("SelectBatchIds", SelectBatchIdsFuncTemplate, data)
		},
		"SelectAll": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &SelectAllTemplateOptions{
				CommonOptions: options,
				TableFields:   getTableFields(spec.ModelFields, false),
			}
			return utils.ExecuteTemplate("SelectAll", SelectAllFuncTemplate, data)
		},
		"SelectCount": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			return utils.ExecuteTemplate("SelectCount", SelectCountFuncTemplate, options)
		},
		"DeleteBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := DeleteByTemplateOptions{
				CommonOptions: options,
				WhereQuery:    genWhereQuery(funcSpec.WhereColumnNames, spec.ModelFields, options.ModelObjName),
			}

			return utils.ExecuteTemplate("DeleteBy", DeleteByFuncTemplate, data)
		},
		"UpdateById": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) { // TODO
			data := &UpdateByIdTemplateOptions{
				CommonOptions:   options,
				ValidateEmpty:   funcSpec.SetValidateEmpty,
				QueryKeyNameRef: fmt.Sprintf("#{%s.%s}", options.ModelObjName, spec.PrimaryKey.Name),
			}
			tmpl := UpdateByIdFuncTemplate
			if !funcSpec.SetValidateEmpty {
				data.NoValidateSetStmt = strings.Join(stream.Map(funcSpec.SelectColumnNames, func(column string) string {
					return fmt.Sprintf("%s=#{%s.%s}", column, options.ModelObjName, getStructFieldName(column, spec.ModelFields))
				}), ",")
			} else {
				if err := validateNullableFields(funcSpec.SelectColumnNames, spec.ModelFields); err != nil {
					return "", err
				}
				tmpl = fmt.Sprintf(tmpl, genIfOfSetAnnotation(funcSpec.SelectColumnNames, spec.ModelFields, options.ModelObjName))
			}

			return utils.ExecuteTemplate("UpdateById", tmpl, data)
		},
		"UpdateBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			data := &UpdateByTemplateOptions{
				CommonOptions:      options,
				ValidateSetField:   funcSpec.SetValidateEmpty,
				ValidateWhereField: funcSpec.SelectValidateEmpty,
				FuncName:           funcSpec.FuncName,
			}
			var (
				tmpl            = UpdateByFuncTemplate
				setAnnotation   string
				whereAnnotation string
			)
			if !funcSpec.SetValidateEmpty {
				data.NoValidateSetStmt = strings.Join(stream.Map(funcSpec.SelectColumnNames, func(column string) string {
					return fmt.Sprintf("%s=#{%s.%s}", column, options.ModelObjName, getStructFieldName(column, spec.ModelFields))
				}), ",")
			} else {
				if err := validateNullableFields(funcSpec.SelectColumnNames, spec.ModelFields); err != nil {
					return "", err
				}
				setAnnotation = genIfOfSetAnnotation(funcSpec.SelectColumnNames, spec.ModelFields, options.ModelObjName)
			}

			if !funcSpec.SelectValidateEmpty {
				data.WhereQuery = genWhereQuery(funcSpec.WhereColumnNames, spec.ModelFields, options.ModelObjName)
			} else {
				if err := validateNullableFields(stream.Map(funcSpec.WhereColumnNames, func(s types.Pair[string, string]) string {
					return s.Val
				}), spec.ModelFields); err != nil {
					return "", err
				}
				whereAnnotation = genIfOfWhereAnnotation(funcSpec.WhereColumnNames, spec.ModelFields, options.ModelObjName)
			}
			tmpl = fmt.Sprintf(tmpl, setAnnotation, whereAnnotation)

			return utils.ExecuteTemplate("UpdateBy", tmpl, data)
		},
		"SelectOneBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			return selectXXXFunc(spec, funcSpec, options, SelectOneByFuncTemplate, SelectOneByFuncTemplateArgsNullable, funcSpec.FuncName, "SelectOneBy")
		},
		"SelectListBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			return selectXXXFunc(spec, funcSpec, options, SelectListByFuncTemplate, SelectListByFuncTemplateArgsNullable, funcSpec.FuncName, "SelectListBy")
		},
		"SelectCountBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			return selectXXXFunc(spec, funcSpec, options, SelectCountByFuncTemplate, SelectCountByFuncTemplateArgsNullable, funcSpec.FuncName, "SelectCountBy")
		},
		"SelectPageBy": func(spec *types.ModelSpec, funcSpec *types.GenFuncSpec, options *CommonOptions) (string, error) {
			return selectXXXFunc(spec, funcSpec, options, SelectPageByFuncTemplate, SelectPageByFuncTemplateArgsNullable, funcSpec.FuncName, "SelectPageBy")
		},
	}
)

type CommonOptions struct {
	MapperName    string // 接收器变量名
	ReceiverName  string // mapper结构体名
	ModelObjName  string // 结构体模型名
	ModelTypeName string // 结构体模型类型名, 如果跟mapper不在同一个包, 则携带包名
	TableName     string // 数据库表名
	PrimaryKey    string // 表主键字段名
}

type AddFuncTemplateOptions struct {
	*CommonOptions
	TableFields  string // 表所有字段
	StructFields string // 所有结构体字段
}

type AddBatchTemplateOptions struct {
	*CommonOptions
	EntityListName string
	EntityObjName  string
	TableFields    string
	StructFields   string
}

type DeleteByIdTemplateOptions struct {
	*CommonOptions
	QueryKeyName    string
	QueryKeyType    string
	QueryKeyNameRef string
}

type DeleteBatchIdsTemplateOptions struct {
	*CommonOptions
	QueryListName   string
	QueryKeyType    string
	QueryKeyName    string
	QueryKeyNameRef string
}

type SelectByIdTemplateOptions struct {
	*CommonOptions
	QueryKeyName    string
	QueryKeyType    string
	SelectColumns   string
	QueryKeyNameRef string
}

type SelectBatchIdsTemplateOptions struct {
	*CommonOptions
	QueryListName   string
	QueryKeyName    string
	QueryKeyType    string
	SelectColumns   string
	QueryKeyNameRef string
}

type SelectAllTemplateOptions struct {
	*CommonOptions
	TableFields string
}

type DeleteByTemplateOptions struct {
	*CommonOptions
	WhereQuery string
}

type UpdateByIdTemplateOptions struct {
	*CommonOptions
	ValidateEmpty     bool
	NoValidateSetStmt string
	QueryKeyNameRef   string
}

type UpdateByTemplateOptions struct {
	*CommonOptions
	ValidateSetField   bool
	ValidateWhereField bool
	NoValidateSetStmt  string
	WhereQuery         string
	FuncName           string
}

type SelectTemplateOptions struct {
	*CommonOptions
	WhereQuery    string
	SelectColumns string
	FuncName      string
}
