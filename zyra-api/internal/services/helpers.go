package services

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
)

func invalid(message string) error { return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, message) }
func requireConfigure(u models.User) error {
	if u.Status != models.StatusActive || !u.Role.CanConfigure() {
		return apperrors.ErrForbidden
	}
	return nil
}
func requireAdmin(u models.User) error {
	if u.Status != models.StatusActive || !u.Role.CanAdminister() {
		return apperrors.ErrForbidden
	}
	return nil
}
func emailAddress(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	a, e := mail.ParseAddress(v)
	if e != nil || a.Address != v {
		return "", invalid("valid email required")
	}
	return v, nil
}
func passwordHash(v string) (string, error) {
	if len(v) < 12 || len(v) > 72 {
		return "", invalid("password must be 12–72 bytes")
	}
	b, e := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
	return string(b), e
}
