package types

type Page[T any] interface {
	PageSize() int

	PageNum() int

	TotalCount() int

	TotalPages() int

	Orders() []OrderItem

	Results() []*T

	SetResults([]*T)
}

type OrderItem struct {
	OrderBy string
	Asc     bool
}
