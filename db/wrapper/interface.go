package wrapper

type QueryWrapper[T any] interface {
	// Eq 指定==查询条件
	Eq(field string, val interface{}) QueryWrapper[T]

	// Ne 指定!=查询条件
	Ne(field string, val interface{}) QueryWrapper[T]

	// Gt 指定>查询条件
	Gt(field string, val interface{}) QueryWrapper[T]

	// Lt 指定<查询条件
	Lt(field string, val interface{}) QueryWrapper[T]

	// Le 指定<=查询条件
	Le(field string, val interface{}) QueryWrapper[T]

	// Ge 指定>=查询条件
	Ge(field string, val interface{}) QueryWrapper[T]

	// Select 指定查询的字段
	Select(fields ...string) QueryWrapper[T]

	// In 指定范围查询条件
	In(field string, values ...interface{}) QueryWrapper[T]

	// Like 指定模糊查询条件, 自动添加% %
	Like(field string, value string) QueryWrapper[T]
}

type UpdateWrapper[T any] interface {
}
