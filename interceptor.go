package vulcan

import (
	"fmt"
	"strings"
)

type ExecInterceptor interface {
	PreHandle(*ExecOption)
	PostHandle(*ResultOption)
}

type FuncInterceptor struct {
	PreHandlerFn  func(*ExecOption)
	PostHandlerFn func(option *ResultOption)
}

func (f FuncInterceptor) PreHandle(option *ExecOption) {
	f.PreHandlerFn(option)
}

func (f FuncInterceptor) PostHandle(option *ResultOption) {
	f.PostHandlerFn(option)
}

var (
	executeInterceptors   []ExecInterceptor
	sqlDebugInterceptor   ExecInterceptor
	paginationInterceptor ExecInterceptor
)

func getInterceptors() []ExecInterceptor {
	interceptors := make([]ExecInterceptor, 0, len(executeInterceptors)+2)
	interceptors = append([]ExecInterceptor{}, executeInterceptors...)
	if paginationInterceptor != nil {
		interceptors = append(interceptors, paginationInterceptor)
	}
	if sqlDebugInterceptor != nil {
		interceptors = append(interceptors, sqlDebugInterceptor)
	}

	return interceptors
}

func InvokePreHandler(option *ExecOption, opts ...Option) {
	for _, opt := range opts {
		opt(option)
	}

	interceptors := getInterceptors()
	if len(interceptors) == 0 {
		return
	}

	for _, i := range interceptors {
		i.PreHandle(option)
	}
}

func InvokePostHandler(option *ResultOption) {
	interceptors := getInterceptors()
	if len(interceptors) == 0 {
		return
	}

	for _, i := range interceptors {
		i.PostHandle(option)
	}
}

type DebugLogger interface {
	Debug(format string, args ...any)
}

func SetupSqlDebugInterceptor(logger DebugLogger) {
	sqlDebugInterceptor = FuncInterceptor{
		PreHandlerFn: func(option *ExecOption) {
			logger.Debug("SQL ==> %s", option.SqlStmt)
			logger.Debug("Parameters ==> "+strings.Repeat("%v, ", len(option.Args)), option.Args...)
		},
	}
}

func SetupPaginationInterceptor() {
	paginationInterceptor = FuncInterceptor{
		PreHandlerFn: func(option *ExecOption) {
			if option.FirstArg == nil || !strings.HasPrefix(option.SqlStmt, "SELECT") {
				return
			}

			page, ok := option.FirstArg.(Page)
			if !ok {
				return
			}

			tail := fmt.Sprintf(" LIMIT %d, %d", page.PageSize(), (page.CurrentPage()-1)*page.PageSize())
			if len(page.Orders()) != 0 {
				tail = page.Orders().SqlStmt() + tail
			}

			option.SqlStmt += tail
			if !page.IsSelectCount() {
				return
			}

			start := strings.Index(option.SqlStmt, "SELECT") + 6
			end := strings.Index(option.SqlStmt, "FROM")
			sqlStmt := option.SqlStmt[:start] + " COUNT(*) " + option.SqlStmt[end:]
			if sqlDebugInterceptor != nil {
				sqlDebugInterceptor.PreHandle(&ExecOption{SqlStmt: sqlStmt})
			}
			var count int
			option.Execer.Exec(sqlStmt, &count)
			page.SetTotalCount(count)
			totalPage := count / page.PageSize()
			if count%page.PageSize() != 0 {
				totalPage++
			}
			page.SetTotalPages(totalPage)
		},
	}
}

func SetPaginationInterceptor(interceptor ExecInterceptor) {
	paginationInterceptor = interceptor
}

func SetSqlDebugInterceptor(interceptor ExecInterceptor) {
	sqlDebugInterceptor = interceptor
}

func AddInterceptors(interceptors ...ExecInterceptor) {
	executeInterceptors = append(executeInterceptors, interceptors...)
}
