package types

type SqlCondition struct {
	Field string
	Cond  SqlKeyWord
	Value interface{}
}

type KvPair[T any] struct {
	Key   string
	Value T
}
