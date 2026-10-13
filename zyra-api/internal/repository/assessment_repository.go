package repository

import (
	"context"
	"gorm.io/gorm"
	"time"
	"zyra-api/internal/models"
)

type AssessmentRepository interface {
	Get(context.Context, *models.Assessment, string, bool) error
	Find(context.Context, *[]models.Assessment, Query) error
	Count(context.Context, Query) (int64, error)
	ListSummaries(context.Context, Query) ([]AssessmentSummary, error)
	Exists(context.Context, Query) (bool, error)
	Create(context.Context, *models.Assessment) error
	Save(context.Context, *models.Assessment) error
}
type assessmentRepository struct{ entity[models.Assessment] }

func NewAssessmentRepository(db *gorm.DB) AssessmentRepository {
	return &assessmentRepository{entity[models.Assessment]{db: db}}
}
func (s *store) Assessments() AssessmentRepository { return NewAssessmentRepository(s.db) }

func (r *assessmentRepository) Exists(ctx context.Context, q Query) (bool, error) {
	var row struct{ ID string }
	q.Limit = 1
	err := query(r.db.WithContext(ctx).Model(&models.Assessment{}), q).Select("id").Scan(&row).Error
	return row.ID != "", translate(err)
}

type AssessmentTicketSummary struct {
	ID           string    `json:"id"`
	Number       int64     `json:"number"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	CheckType    string    `json:"check_type"`
	CreatedAt    time.Time `json:"created_at"`
	AssessmentID string    `json:"-"`
	TicketCount  int64     `json:"-"`
}
type AssessmentSummary struct {
	ID          string                    `json:"id"`
	DatabaseID  string                    `json:"database_id"`
	Type        string                    `json:"type"`
	ReceivedAt  time.Time                 `json:"received_at"`
	ReportTime  string                    `json:"report_time"`
	Hostname    string                    `json:"hostname"`
	IP          string                    `json:"ip"`
	UniqueID    string                    `json:"unique_id"`
	Result      string                    `json:"result"`
	TicketCount int64                     `json:"ticket_count"`
	Tickets     []AssessmentTicketSummary `json:"tickets" gorm:"-"`
}

func (r *assessmentRepository) ListSummaries(ctx context.Context, q Query) ([]AssessmentSummary, error) {
	rows := []AssessmentSummary{}
	if err := query(r.db.WithContext(ctx).Model(&models.Assessment{}), q).
		Select("id,database_id,type,received_at,report_time,hostname,ip,unique_id,result").Scan(&rows).Error; err != nil {
		return nil, translate(err)
	}
	ids := []string{}
	for i := range rows {
		ids = append(ids, rows[i].ID)
		rows[i].Tickets = []AssessmentTicketSummary{}
	}
	if len(ids) == 0 {
		return rows, nil
	}
	// Bound returned links without loading ticket evidence or growing queries per row.
	links := []AssessmentTicketSummary{}
	err := r.db.WithContext(ctx).Raw(`SELECT id,number,title,status,check_type,created_at,assessment_id,ticket_count FROM (
	 SELECT id,number,title,status,check_type,created_at,assessment_id,
	 count(*) OVER (PARTITION BY assessment_id) AS ticket_count,
	 row_number() OVER (PARTITION BY assessment_id ORDER BY created_at DESC,id DESC) AS position
	 FROM tickets WHERE assessment_id IN ?
	) ranked WHERE position <= 5 ORDER BY created_at DESC,id DESC`, ids).Scan(&links).Error
	if err != nil {
		return nil, translate(err)
	}
	positions := map[string]int{}
	for i := range rows {
		positions[rows[i].ID] = i
	}
	for _, link := range links {
		i := positions[link.AssessmentID]
		rows[i].TicketCount = link.TicketCount
		rows[i].Tickets = append(rows[i].Tickets, link)
	}
	return rows, nil
}
