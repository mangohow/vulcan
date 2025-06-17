package vulcan

import (
	"database/sql"
	"fmt"
	"testing"
)

type debugLogger struct{}

func (d debugLogger) Debug(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type fakeExecer struct {
}

func (f fakeExecer) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (f fakeExecer) QueryRow(query string, args ...any) *sql.Row {
	return nil
}

func (f fakeExecer) Exec(query string, args ...any) (sql.Result, error) {
	if len(args) > 0 {
		arg := args[0]
		switch a := arg.(type) {
		case *int:
			*a = 500
		}
	}
	return nil, nil
}

func TestPaginationInterceptor(t *testing.T) {
	SetupSqlDebugInterceptor(debugLogger{})
	SetupPaginationInterceptor()

	paging := NewPaging(1, 10).AddDescs("create_time")
	paginationInterceptor.PreHandle(&ExecOption{
		SqlStmt:   "SELECT username, password FROM t_user WHERE id > ?",
		Execer:    fakeExecer{},
		Extension: paging,
	})
	fmt.Printf("%+v\n", paging)
}
