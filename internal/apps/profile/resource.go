package profile

import (
	"encoding/json"
	"time"
)

type Resource struct {
	Profile   json.RawMessage `json:"profile"`
	UpdatedAt time.Time       `json:"updated_at"`
}
