package vulcan

import (
	"context"
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
	Ctx       context.Context
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

// WithTransaction 使用该函数来根据事务对象生成一个Option, 在执行sql操作时传入相应的方法中
func WithTransaction(execer Execer) Option {
	return func(o *ExecOption) {
		o.Execer = execer
	}
}

type interceptorKey struct{}

func WithInterceptors(interceptor ...InterceptorHandler) Option {
	return func(o *ExecOption) {
		ctx := o.Ctx
		if ctx == nil {
			ctx = context.Background()
		}

		o.Ctx = context.WithValue(ctx, interceptorKey{}, interceptor)
	}
}

// StartTransaction 使用该函数来开启一个事务, 返回Tx对象
func StartTransaction() (*sql.Tx, error) {
	return dbRef.Begin()
}

// Transactional 使用该函数来执行事务, 在回调函数中调用数据库操作语句
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

// OpenMysql 连接mysql
func OpenMysql(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}

	dbRef = db

	return db, nil
}
