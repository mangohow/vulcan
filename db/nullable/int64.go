package nullable

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type Int64 struct {
	sql.NullInt64
}

func (i *Int64) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		i.Valid = false
		return nil
	}

	// 处理字符串
	var val int64
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	i.Int64 = val
	i.Valid = true
	return nil
}

func (i *Int64) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(i.Int64)
}

func (i *Int64) IsNull() bool {
	return i.Valid == false
}

func (i *Int64) IsZero() bool {
	return !i.Valid && i.Int64 == 0
}

func (i *Int64) Ptr() *int64 {
	if i.Valid {
		return &i.Int64
	}

	return nil
}

func (i *Int64) GetOrElse(value int64) int64 {
	if i.Valid {
		return i.Int64
	}

	return value
}

func (i *Int64) String() string {
	if i.Valid {
		return "<nil>"
	}

	return strconv.Itoa(int(i.Int64))
}

// StringFrom 创建 Valid=true 的 NullString
func Int64From(i int64) Int64 {
	return Int64{
		NullInt64: sql.NullInt64{
			Int64: i,
			Valid: true,
		},
	}
}

// StringFromPtr 从 *string 创建 NullString
func Int64FromPtr(i *int64) Int64 {
	if i == nil {
		return Int64{
			NullInt64: sql.NullInt64{
				Valid: false,
			},
		}
	}
	return Int64From(*i)
}
