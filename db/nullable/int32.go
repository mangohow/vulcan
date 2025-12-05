package nullable

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type Int32 struct {
	sql.NullInt32
}

func (i *Int32) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		i.Valid = false
		return nil
	}

	// 处理整数
	var val int32
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	i.Int32 = val
	i.Valid = true
	return nil
}

func (i *Int32) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(i.Int32)
}

func (i *Int32) IsNull() bool {
	return i.Valid == false
}

func (i *Int32) IsZero() bool {
	return !i.Valid && i.Int32 == 0
}

func (i *Int32) Ptr() *int32 {
	if i.Valid {
		return &i.Int32
	}

	return nil
}

func (i *Int32) GetOrElse(value int32) int32 {
	if i.Valid {
		return i.Int32
	}

	return value
}

func (i *Int32) String() string {
	if !i.Valid {
		return "<nil>"
	}

	return strconv.FormatInt(int64(i.Int32), 10)
}

// Int32From 创建 Valid=true 的 Int32
func Int32From(i int32) Int32 {
	return Int32{
		NullInt32: sql.NullInt32{
			Int32: i,
			Valid: true,
		},
	}
}

// Int32FromPtr 从 *int32 创建 Int32
func Int32FromPtr(i *int32) Int32 {
	if i == nil {
		return Int32{
			NullInt32: sql.NullInt32{
				Valid: false,
			},
		}
	}
	return Int32From(*i)
}
