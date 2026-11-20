package domain

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           uuid.UUID
	ClientID         *uuid.UUID
	RefreshTokenHash string
	Audience         string
	Scope            string
	ExpiresAt        time.Time
	IsRevoked        bool
	CreatedAt        time.Time
}

func (Token) TableName() string {
	return "refresh_tokens"
}
