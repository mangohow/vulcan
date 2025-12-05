//go:build 1.22

package nullable

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Value[T any] struct {
	sql.Null[T]
}

func (v *Value[T]) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		v.Valid = false
		return nil
	}

	// 处理字符串
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	v.V = val
	v.Valid = true
	return nil
}

func (v *Value[T]) MarshalJSON() ([]byte, error) {
	if !v.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(v.V)
}

func (v *Value[T]) IsNull() bool {
	return v.Valid == false
}

func (v *Value[T]) Ptr() *T {
	if v.Valid {
		return &v.V
	}

	return nil
}

func (v *Value[T]) GetOrElse(value T) T {
	if v.Valid {
		return v.V
	}

	return value
}

func (v *Value[T]) String() string {
	if !v.Valid {
		return "<nil>"
	}

	// 将泛型值转为 interface{} 以便进行类型判断
	switch val := any(v.V).(type) {
	// 处理有符号整数类型
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)

	// 处理无符号整数类型
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)

	// 处理浮点数类型
	case float32:
		// 使用较小的精度避免过长的小数
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)

	// 处理字符串和字符类型
	case string:
		return val
	case []byte:
		return string(val)

	// 处理布尔类型
	case bool:
		if val {
			return "true"
		}
		return "false"

	// 处理时间类型
	case time.Time:
		return val.Format(time.DateTime)

	// 处理复合类型 - 使用 JSON 编码
	default:
		res, err := v.MarshalJSON()
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(res)
	}
}

// ValueFrom 创建 Valid=true 的 Value[T]
func ValueFrom[T any](s T) Value[T] {
	return Value[T]{
		Null: sql.Null[T]{
			V:     s,
			Valid: true,
		},
	}
}

// ValueFromPtr 从 *string 创建 Value[T]
func ValueFromPtr[T any](s *T) Value[T] {
	if s == nil {
		return Value[T]{
			Null: sql.Null[T]{
				Valid: false,
			},
		}
	}
	return ValueFrom(*s)
}
