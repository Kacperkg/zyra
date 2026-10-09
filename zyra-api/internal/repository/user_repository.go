package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"zyra-api/internal/models"
)

type UserRepository interface {
	Get(context.Context, *models.User, string, bool) error
	Find(context.Context, *[]models.User, Query) error
	Count(context.Context, Query) (int64, error)
	Create(context.Context, *models.User) error
	Save(context.Context, *models.User) error
	FindByIDs(context.Context, []string) ([]models.User, error)
	LockIDs(context.Context, []string) (map[string]models.User, error)
}
type userRepository struct{ entity[models.User] }

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{entity[models.User]{db: db}}
}
func (s *store) Users() UserRepository { return NewUserRepository(s.db) }

// Lock users in stable order before domain rows, avoiding crossed mention/edit locks.
func (r *userRepository) LockIDs(ctx context.Context, ids []string) (map[string]models.User, error) {
	rows := []models.User{}
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Order("id ASC").Find(&rows).Error
	result := make(map[string]models.User, len(rows))
	for _, u := range rows {
		result[u.ID] = u
	}
	return result, translate(err)
}

func (r *userRepository) FindByIDs(ctx context.Context, ids []string) ([]models.User, error) {
	users := []models.User{}
	if len(ids) == 0 {
		return users, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, translate(err)
}
