package dbparser

import (
	"github.com/mangohow/vulcan/internal/ast/parser/types"
	"github.com/mangohow/vulcan/internal/errors"
	"go/ast"
	"go/token"
)

type sqlCall struct {
	funcName string
	args     []ast.Expr
}

type SQLParserFunc func(call *sqlCall) (types.SQL, error)

var (
	sqlParseFuncSet = map[string]SQLParserFunc{
		types.SQLOperateFuncSQL: func(call *sqlCall) (types.SQL, error) {
			return types.NewEmptySQL(), nil
		},
		types.SQLOperateFuncStmt: func(call *sqlCall) (types.SQL, error) {
			if len(call.args) != 1 {
				return nil, errors.Errorf("operate Stmt error, must have only one parameter")
			}
			arg := call.args[0]
			v, ok := arg.(*ast.BasicLit)
			if !ok || v.Kind != token.STRING {
				return nil, errors.Errorf("operate Stmt error, invalid parameter")
			}

			return types.NewSimpleStmt(v.Value), nil
		},
		types.SQLOperateFuncWhere: func(call *sqlCall) (types.SQL, error) {
			if len(call.args) != 1 {
				return nil, errors.Errorf("operate Where error, must have only one parameter")
			}
			arg := call.args[0]
			calls := parseAllCallExprDepth(arg)

			return parseCond(calls, types.SQLOperateFuncWhere)
		},
		types.SQLOperateFuncSet: func(call *sqlCall) (types.SQL, error) {
			if len(call.args) != 1 {
				return nil, errors.Errorf("operate Set error, must have only one parameter")
			}

			arg := call.args[0]
			calls := parseAllCallExprDepth(arg)

			return parseCond(calls, types.SQLOperateFuncSet)
		},
		types.SQLOperateFuncIf: func(call *sqlCall) (types.SQL, error) { // TODO 补充或删除
			return nil, nil
		},
		types.SQLOperateFuncForeach: func(call *sqlCall) (types.SQL, error) {
			if len(call.args) != 6 {
				return nil, errors.Errorf("operate Foreach error, invalid parameter")
			}

			args := make([]string, 0, 6)
			for i, arg := range call.args {
				v, ok := arg.(*ast.BasicLit)
				if !ok || v.Kind != token.STRING {
					return nil, errors.Errorf("operate Foreach error, parameter %d invalid", i)
				}
				args = append(args, v.Value)
			}

			return types.NewForeachStmt(args[0], args[1], args[2], args[3], args[4], args[5]), nil
		},
		types.SQLOperateFuncBuild: func(call *sqlCall) (types.SQL, error) {
			return types.NewEmptySQL(), nil
		},
	}
)

// parseCond 解析条件语句，如 WHERE 或 SET。
// 该函数根据提供的 sqlCall 列表和操作名称来构建相应的 SQL 条件语句。
// 参数:
//
//	calls - 一个包含 SQL 调用信息的切片。
//	optName - 操作名称，用于指定期望的 SQL 语句类型（如 WHERE 或 SET）。
//
// 返回值:
//
//	成功时返回解析后的 SQL 语句对象，否则返回错误。
func parseCond(calls []*sqlCall, optName string) (types.SQL, error) {
	// 检查 calls 列表是否为空，如果为空则返回错误。
	if len(calls) == 0 {
		return nil, errors.Errorf("invalid cond in %s", optName)
	}

	// 初始化条件语句和错误变量。
	var (
		stmt types.Cond
		err  error
	)

	// 根据第一个调用的函数名来决定使用哪种类型的条件语句进行解析。
	switch calls[0].funcName {
	case types.SQLOperateFuncIf:
		// 解析 IF 类型的条件语句。
		stmt, err = parseCondIf(calls)
	case types.SQLOperateFuncCHOOSE:
		// 解析 CHOOSE 类型的条件语句。
		stmt, err = parseCondChoose(calls)
	default:
		// 如果函数名不匹配已知类型，则返回错误。
		return nil, errors.Errorf("invalid operate func %s", calls[0].funcName)
	}

	// 如果解析过程中发生错误，则返回错误。
	if err != nil {
		return nil, err
	}

	// 根据 optName 来决定返回哪种类型的 SQL 语句对象。
	switch optName {
	case types.SQLOperateFuncWhere:
		// 如果是 WHERE 类型的操作名，则返回 WhereStmt 类型的 SQL 语句对象。
		return types.NewWhereStmt(stmt), nil
	case types.SQLOperateFuncSet:
		// 如果是 SET 类型的操作名，则返回 SetStmt 类型的 SQL 语句对象。
		return types.NewSetStmt(stmt), nil
	}

	// 如果 optName 不匹配已知的操作名，则返回错误。
	return nil, errors.Errorf("invalid annotation func %s", optName)
}

// parseCondIf 解析条件语句以构建条件链。
// 该函数接收一个sqlCall对象的切片，每个sqlCall对象包含一组参数，
// 其中每个参数代表一个IF条件语句的组成部分。
// 函数的目的是验证这些条件语句的结构，并构建一个条件链对象。
// 如果任何条件语句的结构不符合预期，函数将返回一个错误。
func parseCondIf(calls []*sqlCall) (types.Cond, error) {
	// 初始化一个IfStmt对象的切片，用于存储解析后的条件语句。
	stmts := make([]*types.IfStmt, 0, len(calls))
	// 遍历每个sqlCall对象，解析并验证条件语句。
	for i, call := range calls {
		// 检查条件语句的参数数量是否正确。
		if len(call.args) != 2 {
			return nil, errors.Errorf("invalid if cond, %d", i)
		}

		// 验证并解析第一个参数，确保它是一个二元表达式。
		arg1, ok := call.args[0].(*ast.BinaryExpr)
		if !ok {
			return nil, errors.Errorf("invalid if cond, cond stmt is invalid, %d", i)
		}
		// 验证并解析第二个参数，确保它是一个字符串类型的字面量。
		arg2, ok := call.args[1].(*ast.BasicLit)
		if !ok || arg2.Kind != token.STRING {
			return nil, errors.Errorf("invalid if cond, sql is invalid, %d", i)
		}
		// 使用解析的参数构建一个IfStmt对象，并添加到stmts切片中。
		ifStmt := types.NewIfStmt(arg1, arg2.Value)
		stmts = append(stmts, ifStmt)
	}

	// 使用解析后的所有条件语句构建并返回一个IfChainStmt对象。
	return types.NewIfChainStmt(stmts), nil
}

// parseCondChoose 解析条件选择语句。
// 该函数接收一个sqlCall对象的切片，并尝试将其转换为一个条件选择语句（types.Cond）。
// 它主要处理两种情况：当条件满足时（WHEN语句）和默认情况（OTHERWISE语句）。
// 如果解析过程中遇到无效的WHEN或OTHERWISE语句，则返回错误。
func parseCondChoose(calls []*sqlCall) (types.Cond, error) {
	// 初始化WHEN语句切片和OTHERWISE语句变量。
	var (
		stmts     = make([]*types.WhenStmt, 0, len(calls))
		otherwise string
	)

	// 遍历调用切片，从第二个元素开始，因为第一个元素是无效的。
	for i := 1; i < len(calls); i++ {
		call := calls[i]

		// 根据调用的函数名称处理不同的情况。
		switch call.funcName {
		case types.SQLOperateFuncWhen:
			// 检查WHEN语句的参数数量是否正确。
			if len(call.args) != 2 {
				return nil, errors.Errorf("invalid choose cond, when stmt is invalid, %d", i)
			}

			// 解析并验证第一个参数（条件表达式）。
			arg1, ok := call.args[0].(*ast.BinaryExpr)
			if !ok {
				return nil, errors.Errorf("invalid choose cond, cond stmt is invalid, %d", i)
			}

			// 解析并验证第二个参数（SQL语句）。
			arg2, ok := call.args[1].(*ast.BasicLit)
			if !ok || arg2.Kind != token.STRING {
				return nil, errors.Errorf("invalid choose cond, sql is invalid, %d", i)
			}

			// 创建并添加WHEN语句到切片中。
			whenStmt := types.NewWhenStmt(arg1, arg2.Value)
			stmts = append(stmts, whenStmt)

		case types.SQLOperateFuncOtherwise:
			// 检查OTHERWISE语句的参数数量是否正确。
			if len(call.args) != 1 {
				return nil, errors.Errorf("invalid choose cond, otherwise stmt is invalid, %d", i)
			}

			// 解析并验证OTHERWISE语句的参数（默认SQL语句）。
			arg, ok := call.args[0].(*ast.BasicLit)
			if !ok || arg.Kind != token.STRING || arg.Value == "" {
				return nil, errors.Errorf("invalid choose cond, otherwise stmt is invalid, %d", i)
			}

			// 设置OTHERWISE语句的值。
			otherwise = arg.Value
		}
	}

	// 使用解析的WHEN语句和OTHERWISE语句创建并返回一个条件选择语句。
	return types.NewChooseStmt(stmts, otherwise), nil
}

func parseSqlOperate(sc *sqlCall) (types.SQL, error) {
	fn, ok := sqlParseFuncSet[sc.funcName]
	if !ok {
		return nil, errors.Errorf("%s operate not found", sc.funcName)
	}

	return fn(sc)
}
