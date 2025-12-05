package nullable

import (
	"database/sql"
	"encoding/json"
	"unsafe"
)

type String struct {
	sql.NullString
}

func (s *String) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		s.Valid = false
		return nil
	}

	// 处理字符串
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	s.String = str
	s.Valid = true
	return nil
}

func (s *String) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(s.String)
}

func (s *String) IsNull() bool {
	return s.Valid == false
}

func (s *String) IsZero() bool {
	return !s.Valid && s.String == ""
}

func (s *String) Ptr() *string {
	if s.Valid {
		return &s.String
	}

	return nil
}

// StringFrom 创建 Valid=true 的 NullString
func StringFrom(s string) String {
	return String{
		NullString: sql.NullString{
			String: s,
			Valid:  true,
		},
	}
}

// StringFromPtr 从 *string 创建 NullString
func StringFromPtr(s *string) String {
	if s == nil {
		return String{
			NullString: sql.NullString{
				Valid: false,
			},
		}
	}
	return StringFrom(*s)
}

func b2s(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}
