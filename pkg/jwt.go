package pkg

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Roles []interface{} `json:"roles,omitempty"`
	Scope string        `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

func LoadPrivateKey(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func LoadPublicKey(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func GenerateJWTRS256(sub, iss string, roles []interface{}, privateKeyPath string, expireMinutes int) (string, error) {
	return GenerateResourceJWTRS256(sub, iss, "", "", roles, privateKeyPath, "auth-service", expireMinutes)
}

func GenerateResourceJWTRS256(sub, iss, audience, scope string, roles []interface{}, privateKeyPath, keyID string, expireMinutes int) (string, error) {
	keyData, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		return "", err
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return "", err
	}
	audiences := jwt.ClaimStrings(nil)
	if audience != "" {
		audiences = jwt.ClaimStrings{audience}
	}
	claims := Claims{
		Roles: roles,
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Subject:   sub,
			Audience:  audiences,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireMinutes) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(privKey)
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

func ValidateJWTRS256(tokenString string, publicKeyPath string) (*Claims, error) {
	keyData, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		return nil, err
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, err
	}
	parsedToken, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
