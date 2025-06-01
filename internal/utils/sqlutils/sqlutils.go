package sqlutils

import (
	"regexp"
	"strings"
)

type SqlParseResult struct {
	SQL        string
	ParamsName []string
}

var sqlParamRegex = regexp.MustCompile(`#\{([\w\.]+)\}`)

func ParseSQLStmt(sql string) *SqlParseResult {
	matches := sqlParamRegex.FindAllStringSubmatch(sql, -1)
	// 提取参数名
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			params = append(params, match[1])
		}
	}

	// 替换占位符
	preparedSQL := sqlParamRegex.ReplaceAllString(sql, "?")

	// 去除\n\r
	preparedSQL = strings.ReplaceAll(preparedSQL, "\n\r", " ")

	return &SqlParseResult{
		SQL:        preparedSQL,
		ParamsName: params,
	}
}
