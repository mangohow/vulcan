package dbparser

import (
	"os"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/mangohow/gowlb/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
)

const (
	commentGenKey = "+gen:"
)

func ParseSelectFields(sql string, paramSpec *types.Param) ([]string, []string, bool, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, nil, false, errors.Wrapf(err, "parse sql %s failed", sql)
	}

	switch st := stmt.(type) {
	case *sqlparser.Select:
		return parseSelectFields(sql, st, paramSpec)
	default:
		return nil, nil, false, nil
	}
}

func parseSelectFields(sql string, selectStmt *sqlparser.Select, paramSpec *types.Param) ([]string, []string, bool, error) {
	var aliasedExprs []*sqlparser.AliasedExpr
	for _, selectExpr := range selectStmt.SelectExprs {
		switch expr := selectExpr.(type) {
		case *sqlparser.StarExpr:
			tableFields, structFields, err := parseStarExpr(paramSpec)
			if err != nil {
				return nil, nil, true, err
			}
			return tableFields, structFields, true, nil
		case *sqlparser.AliasedExpr:
			aliasedExprs = append(aliasedExprs, expr)
		case *sqlparser.Nextval:
		}
	}

	if len(aliasedExprs) != 0 {
		tableFields, structFields, err := parseAliasedExprs(aliasedExprs, paramSpec)
		if err != nil {
			return nil, nil, false, err
		}
		return tableFields, structFields, false, nil
	}

	return nil, nil, false, errors.Errorf("invalid select stmt %s", sql)
}

func parseStarExpr(paramSpec *types.Param) ([]string, []string, error) {
	structTypeSpec := getStructTypeSpec(&paramSpec.Type)
	if structTypeSpec == nil {
		return nil, nil, errors.Errorf("invalid type %s", paramSpec.Type.Name)
	}

	var (
		dbFieldNames     = make([]string, 0, len(structTypeSpec.Fields))
		structFieldNames = make([]string, 0, len(structTypeSpec.Fields))
	)

	stream.ForEach(structTypeSpec.Fields, func(param *types.Param) {
		fieldName := param.Name
		// 先尝试从db标签中获取字段名
		tagStr := param.Type.Tag.Get("db")
		if tagStr == "" {
			return
		}
		tagItems := strings.Split(tagStr, ",")
		dbFieldName := ""
		if len(tagItems) > 0 {
			dbFieldName = strings.Trim(tagItems[0], " ")
		}
		if dbFieldName == "-" {
			dbFieldName = ""
		}
		structFieldNames = append(structFieldNames, fieldName)
		if dbFieldName != "" {
			dbFieldNames = append(dbFieldNames, dbFieldName)
			return
		}

		// 根据转换规则获取 TODO
	})

	if len(structFieldNames) == 0 {
		return nil, nil, errors.Errorf("db tag is need on type %s", structTypeSpec.Name)
	}

	return dbFieldNames, structFieldNames, nil
}

func getStructTypeSpec(typeSpec *types.TypeSpec) *types.TypeSpec {
	for {
		switch {
		case typeSpec.IsSlice(), typeSpec.IsPointer():
			typeSpec = typeSpec.ValueType
		case typeSpec.IsStruct():
			return typeSpec
		default:
			return nil
		}
	}
}

func parseAliasedExprs(aliasedExprs []*sqlparser.AliasedExpr, paramSpec *types.Param) ([]string, []string, error) {
	var (
		tableFields  = make([]string, 0, len(aliasedExprs))
		structFields = make([]string, 0, len(aliasedExprs))
	)

	structTypeSpec := getStructTypeSpec(&paramSpec.Type)
	if structTypeSpec == nil {
		return nil, nil, errors.Errorf("invalid type %s", paramSpec.Type.Name)
	}

	// 解析表字段
	for _, ex := range aliasedExprs {
		tableField := sqlparser.String(ex)
		if ex.As.String() != "" {
			tableField = ex.As.String()
		}
		tableField = strings.Trim(tableField, " `")
		tableFields = append(tableFields, tableField)
	}

	tableFieldToStructFieldMap := make(map[string]string, len(structTypeSpec.Fields))
	// 解析结构体字段 TODO 根据映射方式解析
	for _, param := range structTypeSpec.Fields {
		tagStr := param.Type.Tag.Get("db")
		if tagStr == "" {
			continue
		}
		tags := strings.Split(tagStr, ",")
		tableFieldTag := strings.Trim(tags[0], " ")
		tableFieldToStructFieldMap[tableFieldTag] = param.Name
	}

	for _, tf := range tableFields {
		sf, ok := tableFieldToStructFieldMap[tf]
		if !ok {
			return nil, nil, errors.Errorf("select field %s not found in type %s, please use db tag to specify it", tf, structTypeSpec.Name)
		}
		structFields = append(structFields, sf)
	}

	return tableFields, structFields, nil
}

func ParseSqlFile(filename string) ([]*TableSpec, error) {
	sqlContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", filename)
	}

	var res []*TableSpec
	// 分割sql语句
	sqls := strings.Split(string(sqlContent), ";")
	for _, sq := range sqls {
		lines := strings.Split(sq, "\n")
		var genFuncs []string
		stream.ForEach(lines, func(line string) {
			if !strings.Contains(line, commentGenKey) {
				return
			}

			idx := strings.Index(line, commentGenKey)
			line = strings.Trim(line[idx+len(commentGenKey):], " ")
			funcList := strings.Split(line, "|")
			funcList = stream.Map(funcList, func(s string) string {
				return strings.TrimSpace(s)
			})
			genFuncs = append(genFuncs, funcList...)
		})
		sq = strings.TrimSpace(sq)
		if !strings.Contains(sq, "create table") && !strings.Contains(sq, "CREATE TABLE") {
			continue
		}
		stmt, err := sqlparser.Parse(sq)
		if err != nil {
			return nil, errors.Wrapf(err, "parse sql in file %s", filename)
		}
		createStmt, ok := stmt.(*sqlparser.CreateTable)
		if !ok {
			continue
		}

		tableSpec := ParseCreationFields(createStmt)
		tableSpec.GenFuncList = genFuncs
		res = append(res, tableSpec)
	}

	return res, nil
}

type TableSpec struct {
	TableName   string
	GenFuncList []string
	Columns     []*TableColumn
}

type TableColumn struct {
	Name            string
	Type            string
	IsPrimaryKey    bool
	IsAutoIncrement bool
	NotNull         bool
}

// 解析建表语句
func ParseCreationFields(stmt *sqlparser.CreateTable) *TableSpec {
	res := &TableSpec{
		TableName: stmt.NewName.Name.String(),
	}
	for _, def := range stmt.Columns {
		typeName := strings.ToLower(def.Type)
		items := strings.Split(typeName, " ")
		if len(items) > 1 {
			typeName = items[0]
		}
		index := strings.Index(def.Type, "(")
		if index != -1 {
			typeName = def.Type[:index]
		}
		if len(items) > 1 && items[1] == "unsigned" {
			typeName += " unsigned"
		}
		c := &TableColumn{
			Name: def.Name,
			Type: typeName,
		}

		for _, opt := range def.Options {
			switch opt.Type {
			case sqlparser.ColumnOptionPrimaryKey:
				c.IsPrimaryKey = true
			case sqlparser.ColumnOptionNotNull:
				c.NotNull = true
			case sqlparser.ColumnOptionAutoIncrement:
				c.IsAutoIncrement = true
			}
		}
		res.Columns = append(res.Columns, c)
	}

	var pk string
	for _, constraint := range stmt.Constraints {
		if constraint.Type == sqlparser.ConstraintPrimaryKey && len(constraint.Keys) > 0 {
			pk = constraint.Keys[0].String()
			break
		}
	}

	if pk == "" {
		return res
	}

	stream.ForEachB(res.Columns, func(column *TableColumn) bool {
		if column.Name == pk {
			column.IsPrimaryKey = true
			return false
		}
		return true
	})

	return res
}
