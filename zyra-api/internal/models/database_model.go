package models

import (
	"time"
)

type Database struct {
	Engine                       string     `json:"engine" gorm:"not null;default:oracle;index"`
	MaxLagSequences              *int64     `json:"max_lag_sequences"`
	ObservedUniqueID             string     `json:"observed_unique_id"`
	ObservedUniqueIDAssessmentID string     `json:"observed_unique_id_assessment_id"`
	ObservedUniqueIDReceivedAt   *time.Time `json:"observed_unique_id_received_at"`
	ID                           string     `json:"id"`
	ClientID                     string     `json:"client_id"`
	Name                         string     `json:"name"`
	Hostname                     string     `json:"hostname"`
	IP                           string     `json:"ip"`
	Notes                        string     `json:"notes"`
	Archived                     bool       `json:"archived"`
	CreatedAt                    time.Time  `json:"created_at"`
}
