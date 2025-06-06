package vulcan

import "strings"

type SqlBuilder[T any] struct {
	b         strings.Builder
	whereStmt []string
	setStmt   []string
	args      []any
}

func NewSqlBuilderGenerics[T any](initial, whereInitial, setInitial int) *SqlBuilder[T] {
	r := &SqlBuilder[T]{
		whereStmt: make([]string, 0, whereInitial),
		setStmt:   make([]string, 0, setInitial),
	}
	r.b.Grow(initial)
	return r
}

func NewSqlBuild(initial, whereInitial, setInitial int) *SqlBuilder[struct{}] {
	return NewSqlBuilderGenerics[struct{}](initial, whereInitial, setInitial)
}

func (s *SqlBuilder[T]) AppendWhereStmtConditional(cond bool, arg any, sql string) *SqlBuilder[T] {
	if !cond {
		return s
	}
	s.whereStmt = append(s.whereStmt, sql)
	if arg != nil {
		s.args = append(s.args, arg)
	}

	return s
}

func (s *SqlBuilder[T]) EndWhereStmt() *SqlBuilder[T] {
	if len(s.whereStmt) == 0 {
		return s
	}

	s.b.WriteString("WHERE 1 = 1 ")
	s.b.WriteString(strings.Join(s.whereStmt, " "))

	return s
}

func (s *SqlBuilder[T]) AppendSetStmtConditional(cond bool, arg any, sql string) *SqlBuilder[T] {
	if !cond {
		return s
	}
	s.setStmt = append(s.setStmt, sql)
	if arg != nil {
		s.args = append(s.args, arg)
	}

	return s
}

func (s *SqlBuilder[T]) AppendSetStmt(arg any, sql string) *SqlBuilder[T] {
	s.setStmt = append(s.setStmt, sql)
	if arg != nil {
		s.args = append(s.args, arg)
	}

	return s
}

func (s *SqlBuilder[T]) EndSetStmt() *SqlBuilder[T] {
	if len(s.setStmt) == 0 {
		return s
	}

	s.b.WriteString("SET ")
	s.b.WriteString(strings.Join(s.setStmt, ", "))

	return s
}

func (s *SqlBuilder[T]) AppendStmt(sql string, args ...any) *SqlBuilder[T] {
	s.b.WriteString(sql)
	s.args = append(s.args, args...)
	return s
}

func (s *SqlBuilder[T]) AppendStmtConditional(cond bool, sql string) *SqlBuilder[T] {
	if cond {
		s.b.WriteString(sql)
	}

	return s
}

func (s *SqlBuilder[T]) AppendLoopStmt(collection []T, sep, open, close string, fn func(T) []any, sql string) *SqlBuilder[T] {
	if len(collection) == 0 {
		return s
	}

	if open != "" {
		s.b.WriteString(open)
	}

	for i, v := range collection {
		args := fn(v)
		s.args = append(s.args, args...)
		s.b.WriteString(sql)
		if i != 0 {
			s.b.WriteString(sep)
		}
	}

	if close != "" {
		s.b.WriteString(close)
	}

	return s
}

func (s *SqlBuilder[T]) appendStmtChoosed(keyWord string, conds []bool, args []any, sqls []string) {
	for i, cond := range conds {
		if cond {
			s.b.WriteString("WHERE ")
			s.b.WriteString(sqls[i])
			s.args = append(s.args, args[i])
			return
		}
	}

	if len(conds) == len(args) {
		return
	}
	s.b.WriteString("WHERE ")
	s.b.WriteString(sqls[len(conds)])
	s.args = append(s.args, args[len(conds)])
}

func (s *SqlBuilder[T]) AppendWhereStmtChoosed(conds []bool, args []any, sqls []string) {
	s.appendStmtChoosed("WHERE ", conds, args, sqls)
}

func (s *SqlBuilder[T]) AppendSetStmtChoosed(conds []bool, args []any, sqls []string) {
	s.appendStmtChoosed("SET ", conds, args, sqls)
}

func (s *SqlBuilder[T]) String() string {
	return s.b.String()
}

func (s *SqlBuilder[T]) Args() []any {
	return s.args
}

func MakeSlice[T any](conds ...T) []T {
	return conds
}
