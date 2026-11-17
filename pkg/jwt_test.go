package pkg

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestResourceJWTAndJWKSShareSigningKeyID(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
	}), 0600); err != nil {
		t.Fatal(err)
	}

	tokenString, err := GenerateResourceJWTRS256(
		"client-subject",
		"https://auth.example.test",
		"siak-legacy-api",
		"siak:academic:read",
		nil,
		privatePath,
		"auth-key-1",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}
	var tokenClaims Claims
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &tokenClaims)
	if err != nil {
		t.Fatal(err)
	}
	if token.Header["kid"] != "auth-key-1" {
		t.Fatalf("token kid = %v, want auth-key-1", token.Header["kid"])
	}
	audienceMatches := false
	for _, audience := range tokenClaims.Audience {
		if audience == "siak-legacy-api" {
			audienceMatches = true
			break
		}
	}
	if tokenClaims.Issuer != "https://auth.example.test" ||
		tokenClaims.Subject != "client-subject" ||
		!audienceMatches ||
		tokenClaims.Scope != "siak:academic:read" {
		t.Fatalf("unexpected resource token claims: %+v", tokenClaims)
	}

	jwks, err := PublicJWKS(publicPath, "auth-key-1")
	if err != nil {
		t.Fatal(err)
	}
	keys := jwks["keys"].([]map[string]string)
	if len(keys) != 1 || keys[0]["kid"] != token.Header["kid"] {
		t.Fatalf("JWKS key ID does not match token: %#v", jwks)
	}
	if keys[0]["n"] != base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()) ||
		keys[0]["e"] != base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) {
		t.Fatalf("JWKS does not contain the signing public key: %#v", keys[0])
	}
}
