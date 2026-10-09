package repository

import (
	"context"
	"zyra-api/internal/models"
)

type CommentRepository interface {
	Get(context.Context, *models.Comment, string, bool) error
	Create(context.Context, *models.Comment) error
	Save(context.Context, *models.Comment) error
	FindByIDs(context.Context, []string) ([]models.Comment, error)
}
type commentRepository struct{ entity[models.Comment] }

func (s *store) Comments() CommentRepository {
	return &commentRepository{entity[models.Comment]{db: s.db}}
}
func (r *commentRepository) FindByIDs(ctx context.Context, ids []string) ([]models.Comment, error) {
	rows := []models.Comment{}
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", ids).Find(&rows).Error
	return rows, translate(err)
}
