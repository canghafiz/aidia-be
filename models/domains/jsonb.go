package domains

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONB represents a JSONB field in PostgreSQL
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSONB: value is not []byte")
	}

	// Unmarshal into alias type to avoid infinite recursion
	type jsonbAlias JSONB
	return json.Unmarshal(bytes, (*jsonbAlias)(j))
}

// MarshalJSON implements json.Marshaler interface
func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(j))
}

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("cannot unmarshal into nil JSONB")
	}

	// Unmarshal into alias type to avoid infinite recursion
	type jsonbAlias JSONB
	return json.Unmarshal(data, (*jsonbAlias)(j))
}
