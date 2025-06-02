package mapper

import (
	"github.com/mangohow/vulcan/db/types"
	"github.com/mangohow/vulcan/db/wrapper"
)

type BaseMapper[T any] interface {
	// Insert 插入一条记录, 返回主键Id
	Insert(entity T) (int, error)

	// DeleteById 根据主键Id删除记录, 返回影响的条数
	DeleteById(id int) (int, error)

	// Delete 根据条件删除
	Delete() (int, error)

	// SelectById 根据主键Id进行查询
	SelectById(id int) (*T, error)

	// SelectOne 根据条件查询一条记录
	SelectOne() (*T, error)

	// SelectList 根据查询条件查询多条记录
	SelectList() ([]*T, error)

	// SelectPage 查询一页记录
	SelectPage(types.Page[T], wrapper.QueryWrapper[T]) (types.Page[T], error)
}
