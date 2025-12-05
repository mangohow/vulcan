package nullable

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type Int16 struct {
	sql.NullInt16
}

func (i *Int16) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		i.Valid = false
		return nil
	}

	// 处理整数
	var val int16
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	i.Int16 = val
	i.Valid = true
	return nil
}

func (i *Int16) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(i.Int16)
}

func (i *Int16) IsNull() bool {
	return i.Valid == false
}

func (i *Int16) IsZero() bool {
	return !i.Valid && i.Int16 == 0
}

func (i *Int16) Ptr() *int16 {
	if i.Valid {
		return &i.Int16
	}

	return nil
}

func (i *Int16) GetOrElse(value int16) int16 {
	if i.Valid {
		return i.Int16
	}

	return value
}

func (i *Int16) String() string {
	if !i.Valid {
		return "<nil>"
	}

	return strconv.FormatInt(int64(i.Int16), 10)
}

// Int16From 创建 Valid=true 的 Int16
func Int16From(i int16) Int16 {
	return Int16{
		NullInt16: sql.NullInt16{
			Int16: i,
			Valid: true,
		},
	}
}

// Int16FromPtr 从 *int16 创建 Int16
func Int16FromPtr(i *int16) Int16 {
	if i == nil {
		return Int16{
			NullInt16: sql.NullInt16{
				Valid: false,
			},
		}
	}
	return Int16From(*i)
}
