package types

import (
	"fmt"
	"github.com/mangohow/vulcan/cmd/vulcan/internal/utils/stringutils"
	"go/ast"
	"io"
	"regexp"
	"strings"
)

const (
	SQLInsertFunc = "Insert"
	SQLDeleteFunc = "Delete"
	SQLUpdateFunc = "Update"
	SQLSelectFunc = "Select"
)

const (
	AnnotationPackageName = "github.com/mangohow/vulcan/annotation"
)

var SQLAnnotationFuncs = []string{
	SQLInsertFunc,
	SQLDeleteFunc,
	SQLUpdateFunc,
	SQLSelectFunc,
}

const (
	SQLOperateFuncSQL       = "SQL"
	SQLOperateFuncIf        = "If"
	SQLOperateFuncStmt      = "Stmt"
	SQLOperateFuncWhere     = "Where"
	SQLOperateFuncSet       = "Set"
	SQLOperateFuncCHOOSE    = "Choose"
	SQLOperateFuncWhen      = "When"
	SQLOperateFuncOtherwise = "Otherwise"
	SQLOperateFuncForeach   = "Foreach"
	SQLOperateFuncBuild     = "Build"
)

var (
	SQLOperateNames = []string{
		SQLOperateFuncSQL,
		SQLOperateFuncIf,
		SQLOperateFuncStmt,
		SQLOperateFuncWhere,
		SQLOperateFuncSet,
		SQLOperateFuncCHOOSE,
		SQLOperateFuncWhen,
		SQLOperateFuncOtherwise,
		SQLOperateFuncForeach,
		SQLOperateFuncBuild,
	}
)

type SqlType int

const (
	SqlTypeUnsupported SqlType = iota
	SqlTypeInsert
	SqlTypeDelete
	SqlTypeUpdate
	SqlTypeSelect
)

func (s SqlType) String() string {
	switch s {
	case SqlTypeInsert:
		return "INSERT"
	case SqlTypeDelete:
		return "DELETE"
	case SqlTypeUpdate:
		return "UPDATE"
	case SqlTypeSelect:
		return "SELECT"
	default:
		return fmt.Sprintf("unsupported Sql")
	}
}

func ToSqlType(s string) SqlType {
	switch strings.ToUpper(s) {
	case "INSERT":
		return SqlTypeInsert
	case "DELETE":
		return SqlTypeDelete
	case "UPDATE":
		return SqlTypeUpdate
	case "SELECT":
		return SqlTypeSelect
	default:
		return SqlTypeUnsupported
	}
}

var re = regexp.MustCompile(`#\{([^}]*)\}`)

type SQL interface {
	sqlDoNotCall()
}

type EmptySQL interface {
	SQL
}

type EmptySQLImpl struct {
	EmptySQL
}

func NewEmptySQL() *EmptySQLImpl {
	return &EmptySQLImpl{}
}

type Cond interface {
	condDoNotCall()
}

type SimpleStmt struct {
	SQL
	Sql  string
	Args []string
}

func NewSimpleStmt(stmt string) *SimpleStmt {
	stmt = strings.Trim(stmt, "`\"")
	sql, args := parseSqlArgs(stmt)
	return &SimpleStmt{Sql: sql, Args: args}
}

type WhereStmt struct {
	SQL
	Cond Cond
}

func NewWhereStmt(cond Cond) *WhereStmt {
	switch stmt := cond.(type) {
	case *IfStmt:
		stmt.Sql = strings.TrimRight(stmt.Sql, " ")
	case *IfChainStmt:
		for _, s := range stmt.Stmts {
			s.Sql = strings.TrimRight(s.Sql, " ")
		}
	case *ChooseStmt:
		for _, s := range stmt.Whens {
			s.Sql = strings.TrimRight(s.Sql, " ")
		}
		stmt.Otherwise = strings.TrimRight(stmt.Otherwise, " ")
	}
	return &WhereStmt{Cond: cond}
}

type SetStmt struct {
	SQL
	Cond Cond
}

func NewSetStmt(cond Cond) *SetStmt {
	switch stmt := cond.(type) {
	case *IfStmt:
		stmt.Sql = strings.TrimRight(stmt.Sql, " ")
	case *IfChainStmt:
		for _, s := range stmt.Stmts {
			s.Sql = strings.TrimRight(s.Sql, " ")
		}
	case *ChooseStmt:
		for _, s := range stmt.Whens {
			s.Sql = strings.TrimRight(s.Sql, " ")
		}
		stmt.Otherwise = strings.TrimRight(stmt.Otherwise, " ")
	}
	return &SetStmt{Cond: cond}
}

type IfStmt struct {
	SQL
	Cond
	CondExpr *ast.BinaryExpr
	Sql      string
	Args     []string
}

func NewIfStmt(expr *ast.BinaryExpr, sql string) *IfStmt {
	sql = strings.Trim(sql, "`\"")

	stmt := &IfStmt{
		CondExpr: expr,
	}
	s, args := parseSqlArgs(sql)
	stmt.Sql = s
	stmt.Args = args

	return stmt
}

func parseSqlArgs(sql string) (string, []string) {
	var args []string
	// 提取所有匹配项
	matches := re.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		if len(match) > 1 {
			args = append(args, match[1])
		}
	}

	// 替换所有 #{...} 为 ?
	sql = re.ReplaceAllString(sql, "?")
	return stringutils.TrimTrailingRedundantSpaces(sql), args
}

type IfChainStmt struct {
	SQL
	Cond
	Stmts []*IfStmt
}

func NewIfChainStmt(stmts []*IfStmt) *IfChainStmt {
	return &IfChainStmt{
		Stmts: stmts,
	}
}

type WhenStmt = IfStmt

func NewWhenStmt(expr *ast.BinaryExpr, sql string) *WhenStmt {
	return NewIfStmt(expr, sql)
}

type ChooseStmt struct {
	SQL
	Cond
	Whens         []*WhenStmt
	Otherwise     string
	OtherwiseArgs []string
}

func NewChooseStmt(stmt []*WhenStmt, otherwise string) *ChooseStmt {
	res := &ChooseStmt{
		Whens: stmt,
	}
	if otherwise != "" {
		res.Otherwise, res.OtherwiseArgs = parseSqlArgs(otherwise)

	}

	return res
}

type ForeachStmt struct {
	SQL
	CollectionName string
	ItemName       string
	Separator      string
	Open           string
	Close          string
	Sql            string

	ItemType string
	Args     []string
}

func NewForeachStmt(collectionName, itemName, separator, open, close, sql, itemType string) *ForeachStmt {
	sql = strings.Trim(sql, "`\"")
	sq, args := parseSqlArgs(sql)
	sq = strings.Trim(sq, " ")
	return &ForeachStmt{
		CollectionName: strings.Trim(collectionName, "`\""),
		ItemName:       strings.Trim(itemName, "`\""),
		Separator:      strings.Trim(separator, "`\""),
		Open:           strings.Trim(open, "`\""),
		Close:          strings.Trim(close, "`\""),
		Sql:            sq,
		ItemType:       itemType, // TODO
		Args:           args,
	}
}

type RawSQL interface {
	SQL
	Stmt() string
}

func NewRawSQL(sql string) RawSQL {
	return rawSql(sql)
}

type rawSql string

func (r rawSql) sqlDoNotCall() {

}

func (r rawSql) Stmt() string {
	return string(r)
}

func PrintSQLHelper(s SQL, writer io.Writer) {
	var cond Cond

loop:
	for {
		switch v := s.(type) {
		case EmptySQLImpl:
			fmt.Fprintf(writer, "[Empty SQL]\n")
		case *SimpleStmt:
			fmt.Fprintf(writer, "[SQL Stmt] %s\n", v.Sql)
		case rawSql:
			fmt.Fprintf(writer, "[Static Raw SQL] %s\n", v)
		case *WhereStmt:
			fmt.Fprintf(writer, "[Where Stmt] ")
			cond = v.Cond
		case *SetStmt:
			fmt.Fprintf(writer, "[Set Stmt]")
			cond = v.Cond
		case *ForeachStmt:
			fmt.Fprintf(writer, "[Foreach Stmt] %s %s %s %s %s %s\n", v.CollectionName, v.ItemName, v.Separator, v.Open, v.Close, v.Sql)
		case *ChooseStmt:
			fmt.Fprintf(writer, "[Choose Stmt] ")
			for _, w := range v.Whens {
				fmt.Fprintf(writer, "When: %s ", w.Sql)
			}
			if v.Otherwise != "" {
				fmt.Fprintf(writer, "Otherwise: %s", v.Otherwise)
			}
			fmt.Fprintf(writer, "\n")
			break loop
		case *IfStmt:
			fmt.Fprintf(writer, "[If Stmt] %s\n", v.Sql)
		case *IfChainStmt:
			fmt.Fprintf(writer, "[If Stmt] ")
			for _, i := range v.Stmts {
				fmt.Fprintf(writer, "If: %s ", i.Sql)
			}
			fmt.Fprintf(writer, "\n")
			break loop
		}

		if cond == nil {
			return
		}

		switch v := cond.(type) {
		case *IfChainStmt:
			s = v
		case *ChooseStmt:
			s = v
		default:
			break loop
		}
	}

}
