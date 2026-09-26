package resume

import "time"

type Resource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int       `json:"size_bytes"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewResource(r *Resume) Resource {
	return Resource{ID: r.ID, Name: r.Name, Filename: r.Filename, ContentType: r.ContentType, Size: r.Size, IsDefault: r.IsDefault, CreatedAt: r.CreatedAt}
}

func NewResources(list []Resume) []Resource {
	out := make([]Resource, 0, len(list))
	for i := range list {
		out = append(out, NewResource(&list[i]))
	}
	return out
}
