package repository

import (
	"context"
	"gorm.io/gorm"
	"zyra-api/internal/models"
)

// No update/delete API: account audit entries are append-only.
type AccountLifecycleRepository interface {
	Create(context.Context, *models.AccountLifecycle) error
}
type accountLifecycleRepository struct {
	entity[models.AccountLifecycle]
}

func NewAccountLifecycleRepository(db *gorm.DB) AccountLifecycleRepository {
	return &accountLifecycleRepository{entity[models.AccountLifecycle]{db: db}}
}
func (s *store) AccountLifecycle() AccountLifecycleRepository {
	return NewAccountLifecycleRepository(s.db)
}
