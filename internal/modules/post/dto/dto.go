package dto

type PostCreateRequest struct {
	Title   string `json:"title" form:"title" binding:"required"`
	Content string `json:"content" form:"content" binding:"required"`
}

type PostUpdateRequest struct {
	Title   string `json:"title" form:"title"`
	Content string `json:"content" form:"content"`
}
