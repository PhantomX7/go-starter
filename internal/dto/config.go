package dto

// ConfigUpdateRequest defines the structure for updating a config. IsPublic
// is a pointer so an omitted field preserves the current visibility.
type ConfigUpdateRequest struct {
	Value    string `json:"value" form:"value" binding:"required"`
	IsPublic *bool  `json:"is_public" form:"is_public"`
}

// ConfigResponse defines the structure for config response
type ConfigResponse struct {
	ID       uint   `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsPublic bool   `json:"is_public"`
}
