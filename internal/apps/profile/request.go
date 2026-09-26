package profile

import (
	"encoding/json"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/apperr"
)

// UpdateRequest replaces the whole profile document. The profile is a free-form
// JSON object (see the frontend's Profile type for the expected shape).
type UpdateRequest struct {
	Profile json.RawMessage `json:"profile"`
}

func (r *UpdateRequest) Validate() error {
	var obj map[string]any
	if err := json.Unmarshal(r.Profile, &obj); err != nil || obj == nil {
		return apperr.Invalid("profile must be a JSON object")
	}
	return nil
}
