package vulcan

import (
	"strings"
)

type OrderItem struct {
	Column string
	Desc   bool
}

type OrderItems []OrderItem

func (o OrderItems) SqlStmt() string {
	builder := strings.Builder{}
	for _, order := range o {
		builder.WriteString(" ORDER BY ")
		builder.WriteString(order.Column)
		if order.Desc {
			builder.WriteString(" DESC")
		} else {
			builder.WriteString(" ASC")
		}
	}

	return builder.String()
}

type Page interface {
	PageNum() int
	PageSize() int
	TotalCount() int
	TotalPages() int
	SetTotalCount(int)
	SetTotalPages(int)
	IsSelectCount() bool // 是否查询count
	GetSelectCountSql(originSql string) string
	Orders() OrderItems
}

type Paging struct {
	pageSize   int
	pageNum    int
	totalCount int
	totalPages int
	orders     OrderItems
}

func NewPaging(currentPage, pageSize int) *Paging {
	return &Paging{
		pageNum:  currentPage,
		pageSize: pageSize,
	}
}

func (p *Paging) SetPageSize(pageSize int) *Paging {
	p.pageSize = pageSize
	return p
}

func (p *Paging) SetCurrentPage(cur int) *Paging {
	p.pageNum = cur
	return p
}

func (p *Paging) AddOrderItems(orderItems ...OrderItem) *Paging {
	p.orders = append(p.orders, orderItems...)
	return p
}

func (p *Paging) AddDescs(columns ...string) *Paging {
	for _, column := range columns {
		p.orders = append(p.orders, OrderItem{
			Column: column,
			Desc:   true,
		})
	}
	return p
}

func (p *Paging) AddAscs(columns ...string) *Paging {
	for _, column := range columns {
		p.orders = append(p.orders, OrderItem{
			Column: column,
		})
	}
	return p
}

func (p *Paging) PageNum() int {
	return p.pageNum
}

func (p *Paging) PageSize() int {
	return p.pageSize
}

func (p *Paging) TotalCount() int {
	return p.totalCount
}

func (p *Paging) TotalPages() int {
	return p.totalPages
}

func (p *Paging) SetTotalCount(i int) {
	p.totalCount = i
}

func (p *Paging) SetTotalPages(i int) {
	p.totalPages = i
}

func (p *Paging) IsSelectCount() bool {
	return true
}

func (p *Paging) GetSelectCountSql(originSql string) string {
	start := strings.Index(originSql, "SELECT") + 6
	end := strings.Index(originSql, "FROM")
	return originSql[:start] + " COUNT(*) " + originSql[end:]
}

func (p *Paging) Orders() OrderItems {
	return p.orders
}
