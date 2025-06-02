package wrapper

import (
	"github.com/mangohow/vulcan/db/internal/types"
	"strings"
)

type queryWrapper[T any] struct {
	selectFields  []string
	condition     []types.SqlCondition
	likeCondition []types.KvPair[[]any]
}

func NewQueryWrapper[T any]() QueryWrapper[T] {
	return &queryWrapper[T]{}
}

func (q *queryWrapper[T]) Eq(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Eq, val)
}

func (q *queryWrapper[T]) Ne(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Ne, val)
}

func (q *queryWrapper[T]) Gt(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Gt, val)
}

func (q *queryWrapper[T]) Lt(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Lt, val)
}

func (q *queryWrapper[T]) Le(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Le, val)
}

func (q *queryWrapper[T]) Ge(field string, val interface{}) QueryWrapper[T] {
	return q.addCondition(field, types.Ge, val)
}

func (q *queryWrapper[T]) addCondition(field string, cond types.SqlKeyWord, val any) QueryWrapper[T] {
	q.condition = append(q.condition, types.SqlCondition{
		Field: field,
		Cond:  cond,
		Value: val,
	})

	return q
}

func (q *queryWrapper[T]) Select(fields ...string) QueryWrapper[T] {
	q.selectFields = append(q.selectFields, fields...)
	return q
}

func (q *queryWrapper[T]) In(field string, values ...interface{}) QueryWrapper[T] {
	if len(values) == 0 {
		return q
	}
	b := strings.Builder{}
	b.Grow(len(field) + (len(values)-1)*3 + 7)
	b.WriteString(field)
	b.WriteString(" IN (?")
	for i := 1; i < len(values); i++ {
		b.WriteString(", ?")
	}
	b.WriteString(")")
	q.likeCondition = append(q.likeCondition, types.KvPair[[]any]{
		Key:   b.String(),
		Value: values,
	})
	return q
}

func (q *queryWrapper[T]) Like(field string, value string) QueryWrapper[T] {
	q.condition = append(q.condition, types.SqlCondition{
		Field: field,
		Cond:  types.Like,
		Value: "%" + value + "%",
	})
	return q
}
