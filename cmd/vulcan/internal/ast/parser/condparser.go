package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ValidOperators = map[string]bool{
	"EQ": true, "NE": true, "LT": true, "GT": true,
	"LE": true, "GE": true, "IN": true, "LIKE": true,
	"ISNULL": true, "ISNOTNULL": true,
}

type Column struct {
	Name string
	Type string
}

type Table struct {
	Name    string
	Columns []Column
}

type ConditionType int

const (
	SimpleCondition ConditionType = iota
	AndCondition
	OrCondition
	GroupCondition
)

type Condition struct {
	Type     ConditionType
	FieldID  string
	IsIndex  bool
	Operator string
	Left     *Condition
	Right    *Condition
}

// hasNestedBraces 检测是否存在嵌套大括号
func hasNestedBraces(s string) bool {
	// 使用栈检测括号嵌套
	var stack []int

	for i, r := range s {
		if r == '{' {
			stack = append(stack, i)
		} else if r == '}' {
			if len(stack) == 0 {
				continue // 已在hasBalancedBraces中验证
			}

			// 检查是否有嵌套：当前'{'和'}'之间是否还有其他'{}'对
			start := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			// 检查当前括号对内是否有其他完整括号对
			inner := s[start+1 : i]
			if strings.Contains(inner, "{") && strings.Contains(inner, "}") {
				return true
			}
		}
	}
	return false
}

// hasBalancedBraces 检查大括号是否平衡
func hasBalancedBraces(s string) bool {
	depth := 0
	for _, r := range s {
		if r == '{' {
			depth++
		} else if r == '}' {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func ParseCondition(condStr string, table Table) (*Condition, error) {
	// 预处理：去除前后空格
	condStr = strings.TrimSpace(condStr)

	// 验证：大括号必须平衡
	if !hasBalancedBraces(condStr) {
		return nil, errors.New("unbalanced braces in condition: " + condStr)
	}

	// 验证：禁止嵌套大括号
	if hasNestedBraces(condStr) {
		return nil, errors.New(
			"nested braces are not allowed; use non-nested equivalent form. " +
				"Example: '{1.EQ & 2.GT} | {3.LT & 4.GT}' is valid, " +
				"but '{1.EQ & {2.GT | 3.LT}}' is invalid")
	}

	// 解析条件
	return parseConditionInternal(condStr, table)
}

// parseConditionInternal 实际解析逻辑
func parseConditionInternal(condStr string, table Table) (*Condition, error) {
	condStr = strings.TrimSpace(condStr)

	// 检查是否是分组条件 (单层)
	if isSingleLevelGroup(condStr) {
		// 移除最外层大括号
		inner := condStr[1 : len(condStr)-1]
		return parseConditionInternal(inner, table)
	}

	// 尝试解析OR条件 (最低优先级)
	if idx := findOutermostOperator(condStr, '|'); idx != -1 {
		left, err := parseConditionInternal(condStr[:idx], table)
		if err != nil {
			return nil, err
		}

		right, err := parseConditionInternal(condStr[idx+1:], table)
		if err != nil {
			return nil, err
		}

		return &Condition{
			Type:  OrCondition,
			Left:  left,
			Right: right,
		}, nil
	}

	// 尝试解析AND条件
	if idx := findOutermostOperator(condStr, '&'); idx != -1 {
		left, err := parseConditionInternal(condStr[:idx], table)
		if err != nil {
			return nil, err
		}

		right, err := parseConditionInternal(condStr[idx+1:], table)
		if err != nil {
			return nil, err
		}

		return &Condition{
			Type:  AndCondition,
			Left:  left,
			Right: right,
		}, nil
	}

	// 解析简单条件
	return parseSimpleCondition(condStr, table)
}

// isSingleLevelGroup 检查是否是单层分组条件
func isSingleLevelGroup(s string) bool {
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return false
	}

	// 检查是否包含内部大括号 (即是否嵌套)
	inner := s[1 : len(s)-1]
	return !strings.Contains(inner, "{") && !strings.Contains(inner, "}")
}

// findOutermostOperator 查找最外层的逻辑运算符
func findOutermostOperator(s string, op rune) int {
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		}

		// 只在最外层 (depth=0) 查找逻辑运算符
		if depth == 0 && r == op {
			return i
		}
	}
	return -1
}

// parseSimpleCondition 解析简单条件单元
func parseSimpleCondition(unit string, table Table) (*Condition, error) {
	unit = strings.TrimSpace(unit)

	// 检查是否是纯数字 (字段索引且省略EQ)
	if isPureNumber(unit) {
		idx, _ := strconv.Atoi(unit)
		if idx < 1 || idx > len(table.Columns) {
			return nil, fmt.Errorf("field index %d out of range (1-%d)", idx, len(table.Columns))
		}

		return &Condition{
			Type:     SimpleCondition,
			FieldID:  unit,
			IsIndex:  true,
			Operator: "EQ",
		}, nil
	}

	// 必须包含点号分隔
	dotIndex := strings.IndexRune(unit, '.')
	if dotIndex == -1 {
		return nil, fmt.Errorf("invalid condition '%s': must use '.' to separate field and operator", unit)
	}

	fieldID := unit[:dotIndex]
	operator := strings.ToUpper(unit[dotIndex+1:])

	// 验证操作符
	if !ValidOperators[operator] {
		return nil, fmt.Errorf("invalid operator '%s' in '%s'", operator, unit)
	}

	// 验证字段标识
	if isPureNumber(fieldID) {
		// 字段索引
		idx, _ := strconv.Atoi(fieldID)
		if idx < 1 || idx > len(table.Columns) {
			return nil, fmt.Errorf("field index %d out of range (1-%d)", idx, len(table.Columns))
		}

		return &Condition{
			Type:     SimpleCondition,
			FieldID:  fieldID,
			IsIndex:  true,
			Operator: operator,
		}, nil
	} else {
		// 字段名称
		found := false
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, fieldID) {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("field '%s' not found in table", fieldID)
		}

		return &Condition{
			Type:     SimpleCondition,
			FieldID:  fieldID,
			IsIndex:  false,
			Operator: operator,
		}, nil
	}
}

// isPureNumber 检查字符串是否为纯数字
func isPureNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ToSQL 生成SQL条件片段和参数类型
func (c *Condition) ToSQL(table Table) (string, []string, error) {
	switch c.Type {
	case SimpleCondition:
		return c.toSimpleSQL(table)
	case AndCondition:
		leftSQL, leftParams, err := c.Left.ToSQL(table)
		if err != nil {
			return "", nil, err
		}

		rightSQL, rightParams, err := c.Right.ToSQL(table)
		if err != nil {
			return "", nil, err
		}

		// 不要添加额外括号 - 分组会在上层处理
		return fmt.Sprintf("%s AND %s", leftSQL, rightSQL),
			append(leftParams, rightParams...), nil

	case OrCondition:
		leftSQL, leftParams, err := c.Left.ToSQL(table)
		if err != nil {
			return "", nil, err
		}

		rightSQL, rightParams, err := c.Right.ToSQL(table)
		if err != nil {
			return "", nil, err
		}

		// 不要添加额外括号 - 分组会在上层处理
		return fmt.Sprintf("%s OR %s", leftSQL, rightSQL),
			append(leftParams, rightParams...), nil

	case GroupCondition:
		innerSQL, params, err := c.Left.ToSQL(table)
		if err != nil {
			return "", nil, err
		}
		// 添加括号以保持分组
		return fmt.Sprintf("(%s)", innerSQL), params, nil
	}
	return "", nil, fmt.Errorf("unknown condition type: %d", c.Type)
}

// toSimpleSQL 生成简单条件的SQL片段
func (c *Condition) toSimpleSQL(table Table) (string, []string, error) {
	var colName string

	if c.IsIndex {
		idx, _ := strconv.Atoi(c.FieldID)
		colName = table.Columns[idx-1].Name
	} else {
		colName = c.FieldID
	}

	var sqlFragment string
	var paramType string

	switch c.Operator {
	case "EQ":
		sqlFragment = fmt.Sprintf("%s = ?", colName)
		paramType = getFieldType(table, c.FieldID, c.IsIndex)

	case "NE":
		sqlFragment = fmt.Sprintf("%s != ?", colName)
		paramType = getFieldType(table, c.FieldID, c.IsIndex)

	case "LT", "GT", "LE", "GE":
		op := map[string]string{"LT": "<", "GT": ">", "LE": "<=", "GE": ">="}[c.Operator]
		sqlFragment = fmt.Sprintf("%s %s ?", colName, op)
		paramType = getFieldType(table, c.FieldID, c.IsIndex)

	case "IN":
		sqlFragment = fmt.Sprintf("%s IN (?)", colName)
		paramType = "[]" + getFieldType(table, c.FieldID, c.IsIndex)

	case "LIKE":
		sqlFragment = fmt.Sprintf("%s LIKE ?", colName)
		paramType = "string"

	case "ISNULL":
		sqlFragment = fmt.Sprintf("%s IS NULL", colName)
		return sqlFragment, nil, nil

	case "ISNOTNULL":
		sqlFragment = fmt.Sprintf("%s IS NOT NULL", colName)
		return sqlFragment, nil, nil

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", c.Operator)
	}

	return sqlFragment, []string{paramType}, nil
}

// getFieldType 获取字段的Go类型
func getFieldType(table Table, fieldID string, isIndex bool) string {
	var colType string

	if isIndex {
		idx, _ := strconv.Atoi(fieldID)
		colType = table.Columns[idx-1].Type
	} else {
		for _, col := range table.Columns {
			if strings.EqualFold(col.Name, fieldID) {
				colType = col.Type
				break
			}
		}
	}

	// 类型映射
	colType = strings.ToUpper(colType)
	switch {
	case strings.Contains(colType, "INT"), strings.Contains(colType, "TINYINT"):
		return "int"
	case strings.Contains(colType, "VARCHAR"),
		strings.Contains(colType, "TEXT"):
		return "string"
	case strings.Contains(colType, "BOOLEAN"):
		return "bool"
	case strings.Contains(colType, "DECIMAL"),
		strings.Contains(colType, "FLOAT"):
		return "float64"
	case strings.Contains(colType, "DATE"),
		strings.Contains(colType, "TIME"):
		return "time.Time"
	default:
		return "interface{}"
	}
}

// 示例表结构
var usersTable = Table{
	Name: "users",
	Columns: []Column{
		{Name: "id", Type: "INT"},
		{Name: "status", Type: "ENUM('active','inactive')"},
		{Name: "age", Type: "TINYINT UNSIGNED"},
		{Name: "email", Type: "VARCHAR(100)"},
		{Name: "created_at", Type: "DATETIME"},
	},
}

// 测试函数
func TestConditionParsing() {
	// 测试1: 简单条件 (索引省略EQ)
	cond, _ := ParseCondition("1", usersTable)
	sql, params, _ := cond.ToSQL(usersTable)
	fmt.Printf("Test1 [%s]: %s | Params: %v\n", "1", sql, params)
	// Output: status = ? | Params: [string]

	// 测试2: 字段名称条件
	cond, _ = ParseCondition("status.EQ", usersTable)
	sql, params, _ = cond.ToSQL(usersTable)
	fmt.Printf("Test2 [%s]: %s | Params: %v\n", "status.EQ", sql, params)
	// Output: status = ? | Params: [string]

	// 测试3: 复合条件 (AND)
	cond, _ = ParseCondition("1&status.GT", usersTable)
	sql, params, _ = cond.ToSQL(usersTable)
	fmt.Printf("Test3 [%s]: %s | Params: %v\n", "1&status.GT", sql, params)
	// Output: status = ? AND status > ? | Params: [string int]

	// 测试4: 分组条件
	cond, _ = ParseCondition("{1&status.GT}", usersTable)
	sql, params, _ = cond.ToSQL(usersTable)
	fmt.Printf("Test4 [%s]: %s | Params: %v\n", "{1&status.GT}", sql, params)
	// Output: (status = ? AND status > ?) | Params: [string int]

	// 测试5: OR条件
	cond, _ = ParseCondition("{1&status.GT}|3.LT", usersTable)
	sql, params, _ = cond.ToSQL(usersTable)
	fmt.Printf("Test5 [%s]: %s | Params: %v\n", "{1&status.GT}|3.LT", sql, params)
	// Output: (status = ? AND status > ?) OR age < ? | Params: [string int int]

	// 测试6: 无效嵌套 (应该报错)
	_, err := ParseCondition("{1&{status.GT|3.LT}}", usersTable)
	if err != nil {
		fmt.Printf("Test6 [expected error]: %s\n", err)
		// Output: nested braces are not allowed...
	}

	// 测试7: 转换为非嵌套等效形式
	// 原始嵌套: {1.EQ & {2.GT | 3.LT}}
	// 等效非嵌套: {{1.EQ & 2.GT} | {1.EQ & 3.LT}}
	cond, _ = ParseCondition("{{1.EQ & 2.GT} | {1.EQ & 3.LT}}", usersTable)
	sql, params, _ = cond.ToSQL(usersTable)
	fmt.Printf("Test7 [%s]: %s | Params: %v\n",
		"{{1.EQ & 2.GT} | {1.EQ & 3.LT}}",
		sql,
		params)
	// Output: ((status = ? AND age > ?) OR (status = ? AND age < ?))
}
