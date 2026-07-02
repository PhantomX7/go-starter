package dto

// ConfigUpdateRequest defines the structure for updating a config
type ConfigUpdateRequest struct {
	Value string `json:"value" form:"value" binding:"required"`
}

// ConfigResponse defines the structure for config response
type ConfigResponse struct {
	ID    uint   `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}
