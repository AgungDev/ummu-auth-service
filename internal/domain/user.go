package domain

import "github.com/google/uuid"

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string
	Email        string
	PasswordHash string
	FullName     string
	Status       string
	GoogleSub    *string `gorm:"type:text;uniqueIndex" json:"-"`
}

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}
