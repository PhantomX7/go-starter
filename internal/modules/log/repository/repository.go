package repository

import (
	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/repository"

	"gorm.io/gorm"
)

// LogRepository defines the interface for log repository operations
type LogRepository interface {
	repository.IRepository[models.Log]
}

// logRepository implements the LogRepository interface
type logRepository struct {
	repository.Repository[models.Log]
}

// NewLogRepository creates a new instance of LogRepository
func NewLogRepository(db *gorm.DB) LogRepository {
	return &logRepository{
		Repository: repository.Repository[models.Log]{
			DB: db,
		},
	}
}
