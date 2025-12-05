package nullable

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Time struct {
	sql.NullTime
}

func (t *Time) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		t.Valid = false
		return nil
	}

	// 处理时间
	var val time.Time
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	t.Time = val
	t.Valid = true
	return nil
}

func (t *Time) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time)
}

func (t *Time) IsNull() bool {
	return t.Valid == false
}

func (t *Time) IsZero() bool {
	return !t.Valid && t.Time.IsZero()
}

func (t *Time) Ptr() *time.Time {
	if t.Valid {
		return &t.Time
	}

	return nil
}

func (t *Time) GetOrElse(value time.Time) time.Time {
	if t.Valid {
		return t.Time
	}

	return value
}

func (t *Time) String() string {
	if !t.Valid {
		return "<nil>"
	}

	return t.Time.Format(time.DateTime)
}

// TimeFrom 创建 Valid=true 的 Time
func TimeFrom(t time.Time) Time {
	return Time{
		NullTime: sql.NullTime{
			Time:  t,
			Valid: true,
		},
	}
}

// TimeFromPtr 从 *time.Time 创建 Time
func TimeFromPtr(t *time.Time) Time {
	if t == nil {
		return Time{
			NullTime: sql.NullTime{
				Valid: false,
			},
		}
	}
	return TimeFrom(*t)
}
