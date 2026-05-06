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

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONB: unsupported type %T", value)
	}

	// Unmarshal into alias type to avoid infinite recursion
	type jsonbAlias JSONB
	return json.Unmarshal(b, (*jsonbAlias)(j))
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

// JSONBStringArray stores []string as PostgreSQL JSONB
type JSONBStringArray []string

func (a JSONBStringArray) Value() (driver.Value, error) {
	if a == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a)
}

func (a *JSONBStringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("failed to scan JSONBStringArray: unsupported type %T", value)
	}
}
