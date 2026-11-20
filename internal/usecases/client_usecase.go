package usecases

import (
	"context"
	"fmt"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	"auth_service/internal/repositories"
	"auth_service/internal/security"
	loggerpkg "auth_service/pkg/logger"
	"github.com/google/uuid"
)

type clientUsecase struct {
	clientRepo repositories.ClientRepositoryInterface
	security   security.SecurityService
	logger     loggerpkg.Logger
}

// NewClientUsecase creates a new client usecase instance
func NewClientUsecase(clientRepo repositories.ClientRepositoryInterface, securityService security.SecurityService, logger loggerpkg.Logger) ClientUsecaseInterface {
	return &clientUsecase{clientRepo: clientRepo, security: securityService, logger: logger}
}

// Create creates a new client with hashed secret
func (uc *clientUsecase) Create(ctx context.Context, req dto.ClientRequest) (*domain.Client, error) {
	uc.logger.Info("", "", "", fmt.Sprintf("create client request received: name=%s", req.Name))
	if err := validateClientAuthorization(req); err != nil {
		return nil, err
	}

	// Hash client secret
	hashedSecret, err := uc.security.HashPassword(req.ClientSecret)
	if err != nil {
		uc.logger.Error("", "", "", fmt.Sprintf("failed to hash client secret for name=%s: %v", req.Name, err))
		return nil, fmt.Errorf("failed to hash client secret: %w", err)
	}

	// build child records for normalized tables
	var redirectURIs []domain.ClientRedirectURI
	for _, r := range req.RedirectURIs {
		redirectURIs = append(redirectURIs, domain.ClientRedirectURI{RedirectURI: r})
	}
	var grants []domain.ClientGrant
	for _, g := range req.Grants {
		grants = append(grants, domain.ClientGrant{GrantType: g})
	}
	scopes := make([]domain.ClientScope, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, domain.ClientScope{Scope: scope})
	}
	audiences := make([]domain.ClientAudience, 0, len(req.Audiences))
	for _, audience := range req.Audiences {
		audiences = append(audiences, domain.ClientAudience{Audience: audience})
	}
	contexts, err := clientAccessContexts(req.AccessContexts)
	if err != nil {
		return nil, err
	}
	status := req.Status
	if status == "" {
		status = "inactive"
	}

	client := &domain.Client{
		ID:               uuid.New(),
		Name:             req.Name,
		ClientSecretHash: hashedSecret,
		RedirectURIs:     redirectURIs,
		Grants:           grants,
		Scopes:           scopes,
		Audiences:        audiences,
		AccessContexts:   contexts,
		IsConfidential:   req.IsConfidential,
		Status:           status,
	}

	if err := uc.clientRepo.Create(ctx, client); err != nil {
		uc.logger.Error("", "", "", fmt.Sprintf("failed to create client name=%s: %v", req.Name, err))
		return nil, err
	}

	uc.logger.Info("", "", "", fmt.Sprintf("client created: id=%s", client.ID.String()))
	return client, nil
}

// GetByID retrieves a client by ID
func (uc *clientUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	return uc.clientRepo.FindByID(ctx, id)
}

// GetAll retrieves all clients
func (uc *clientUsecase) GetAll(ctx context.Context) ([]domain.Client, error) {
	return uc.clientRepo.FindAll(ctx)
}

// Update updates client information
func (uc *clientUsecase) Update(ctx context.Context, id uuid.UUID, req dto.ClientRequest) error {
	uc.logger.Info("", "", "", fmt.Sprintf("update client request received: id=%s", id.String()))
	if err := validateClientAuthorization(req); err != nil {
		return err
	}

	existingClient, err := uc.clientRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existingClient == nil {
		return domain.ErrNotFound
	}
	hashedSecret, err := uc.security.HashPassword(req.ClientSecret)
	if err != nil {
		uc.logger.Error("", "", "", fmt.Sprintf("failed to hash client secret for id=%s: %v", id.String(), err))
		return fmt.Errorf("failed to hash client secret: %w", err)
	}

	var redirectURIs []domain.ClientRedirectURI
	for _, r := range req.RedirectURIs {
		redirectURIs = append(redirectURIs, domain.ClientRedirectURI{RedirectURI: r})
	}
	var grants []domain.ClientGrant
	for _, g := range req.Grants {
		grants = append(grants, domain.ClientGrant{GrantType: g})
	}
	scopes := make([]domain.ClientScope, 0, len(req.Scopes))
	for _, scope := range req.Scopes {
		scopes = append(scopes, domain.ClientScope{Scope: scope})
	}
	audiences := make([]domain.ClientAudience, 0, len(req.Audiences))
	for _, audience := range req.Audiences {
		audiences = append(audiences, domain.ClientAudience{Audience: audience})
	}
	contexts, err := clientAccessContexts(req.AccessContexts)
	if err != nil {
		return err
	}
	status := req.Status
	if status == "" {
		status = existingClient.Status
	}

	client := &domain.Client{
		Name:             req.Name,
		ClientSecretHash: hashedSecret,
		RedirectURIs:     redirectURIs,
		Grants:           grants,
		Scopes:           scopes,
		Audiences:        audiences,
		AccessContexts:   contexts,
		IsConfidential:   req.IsConfidential,
		Status:           status,
	}

	if err := uc.clientRepo.Update(ctx, id, client); err != nil {
		uc.logger.Error("", "", "", fmt.Sprintf("failed to update client id=%s: %v", id.String(), err))
		return err
	}

	uc.logger.Info("", "", "", fmt.Sprintf("client updated: id=%s", id.String()))
	return nil
}

func validateClientAuthorization(req dto.ClientRequest) error {
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" {
		return fmt.Errorf("client status must be active or inactive")
	}
	for _, grant := range req.Grants {
		if grant == "password" {
			return fmt.Errorf("password grant is not supported")
		}
		switch grant {
		case "client_credentials", "authorization_code", "refresh_token":
		default:
			return fmt.Errorf("unsupported client grant type %q", grant)
		}
		if grant == "client_credentials" && !req.IsConfidential {
			return fmt.Errorf("client_credentials requires a confidential client")
		}
	}
	if len(req.Scopes) == 0 || len(req.Audiences) == 0 {
		return fmt.Errorf("at least one allowed scope and audience is required")
	}
	if err := uniqueNonEmpty(req.Scopes, "scope"); err != nil {
		return err
	}
	return uniqueNonEmpty(req.Audiences, "audience")
}

func uniqueNonEmpty(values []string, name string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate %s %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func clientAccessContexts(inputs []dto.ClientAccessContextRequest) ([]domain.ClientAccessContext, error) {
	contexts := make([]domain.ClientAccessContext, 0, len(inputs))
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		scopeType := strings.TrimSpace(input.ScopeType)
		scopeID := strings.TrimSpace(input.ScopeID)
		switch scopeType {
		case "global":
			if scopeID != "" {
				return nil, fmt.Errorf("global scope must not include scope_id")
			}
		case "faculty", "department", "unit", "assigned":
			if scopeID == "" {
				return nil, fmt.Errorf("%s scope requires scope_id", scopeType)
			}
		default:
			return nil, fmt.Errorf("unsupported client scope_type %q", scopeType)
		}
		key := scopeType + "\x00" + scopeID
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate client access context")
		}
		seen[key] = struct{}{}
		contexts = append(contexts, domain.ClientAccessContext{ScopeType: scopeType, ScopeID: scopeID})
	}
	return contexts, nil
}

// Delete removes a client
func (uc *clientUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	uc.logger.Info("", "", "", fmt.Sprintf("delete client request received: id=%s", id.String()))
	if err := uc.clientRepo.Delete(ctx, id); err != nil {
		uc.logger.Error("", "", "", fmt.Sprintf("failed to delete client id=%s: %v", id.String(), err))
		return err
	}
	uc.logger.Info("", "", "", fmt.Sprintf("client deleted: id=%s", id.String()))
	return nil
}
