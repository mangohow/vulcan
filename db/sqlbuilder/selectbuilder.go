package sqlbuilder

import (
	"fmt"
	"strconv"
	"strings"
)

type SelectSQLBuilder struct {
	Fields      []string
	TableName   string
	Condition   []string
	DescOrderBy []string
	AscOrderBy  []string
	Limit       []int
}

func (b *SelectSQLBuilder) Build() string {
	builder := strings.Builder{}
	builder.Grow(64)
	builder.WriteString("SELECT ")
	if len(b.Fields) == 0 {
		builder.WriteString("* ")
	} else {
		builder.WriteString(strings.Join(b.Fields, ", "))
	}
	builder.WriteString(" FROM ")
	builder.WriteString(b.TableName)
	builder.WriteString(" WHERE ")
	for i, cond := range b.Condition {
		if i == len(b.Condition)-1 {
			builder.WriteString(cond + " = ? ")
		} else {
			builder.WriteString(cond + " = ?, ")
		}
	}
	orderTotal := len(b.DescOrderBy) + len(b.AscOrderBy)
	if orderTotal > 0 {
		builder.WriteString(" ORDER BY ")
	}

	for _, descOrderBy := range b.DescOrderBy {
		orderTotal--
		if orderTotal == 0 {
			builder.WriteString(descOrderBy + " DESC")
		} else {
			builder.WriteString(descOrderBy + " DESC, ")
		}
	}

	for _, ascOrderby := range b.AscOrderBy {
		orderTotal--
		if orderTotal == 0 {
			builder.WriteString(ascOrderby + " ASC")
		}
		builder.WriteString(ascOrderby + " ASC, ")
	}
	if len(b.Limit) == 1 {
		builder.WriteString(" LIMIT " + strconv.Itoa(b.Limit[0]))
	} else if len(b.Limit) == 2 {
		builder.WriteString(" LIMIT " + fmt.Sprintf("%d, %d", b.Limit[0], b.Limit[1]))
	}

	return builder.String()
}
