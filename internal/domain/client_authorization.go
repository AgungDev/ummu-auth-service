package domain

import "github.com/google/uuid"

type ClientScope struct {
	ClientID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Scope    string    `gorm:"primaryKey"`
}

type ClientAudience struct {
	ClientID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Audience string    `gorm:"primaryKey"`
}

type ClientAccessContext struct {
	ClientID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	ScopeType string    `gorm:"primaryKey"`
	ScopeID   string    `gorm:"primaryKey"`
}
