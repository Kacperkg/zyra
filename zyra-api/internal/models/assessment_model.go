package models

import (
	"time"
)

type CheckResult struct {
	CheckType string   `json:"check_type"`
	Resource  string   `json:"resource,omitempty"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Evidence  string   `json:"evidence"`
	Value     *float64 `json:"value,omitempty"`
	Threshold *float64 `json:"threshold,omitempty"`
}

type Assessment struct {
	UniqueID   string        `json:"unique_id"`
	ID         string        `json:"id" gorm:"index:idx_assessment_database_received,priority:3"`
	DatabaseID string        `json:"database_id" gorm:"index:idx_assessment_schedule,priority:1;index:idx_assessment_database_received,priority:1"`
	MessageID  string        `json:"message_id" gorm:"uniqueIndex;not null"`
	Type       string        `json:"type" gorm:"index:idx_assessment_type_received,priority:1"`
	Subject    string        `json:"subject" gorm:"index:idx_assessment_schedule,priority:3"`
	Sender     string        `json:"sender" gorm:"index:idx_assessment_schedule,priority:2"`
	ReceivedAt time.Time     `json:"received_at" gorm:"index:idx_assessment_schedule,priority:4;index;index:idx_assessment_type_received,priority:2;index:idx_assessment_result_received,priority:2;index:idx_assessment_database_received,priority:2"`
	ReportTime string        `json:"report_time"`
	Hostname   string        `json:"hostname"`
	IP         string        `json:"ip"`
	Result     string        `json:"result" gorm:"index:idx_assessment_result_received,priority:1"`
	RawBody    string        `json:"-"`
	Results    []CheckResult `json:"results" gorm:"serializer:json;type:jsonb"`
	Settings   CheckSettings `json:"settings" gorm:"serializer:json;type:jsonb"`
}
