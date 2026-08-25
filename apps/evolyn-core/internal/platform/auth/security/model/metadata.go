package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

func (m EventMetadata) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m *EventMetadata) Scan(v interface{}) error {
	if v == nil {
		*m = EventMetadata{}
		return nil
	}
	switch data := v.(type) {
	case []byte:
		return json.Unmarshal(data, m)
	case string:
		return json.Unmarshal([]byte(data), m)
	default:
		return fmt.Errorf("cannot scan %T into EventMetadata", v)
	}
}
