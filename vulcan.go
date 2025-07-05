package vulcan

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbRef *sql.DB
)

type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)

	Query(query string, args ...any) (*sql.Rows, error)

	QueryRow(query string, args ...any) *sql.Row
}

type ExecOption struct {
	SqlStmt   string `name:"sql"`
	Args      []any  `name:"args"`
	Execer    Execer `name:"execer"`
	Extension any    `name:"extension"`
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

func (e *ExecOption) Select(query string, args ...any) (*sql.Rows, error) {
	return e.Execer.Query(query, args...)
}

func (e *ExecOption) Get(query string, args ...any) *sql.Row {
	return e.Execer.QueryRow(query, args...)
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

func Transactional(fn func(opts ...Option) error) (err error) {
	var tx *sql.Tx
	tx, err = dbRef.Begin()
	if err != nil {
		return err
	}
	defer func() {
		var e error
		if r := recover(); r != nil || err != nil {
			e = tx.Rollback()
			if e == nil && r != nil {
				e = fmt.Errorf("recovered from %v", r)
			}
		} else {
			e = tx.Commit()
		}
		if e != nil {
			err = e
		}
	}()

	return fn(WithTransaction(tx))
}

func OpenMysql(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	dbRef = db

	return db, nil
}

// tableName: xxx
// genFunc: Add, BatchAdd, DeleteById, UpdateById, GetById, SelectListByIds, SelectPage
type TableProperty struct{}
