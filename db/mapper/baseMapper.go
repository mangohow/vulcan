package mapper

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/mangohow/vulcan/db/sqlbuilder"
	"github.com/mangohow/vulcan/db/types"
	"github.com/mangohow/vulcan/db/wrapper"
	"reflect"
	"strings"
)

var tableNameType = reflect.TypeOf(types.TableName{})

type BaseMapperImpl[T any] struct {
	tableName   string
	tableFields map[string]int
	primary     primaryKeyInfo
	newFunc     func() *T // 创建类型T的函数

	dbOpt *sqlx.DB

	// 根据Id查找的sql进行缓存
	selectByIdSql string
	// 根据Id进行删除的sql进行缓存
	deleteByIdSql string
}

func NewBaseMapperImpl[T any](dbOpt *sqlx.DB) BaseMapper[T] {
	checkType[T]()
	// 利用反射获取tableName和tableField
	name, primary, fields := getTableInfo[T]()
	if name == "" {
		panic("can't get table name")
	}
	if primary.name == "" || primary.index == -1 {
		panic("can't get table primary id")
	}

	b := &BaseMapperImpl[T]{
		tableName:   name,
		tableFields: fields,
		newFunc: func() *T {
			return new(T)
		},
		dbOpt: dbOpt,
	}

	// 该sql不会变化, 直接缓存起来
	b.selectByIdSql = (&sqlbuilder.SelectSQLBuilder{
		TableName: b.tableName,
		Condition: []string{b.primary.name},
	}).Build()

	b.deleteByIdSql = (&sqlbuilder.DeleteBuilder{
		TableName: b.tableName,
		Condition: []string{b.primary.name},
	}).Build()

	return b
}

func (b *BaseMapperImpl[T]) SelectById(id int) (*T, error) {
	res := b.newFunc()
	err := b.dbOpt.Get(res, b.selectByIdSql, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (b *BaseMapperImpl[T]) SelectPage(p types.Page[T], wp wrapper.QueryWrapper[T]) (types.Page[T], error) {
	if p == nil {
		return nil, errors.New("input parameter page is nil")
	}
	pageStart := (p.PageNum() - 1) * p.PageSize()

	builder := &sqlbuilder.SelectSQLBuilder{
		TableName:   b.tableName,
		Condition:   nil,
		DescOrderBy: nil,
		AscOrderBy:  nil,
		Limit:       []int{pageStart, p.PageSize()},
	}

	// 查询记录的sql
	selectSql := builder.Build()
	builder.Fields = []string{"COUNT(*)"}
	// 查询总数的sql
	countSql := builder.Build()

	res := make([]*T, 0, p.PageNum())
	err := b.dbOpt.Select(res, selectSql)
	if err != nil {
		return nil, err
	}

	var count int
	err = b.dbOpt.Get(&count, countSql)
	if err != nil {
		return nil, err
	}

	return nil, nil // TODO
}

func (b *BaseMapperImpl[T]) DeleteById(id int) (int, error) {
	res, err := b.dbOpt.Exec(b.deleteByIdSql, id)
	if err != nil {
		return 0, err
	}

	affected, _ := res.RowsAffected()
	return int(affected), nil
}

func (b *BaseMapperImpl[T]) Insert(entity T) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BaseMapperImpl[T]) Delete() (int, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BaseMapperImpl[T]) SelectOne() (*T, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BaseMapperImpl[T]) SelectList() ([]*T, error) {
	//TODO implement me
	panic("implement me")
}

// 类型校验, 必须是结构体或结构体指针, 不允许多级指针或其他类型
func checkType[T any]() {
	var val T
	rt := reflect.TypeOf(val)
	if rt.Kind() != reflect.Ptr {
		panic("type T must be a struct")
	}
}

// 获取一个根据索引返回字段的函数
func fieldFunc[T any](val T) func(i int) interface{} {
	rv := reflect.ValueOf(val)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	return func(i int) interface{} {
		return rv.Field(i).Interface()
	}
}

type primaryKeyInfo struct {
	// 主键id字段名称
	name string
	// 主键id字段在结构体索引
	index int
	// 是否是自增主键
	autoIncrement bool
}

// 获取表名和所有字段
func getTableInfo[T any]() (string, primaryKeyInfo, map[string]int) {
	var (
		t         T
		tableName string
		primary   = primaryKeyInfo{
			index: -1,
		}
		tableFields = make(map[string]int)
	)
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		panic("type T must be a struct")
	}
	n := rt.NumField()
	for i := 0; i < n; i++ {
		field := rt.Field(i)

		// 获取tableField
		if tf := field.Tag.Get(types.TableFieldTagKey); tf != "" {
			tags := strings.Split(tf, ",")
			if len(tags) == 0 {
				tableFields[tf] = i
			} else {
				for _, tag := range tags {
					switch tag {
					case types.TablePrimaryIdTagValue:
						tableFields[tags[0]] = i
						primary.name = tags[0]
						primary.index = i
					case types.TableAutoFillTagValue:
						primary.autoIncrement = true
					}
				}
			}
		}

		if tableName != "" || field.Type != tableNameType {
			continue
		}

		tableName = field.Tag.Get(types.TableNameTagKey)
	}

	return tableName, primary, tableFields
}
