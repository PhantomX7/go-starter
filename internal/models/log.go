package models

import (
	"github.com/PhantomX7/athleton/internal/dto"
)

type LogAction string

const (
	LogActionCreate         LogAction = "create"
	LogActionUpdate         LogAction = "update"
	LogActionDelete         LogAction = "delete"
	LogActionLogin          LogAction = "login"
	LogActionLogout         LogAction = "logout"
	LogActionImport         LogAction = "import"
	LogActionExport         LogAction = "export"
	LogActionChangePassword LogAction = "change_password"
)

const (
	LogEntityTypeAdminRole      = "admin_role"
	LogEntityTypeBanner         = "banner"
	LogEntityTypeBlog           = "blog"
	LogEntityTypeBlogCategory   = "blog_category"
	LogEntityTypeBrand          = "brand"
	LogEntityTypeCategory       = "category"
	LogEntityTypeConfig         = "config"
	LogEntityTypePcBuild        = "pc_build"
	LogEntityTypeProduct        = "product"
	LogEntityTypeFooterMenu     = "footer_menu"
	LogEntityTypeMenu           = "menu"
	LogEntityTypeSpecDefinition = "spec_definition"
	LogEntityTypeUser           = "user"
)

func (l LogAction) ToString() string {
	return string(l)
}

type Log struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     *uint     `json:"user_id" gorm:"index"`
	Action     LogAction `json:"action" gorm:"type:varchar(50);not null"`
	EntityType string    `json:"entity_type" gorm:"type:varchar(50);index"`
	EntityID   uint      `json:"entity_id" gorm:"index"`

	Message string `json:"message" gorm:"type:text"`
	// IPAddress string `json:"ip_address" gorm:"type:varchar(50)"`
	// UserAgent string `json:"user_agent" gorm:"type:text"`

	Timestamp

	// Relationships
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`

	// Entity References (loaded manually based on EntityType)
	// GORM polymorphic is for parent-to-child (has-one/has-many), not child-to-parent
	// These fields are populated manually in the service/repository layer
	AdminRole  *AdminRole `json:"admin_role,omitempty" gorm:"-"`
	Config     *Config    `json:"config,omitempty" gorm:"-"`
	TargetUser *User      `json:"target_user,omitempty" gorm:"-"`
}

func (l Log) ToResponse() dto.LogResponse {
	response := dto.LogResponse{
		ID:         l.ID,
		UserID:     l.UserID,
		Action:     l.Action.ToString(),
		EntityType: l.EntityType,
		EntityID:   l.EntityID,
		Message:    l.Message,
		CreatedAt:  l.CreatedAt,
	}

	if l.User != nil {
		userResp := l.User.ToResponse()
		response.User = userResp
	}

	if l.AdminRole != nil {
		resp := l.AdminRole.ToResponse()
		response.AdminRole = resp
	}
	if l.Config != nil {
		resp := l.Config.ToResponse()
		response.Config = resp
	}
	if l.TargetUser != nil {
		resp := l.TargetUser.ToResponse()
		response.TargetUser = resp
	}

	return response
}
