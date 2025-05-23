package vulcan

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

var (
	dbRef *sqlx.DB
)

type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)

	Select(dest any, query string, args ...any) error
}

type ExecOption struct {
	SqlStmt  string `name:"sql"`
	Args     []any  `name:"args"`
	Execer   Execer `name:"execer"`
	FirstArg any    `name:"firstArg"`
}

type SQLType int

const (
	SQLTypeSelect SQLType = iota
	SQLTypeUpdate
	SQLTypeInsert
	SQLTypeDelete
)

type ResultOption struct {
	SQlType   SQLType
	Result    any
	Err       error
	SQLResult sql.Result
}

func NewResultOption(sqlType SQLType, res any, err error, sqlRes sql.Result) *ResultOption {
	return &ResultOption{
		SQlType:   sqlType,
		Result:    res,
		Err:       err,
		SQLResult: sqlRes,
	}
}

func (e *ExecOption) Exec(query string, args ...any) (sql.Result, error) {
	return e.Execer.Exec(query, args...)
}

func (e *ExecOption) Select(dest any, query string, args ...any) error {
	return e.Execer.Select(dest, query, args...)
}

type Option func(*ExecOption)

func WithTransaction(execer Execer) Option {
	return func(o *ExecOption) {
		o.Execer = execer
	}
}

func StartTransaction() (*sql.Tx, error) {
	return dbRef.Begin()
}
