package dbparser

import (
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
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

	stream.ForEach(structTypeSpec.Fields, func(param *types.Param) bool {
		fieldName := param.Name
		// 先尝试从db标签中获取字段名
		tagStr := param.Type.Tag.Get("db")
		if tagStr == "" {
			return true
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
			return true
		}

		// 根据转换规则获取 TODO

		return true
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
