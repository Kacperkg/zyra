package models

import "time"

type EmailSource struct {
	Weekdays       []string   `json:"weekdays" gorm:"serializer:json;type:jsonb;not null;default:'[\"mon\",\"tue\",\"wed\",\"thu\",\"fri\",\"sat\",\"sun\"]'"`
	EffectiveAt    *time.Time `json:"effective_at"`
	Mailbox        string     `json:"mailbox"`
	AssessmentType string     `json:"assessment_type" gorm:"not null;default:daily_check"`
	ID             string     `json:"id"`
	DatabaseID     string     `json:"database_id" gorm:"index"`
	Sender         string     `json:"sender"`
	Subject        string     `json:"subject"`
	Timezone       string     `json:"timezone"`
	Enabled        bool       `json:"enabled" gorm:"index"`
	Schedules      []Schedule `json:"schedules" gorm:"serializer:json;type:jsonb"`
}

type Schedule struct {
	Name  string `json:"name"`
	Start string `json:"start"`
	End   string `json:"end"`
}
