package models

import (
	"time"
)

type Role string

const (
	RoleOwner   Role = "owner"
	RoleAdmin   Role = "admin"
	RoleTrusted Role = "trusted"
	RoleNormal  Role = "normal"
)

func (r Role) Valid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleTrusted || r == RoleNormal
}
func (r Role) CanConfigure() bool  { return r == RoleOwner || r == RoleAdmin || r == RoleTrusted }
func (r Role) CanAdminister() bool { return r == RoleOwner || r == RoleAdmin }

type AccountStatus string

const (
	StatusActive  AccountStatus = "active"
	StatusRetired AccountStatus = "retired"
)

type User struct {
	ID           string        `json:"id"`
	Email        string        `json:"email" gorm:"uniqueIndex;not null"`
	Name         string        `json:"name"`
	Role         Role          `json:"role"`
	Theme        string        `json:"theme"`
	Status       AccountStatus `json:"status" gorm:"not null;default:active"`
	AvatarURL    string        `json:"avatar_url"`
	Appearance   string        `json:"appearance" gorm:"not null;default:modern"`
	PasswordHash string        `json:"-"`
	CreatedAt    time.Time     `json:"created_at"`
}
