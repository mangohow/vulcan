package mapper

import (
	"github.com/mangohow/vulcan/db/types"
	"github.com/mangohow/vulcan/db/wrapper"
)

type Config struct {
	IncludeFields []string
	ExcludeFields []string
}

type Option func(*Config)

func IncludeFields(fields ...string) Option {
	return func(c *Config) {
		c.IncludeFields = append(c.IncludeFields, fields...)
	}
}

func ExcludeFields(fields ...string) Option {
	return func(c *Config) {
		c.ExcludeFields = append(c.ExcludeFields, fields...)
	}
}

type BaseMapper[T any] interface {
	// Insert 插入一条记录, 返回影响的行数
	// 默认插入除自增Id之外的字段, 同时返回自增Id
	// 可以通过opts指定要插入的字段或排除的字段
	Insert(entity T, opts ...Option) (int, error)

	// InsertBatch 插入多条记录, 返回影响的行数
	// 默认插入除自增Id之外的字段
	// 可以通过opts指定要插入的字段或排除的字段
	InsertBatch(entities []T, opts ...Option) (int, error)

	// DeleteById 根据主键Id删除记录, 返回影响的条数
	DeleteById(id int) (int, error)

	// DeleteBatchIds 根据id集合批量删除, 返回影响的条数
	DeleteBatchIds(idList []int) (int, error)

	// Delete 根据条件删除, 返回影响的条数
	Delete() (int, error)

	// UpdateById 根据主键更新, 默认全部更新, 可以通过opts指定要更新的字段或排除的字段
	UpdateById(entity T, opts ...Option) (int, error)

	// UpdateByBatchIds 根据主键批量更新, 默认全部更新, 可以通过opts指定要更新的字段或排除的字段
	UpdateByBatchIds(entity []T, opts ...Option) (int, error)

	// Update 根据条件更新, 返回影响的条数
	Update() (int, error)

	// SelectById 根据主键Id进行查询
	SelectById(id int) (T, error)

	// SelectBatchIds 根据id集合批量查询
	SelectBatchIds(idList []int) ([]*T, error)

	// SelectOne 根据条件查询一条记录
	SelectOne() (T, error)

	// SelectList 根据查询条件查询多条记录
	SelectList() ([]T, error)

	// SelectPage 查询一页记录
	SelectPage(types.Page[T], wrapper.QueryWrapper[T]) (types.Page[T], error)
}
