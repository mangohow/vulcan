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

// WithTransaction 使用该函数来根据事务对象生成一个Option, 在执行sql操作时传入相应的方法中
func WithTransaction(execer Execer) Option {
	return func(o *ExecOption) {
		o.Execer = execer
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

// TableProperty 使用该类型在一个model结构体中通过tag指定生成的代码所需的配置
// 1、使用tableName指定表名称
// tableName: xxx
// 例如:
//
//		type User struct {
//		    vulcan.TableProperty `tableName:"t_user"`
//			Id       int     `db:"id,pk"`
//			UserName string  `db:"username"`
//	     	Password string  `db:"password"`
//			Email    string  `db:"email"`
//			Address  string  `db:"address"`
//		}
//
// 2、使用gen指定需要生成的函数列表, 如果不指定, 则默认全部生成
// 函数列表如下：
// Add: 新增操作
// BatchAdd: 批量新增
// DeleteById: 根据主键Id删除
// GetById: 根据主键Id查询
// SelectListByIds: 根据主键Id列表查询
// SelectList([2,4] | true): 条件查询, 中括号中的参数为结构体中字段索引, 根据这些字段进行查询; 比如在User结构体中, 2为Password字段, 4为Address字段, 最后一个参数为是否根据字段为默认值来判断是否使用该字段
// SelectPage([2,4] | false): 分页查询, 中括号中的参数为结构体中字段索引, 根据这些字段进行查询, 最后一个参数为是否根据字段为默认值来判断是否使用该字段
// Delete([3] | false): 根据条件删除, 中括号中的参数为结构体中字段索引, 根据这些字段进行查询, 最后一个参数为是否根据字段为默认值来判断是否使用该字段
// UpdateById([1-3] | true): 根据主键Id更新, 中括号中参数为要更新的字段在结构体中的索引, 最后一个参数为是否根据字段为默认值来判断是否使用该字段
//
// updateById中的参数指定需要更新的字段, 使用字段在结构体中的index, 可以使用单数字, 也可以使用index1-index2表示, 闭区间
// 例如：下面的示例指定了要生成的函数有Add、DeleteById、UpdateById和GetById
//
//	      其中UpdateById函数中根据Id更新索引为2、3、4的字段，即Password、Email、Address, true表示在更新时需要判断该字段是不是空(默认值, 字符串为空字符串, int为0...)
//
//			type User struct {
//			    vulcan.TableProperty `gen:"Add,DeleteById,UpdateById([2-4], true),GetById"`
//		 	}
type TableProperty struct{}
