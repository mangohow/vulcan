package vulcan

import "strings"

type SqlBuilder struct {
	b         strings.Builder
	whereStmt []string
	setStmt   []string
	args      []any
}

func NewSqlBuilder(initial, whereInitial, setInitial int) *SqlBuilder {
	r := &SqlBuilder{
		whereStmt: make([]string, 0, whereInitial),
		setStmt:   make([]string, 0, setInitial),
	}
	r.b.Grow(initial)
	return r
}

func (s *SqlBuilder) AppendWhereStmtConditional(cond bool, sql string, args ...any) *SqlBuilder {
	if !cond {
		return s
	}
	s.whereStmt = append(s.whereStmt, sql)
	s.args = append(s.args, args...)

	return s
}

func (s *SqlBuilder) EndWhereStmt() *SqlBuilder {
	if len(s.whereStmt) == 0 {
		return s
	}

	s.b.WriteString("WHERE 1 = 1 ")
	s.b.WriteString(strings.Join(s.whereStmt, " "))
	s.b.WriteString(" ")

	return s
}

func (s *SqlBuilder) AppendSetStmtConditional(cond bool, sql string, args ...any) *SqlBuilder {
	if !cond {
		return s
	}
	s.setStmt = append(s.setStmt, sql)
	s.args = append(s.args, args...)

	return s
}

func (s *SqlBuilder) EndSetStmt() *SqlBuilder {
	if len(s.setStmt) == 0 {
		return s
	}

	s.b.WriteString("SET ")
	s.b.WriteString(strings.Join(s.setStmt, ", "))
	s.b.WriteString(" ")

	return s
}

func (s *SqlBuilder) AppendStmt(sql string, args ...any) *SqlBuilder {
	s.b.WriteString(sql)
	s.args = append(s.args, args...)
	return s
}

func (s *SqlBuilder) AppendStmtConditional(cond bool, sql string, args ...any) *SqlBuilder {
	if cond {
		s.b.WriteString(sql)
		s.args = append(s.args, args...)
	}

	return s
}

// 由于go不支持方法泛型
func AppendLoopStmt[T any](s *SqlBuilder, collection []T, sep, open, close string, fn func(T) []any, sql string) {
	if len(collection) == 0 {
		return
	}

	if open != "" {
		s.b.WriteString(open)
	}

	for i, v := range collection {
		args := fn(v)
		s.args = append(s.args, args...)
		s.b.WriteString(sql)
		if i < len(collection)-1 && sep != "" {
			s.b.WriteString(sep)
		}
	}

	if close != "" {
		s.b.WriteString(close)
	}
	s.b.WriteString(" ")

	return
}

type ConditionalSql struct {
	Cond bool
	Sql  string
	Args []any
}

func NewConditionSql(cond bool, sql string, args ...any) ConditionalSql {
	return ConditionalSql{Cond: cond, Sql: sql, Args: args}
}

func (s *SqlBuilder) appendStmtChoosed(keyWord string, conds []ConditionalSql, defaultSql string, args []any) {
	for i := range conds {
		if conds[i].Cond {
			s.b.WriteString(keyWord)
			s.b.WriteString(conds[i].Sql)
			s.args = append(s.args, conds[i].Args...)
			return
		}
	}

	if defaultSql == "" {
		return
	}

	s.b.WriteString("WHERE ")
	s.b.WriteString(defaultSql)
	s.args = append(s.args, args...)
}

func (s *SqlBuilder) AppendWhereStmtChoosed(conds []ConditionalSql, defaultSql string, args []any) {
	s.appendStmtChoosed("WHERE ", conds, defaultSql, args)
}

func (s *SqlBuilder) AppendSetStmtChoosed(conds []ConditionalSql, defaultSql string, args []any) {
	s.appendStmtChoosed("SET ", conds, defaultSql, args)
}

func (s *SqlBuilder) String() string {
	return s.b.String()
}

func (s *SqlBuilder) Args() []any {
	return s.args
}

func MakeSlice[T any](ss ...T) []T {
	return ss
}
