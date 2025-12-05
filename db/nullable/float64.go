package nullable

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type Float64 struct {
	sql.NullFloat64
}

func (f *Float64) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		f.Valid = false
		return nil
	}

	// 处理浮点数
	var val float64
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	f.Float64 = val
	f.Valid = true
	return nil
}

func (f *Float64) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float64)
}

func (f *Float64) IsNull() bool {
	return f.Valid == false
}

func (f *Float64) IsZero() bool {
	return !f.Valid && f.Float64 == 0
}

func (f *Float64) Ptr() *float64 {
	if f.Valid {
		return &f.Float64
	}

	return nil
}

func (f *Float64) GetOrElse(value float64) float64 {
	if f.Valid {
		return f.Float64
	}

	return value
}

func (f *Float64) String() string {
	if !f.Valid {
		return "<nil>"
	}

	return strconv.FormatFloat(f.Float64, 'f', -1, 64)
}

// Float64From 创建 Valid=true 的 Float64
func Float64From(f float64) Float64 {
	return Float64{
		NullFloat64: sql.NullFloat64{
			Float64: f,
			Valid:   true,
		},
	}
}

// Float64FromPtr 从 *float64 创建 Float64
func Float64FromPtr(f *float64) Float64 {
	if f == nil {
		return Float64{
			NullFloat64: sql.NullFloat64{
				Valid: false,
			},
		}
	}
	return Float64From(*f)
}
