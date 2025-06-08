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

	// 去除\n
	for {
		idx := strings.Index(preparedSQL, "\n")
		if idx == -1 {
			break
		}

		prev := preparedSQL[:idx]
		idx++
		for ; idx < len(preparedSQL) && preparedSQL[idx] == ' '; idx++ {
		}
		preparedSQL = strings.TrimRight(prev, " ") + " " + preparedSQL[idx:]
	}

	// 去除\t
	preparedSQL = strings.ReplaceAll(preparedSQL, "\t", "")

	return &SqlParseResult{
		SQL:        preparedSQL,
		ParamsName: params,
	}
}
