package vulcan

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Handler func(option *ExecOption) (any, error)
type InterceptorHandler func(option *ExecOption, next Handler) (any, error)

var (
	executeInterceptors         []InterceptorHandler
	sqlDebugInterceptor         InterceptorHandler
	paginationInterceptor       InterceptorHandler
	slowQueryLoggingInterceptor InterceptorHandler
)

func Invoke[T any](option *ExecOption, execHandler func() (T, error)) (T, error) {
	if option.Ctx == nil {
		option.Ctx = context.Background()
	}
	// 构建拦截器链
	interceptorChain := buildInterceptorChain(option)
	if interceptorChain == nil {
		return execHandler()
	}

	// 创建最终的执行处理器
	finalHandler := func(option *ExecOption) (any, error) {
		return execHandler()
	}

	// 执行拦截器链
	res, err := interceptorChain(option, finalHandler)
	if err != nil {
		return *new(T), err
	}

	return res.(T), nil
}

// buildInterceptorChain 构建拦截器链
func buildInterceptorChain(option *ExecOption) InterceptorHandler {
	// 从上下文中获取额外的拦截器
	var extraInterceptors []InterceptorHandler
	value := option.Ctx.Value(interceptorKey{})
	if value != nil {
		if handler, ok := value.(InterceptorHandler); ok {
			extraInterceptors = append(extraInterceptors, handler)
		} else if handlers, ok := value.([]InterceptorHandler); ok {
			extraInterceptors = handlers
		}
	}

	// 获取按正确顺序排列的全局拦截器
	interceptors := make([]InterceptorHandler, 0, len(executeInterceptors)+len(extraInterceptors)+4)

	// 获取缓存interceptor
	if cacheInterceptor := getCacheInterceptor(option.Ctx); cacheInterceptor != nil {
		interceptors = append(interceptors, cacheInterceptor)
	}

	// 1. 分页拦截器优先执行
	if paginationInterceptor != nil {
		interceptors = append(interceptors, paginationInterceptor)
	}

	// 2. SQL调试拦截器
	if sqlDebugInterceptor != nil {
		interceptors = append(interceptors, sqlDebugInterceptor)
	}

	// 3. 自定义拦截器
	interceptors = append(interceptors, executeInterceptors...)
	interceptors = append(interceptors, extraInterceptors...)

	// 4. 慢查询日志拦截器最后执行
	if slowQueryLoggingInterceptor != nil {
		interceptors = append(interceptors, slowQueryLoggingInterceptor)
	}

	// 如果没有拦截器，直接返回nil
	if len(interceptors) == 0 {
		return nil
	}

	// 返回链式调用的拦截器
	return chainInterceptors(interceptors)
}

// chainInterceptors 将多个拦截器链接成一个
func chainInterceptors(interceptors []InterceptorHandler) InterceptorHandler {
	return func(option *ExecOption, finalHandler Handler) (any, error) {
		return interceptors[0](option, getChainHandler(interceptors, 0, finalHandler))
	}
}

// getChainHandler 获取下一个拦截器
func getChainHandler(interceptors []InterceptorHandler, curr int, finalHandler Handler) Handler {
	if curr == len(interceptors)-1 {
		return finalHandler
	}

	return func(option *ExecOption) (any, error) {
		return interceptors[curr+1](option, getChainHandler(interceptors, curr+1, finalHandler))
	}
}

type DebugLogger interface {
	Debug(format string, args ...any)
}

func SetupSqlDebugInterceptor(logger DebugLogger) {
	sqlDebugInterceptor = func(option *ExecOption, next Handler) (any, error) {
		logger.Debug("SQL        ==> %s", option.SqlStmt)
		builder := strings.Builder{}
		for i, arg := range option.Args {
			switch arg.(type) {
			case string:
				builder.WriteString(fmt.Sprintf("%T(%q)", arg, arg))
			case time.Time:
				builder.WriteString(fmt.Sprintf("DATETIME(%s)", arg.(time.Time).Format(time.DateTime)))
			case *time.Time:
				if arg != nil {
					builder.WriteString(fmt.Sprintf("DATETIME(%v)", arg.(*time.Time).Format(time.DateTime)))
				}
			default:
				builder.WriteString(fmt.Sprintf("%T(%v)", arg, arg))
			}
			if i != len(option.Args)-1 {
				builder.WriteString(", ")
			}
		}
		logger.Debug("PARAMETERS ==> " + builder.String())

		return next(option)
	}
}

func SetupPaginationInterceptor() {
	paginationInterceptor = func(option *ExecOption, next Handler) (any, error) {
		if option.Extension == nil || !strings.HasPrefix(option.SqlStmt, "SELECT") {
			return next(option)
		}

		page, ok := option.Extension.(Page)
		if !ok || page.PageSize() == 0 || page.PageNum() == 0 {
			return next(option)
		}

		tail := fmt.Sprintf("LIMIT %d, %d", page.PageSize(), (page.PageNum()-1)*page.PageSize())
		if len(page.Orders()) != 0 {
			tail = page.Orders().SqlStmt() + tail
		}

		option.SqlStmt += tail
		if !page.IsSelectCount() {
			return next(option)
		}

		countSql := page.GetSelectCountSql(option.SqlStmt)

		if sqlDebugInterceptor != nil {
			sqlDebugInterceptor(&ExecOption{SqlStmt: countSql}, func(option *ExecOption) (any, error) {
				return nil, nil
			})
		}
		var count int
		option.Execer.Exec(countSql, &count)
		page.SetTotalCount(count)
		totalPage := count / page.PageSize()
		if count%page.PageSize() != 0 {
			totalPage++
		}
		page.SetTotalPages(totalPage)

		return next(option)
	}
}

type slowQueryKey struct{}
type slowQuerySqlKey struct{}

func SetupSlowQueryLoggingInterceptor(limit int64, loggerFunc func(used int64, sql string)) {
	slowQueryLoggingInterceptor = func(option *ExecOption, next Handler) (any, error) {
		start := time.Now().UnixMilli()
		resp, err := next(option)
		if err != nil {
			return resp, err
		}

		used := time.Now().UnixMilli() - start
		if used > limit {
			loggerFunc(used, option.SqlStmt)
		}

		return resp, nil
	}
}

func SetPaginationInterceptor(interceptor InterceptorHandler) {
	paginationInterceptor = interceptor
}

func SetSqlDebugInterceptor(interceptor InterceptorHandler) {
	sqlDebugInterceptor = interceptor
}

func SetSlowQueryLoggingInterceptor(interceptor InterceptorHandler) {
	slowQueryLoggingInterceptor = interceptor
}

func AddInterceptors(interceptors ...InterceptorHandler) {
	executeInterceptors = append(executeInterceptors, interceptors...)
}
