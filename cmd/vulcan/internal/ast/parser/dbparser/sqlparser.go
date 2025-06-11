package dbparser

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/mangohow/mangokit/tools/stream"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/errors"
	"strings"
)

func ParseSelectFields(sql string, paramSpec *types.Param) ([]string, []string, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parse sql %s failed", sql)
	}

	switch st := stmt.(type) {
	case *sqlparser.Select:
		return parseSelectFields(st, paramSpec)
	default:
		return nil, nil, nil
	}
}

func parseSelectFields(selectStmt *sqlparser.Select, paramSpec *types.Param) ([]string, []string, error) {
	var aliasedExprs []*sqlparser.AliasedExpr
	for _, selectExpr := range selectStmt.SelectExprs {
		switch expr := selectExpr.(type) {
		case *sqlparser.StarExpr:
			return parseStarExpr(paramSpec)
		case *sqlparser.AliasedExpr:
			aliasedExprs = append(aliasedExprs, expr)
		case *sqlparser.Nextval:
		}
	}

	if len(aliasedExprs) != 0 {
		return parseAliasedExprs(aliasedExprs, paramSpec)
	}

	return nil, nil, nil
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
		tagItems := strings.Split(tagStr, ",")
		dbFieldName := ""
		if len(tagItems) > 0 {
			dbFieldName = strings.Trim(tagItems[0], " ")
		}
		if dbFieldName == "-" {
			dbFieldName = ""
		}
		structFieldNames = append(structFieldNames, paramSpec.Name+"."+fieldName)
		if dbFieldName != "" {
			dbFieldNames = append(dbFieldNames, dbFieldName)
			return true
		}

		// 根据转换规则获取

		return true
	})

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

	return nil, nil, nil
}
