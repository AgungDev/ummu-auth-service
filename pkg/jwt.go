package pkg

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessScope struct {
	Type string  `json:"type"`
	ID   *string `json:"id"`
}

type AccessContext struct {
	Role        string      `json:"role,omitempty"`
	Scope       AccessScope `json:"scope"`
	Permissions []string    `json:"permissions,omitempty"`
}

type Claims struct {
	PrincipalType  string          `json:"principal_type"`
	ClientID       string          `json:"client_id,omitempty"`
	Roles          []string        `json:"roles,omitempty"`
	Permissions    []string        `json:"permissions,omitempty"`
	Scope          string          `json:"scope,omitempty"`
	AccessContexts []AccessContext `json:"access_contexts,omitempty"`
	jwt.RegisteredClaims
}

func LoadPrivateKey(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func LoadPublicKey(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func GenerateSignedJWT(claims Claims, privateKeyPath, keyID string, expireMinutes int) (string, error) {
	keyData, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		return "", err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" || claims.Issuer == "" || len(claims.Audience) == 0 {
		return "", errors.New("JWT subject, issuer, and audience are required")
	}
	if claims.PrincipalType != "user" && claims.PrincipalType != "client" {
		return "", errors.New("JWT principal_type must be user or client")
	}
	if claims.PrincipalType == "user" && (claims.ClientID != "" || claims.Scope != "") {
		return "", errors.New("user JWT must not contain client_id or OAuth scope")
	}
	if claims.PrincipalType == "client" &&
		(claims.ClientID == "" || claims.Subject != claims.ClientID || claims.Scope == "" ||
			len(claims.Roles) != 0 || len(claims.Permissions) != 0) {
		return "", errors.New("client JWT requires matching client_id and sub, and must not contain user RBAC claims")
	}
	now := time.Now()
	claims.ID = uuid.NewString()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(expireMinutes) * time.Minute))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(privateKey)
}

func PublicJWKS(publicKeyPath, keyID string) (map[string]any, error) {
	keyData, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		return nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, err
	}
	exponent := big.NewInt(int64(publicKey.E)).Bytes()
	return map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": keyID,
			"n":   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(exponent),
		}},
	}, nil
}

func ValidateJWTRS256(tokenString, publicKeyPath, issuer string, allowedAudiences []string) (*Claims, error) {
	keyData, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		return nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return publicKey, nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}
	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Subject == "" || claims.ID == "" {
		return nil, errors.New("token subject and jti are required")
	}
	if claims.PrincipalType != "user" && claims.PrincipalType != "client" {
		return nil, errors.New("invalid principal_type")
	}
	if claims.PrincipalType == "client" &&
		(claims.ClientID == "" || claims.Subject != claims.ClientID || claims.Scope == "" ||
			len(claims.Roles) != 0 || len(claims.Permissions) != 0) {
		return nil, errors.New("client token requires client_id and scope")
	}
	if claims.PrincipalType == "user" && (claims.ClientID != "" || claims.Scope != "") {
		return nil, errors.New("user token must not contain client_id or OAuth scope")
	}
	if len(claims.Audience) == 0 || !hasAllowedAudience(claims.Audience, allowedAudiences) {
		return nil, errors.New("token audience is not allowed")
	}
	return claims, nil
}

func hasAllowedAudience(audiences, allowed []string) bool {
	for _, audience := range audiences {
		for _, candidate := range allowed {
			if audience == candidate {
				return true
			}
		}
	}
	return false
}
