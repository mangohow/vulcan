package nullable

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type Byte struct {
	sql.NullByte
}

func (b *Byte) UnmarshalJSON(data []byte) error {
	// 处理 null
	if b2s(data) == "null" {
		b.Valid = false
		return nil
	}

	// 处理字节
	var val byte
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	b.Byte = val
	b.Valid = true
	return nil
}

func (b *Byte) MarshalJSON() ([]byte, error) {
	if !b.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(b.Byte)
}

func (b *Byte) IsNull() bool {
	return b.Valid == false
}

func (b *Byte) IsZero() bool {
	return !b.Valid && b.Byte == 0
}

func (b *Byte) Ptr() *byte {
	if b.Valid {
		return &b.Byte
	}

	return nil
}

func (b *Byte) GetOrElse(value byte) byte {
	if b.Valid {
		return b.Byte
	}

	return value
}

func (b *Byte) String() string {
	if !b.Valid {
		return "<nil>"
	}

	return strconv.FormatUint(uint64(b.Byte), 10)
}

// ByteFrom 创建 Valid=true 的 Byte
func ByteFrom(b byte) Byte {
	return Byte{
		NullByte: sql.NullByte{
			Byte:  b,
			Valid: true,
		},
	}
}

// ByteFromPtr 从 *byte 创建 Byte
func ByteFromPtr(b *byte) Byte {
	if b == nil {
		return Byte{
			NullByte: sql.NullByte{
				Valid: false,
			},
		}
	}
	return ByteFrom(*b)
}
