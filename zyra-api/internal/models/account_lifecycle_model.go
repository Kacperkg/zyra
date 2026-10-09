package models

import "time"

// AccountLifecycle records transitions without exposing or copying user content.
type AccountLifecycle struct {
	ID             string        `json:"id" gorm:"primaryKey"`
	ActorID        string        `json:"actor_id"`
	TargetID       string        `json:"target_id" gorm:"index:idx_account_lifecycle_target_time,priority:1"`
	PreviousStatus AccountStatus `json:"previous_status"`
	Status         AccountStatus `json:"status"`
	PreviousRole   Role          `json:"previous_role"`
	Role           Role          `json:"role"`
	CreatedAt      time.Time     `json:"created_at" gorm:"index:idx_account_lifecycle_target_time,priority:2"`
}
