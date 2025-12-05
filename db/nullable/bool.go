package nullable

import (
	"database/sql"
	"encoding/json"
)

type Bool struct {
	sql.NullBool
}

func (b *Bool) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		b.Valid = false
		return nil
	}

	// 处理布尔值
	var val bool
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	b.Bool = val
	b.Valid = true
	return nil
}

func (b *Bool) MarshalJSON() ([]byte, error) {
	if !b.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(b.Bool)
}

func (b *Bool) IsNull() bool {
	return b.Valid == false
}

func (b *Bool) IsZero() bool {
	return !b.Valid && b.Bool == false
}

func (b *Bool) Ptr() *bool {
	if b.Valid {
		return &b.Bool
	}

	return nil
}

func (b *Bool) GetOrElse(value bool) bool {
	if b.Valid {
		return b.Bool
	}

	return value
}

func (b *Bool) String() string {
	if !b.Valid {
		return "<nil>"
	}

	if b.Bool {
		return "true"
	}
	return "false"
}

// BoolFrom 创建 Valid=true 的 Bool
func BoolFrom(b bool) Bool {
	return Bool{
		NullBool: sql.NullBool{
			Bool:  b,
			Valid: true,
		},
	}
}

// BoolFromPtr 从 *bool 创建 Bool
func BoolFromPtr(b *bool) Bool {
	if b == nil {
		return Bool{
			NullBool: sql.NullBool{
				Valid: false,
			},
		}
	}
	return BoolFrom(*b)
}
