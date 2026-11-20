package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name             string
	ClientSecretHash string
	IsConfidential   bool
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Grants           []ClientGrant         `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`
	RedirectURIs     []ClientRedirectURI   `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`
	Scopes           []ClientScope         `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`
	Audiences        []ClientAudience      `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`
	AccessContexts   []ClientAccessContext `gorm:"foreignKey:ClientID;constraint:OnDelete:CASCADE"`
}
