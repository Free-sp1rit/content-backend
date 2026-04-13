package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")
var ErrExpiredToken = errors.New("token expired")
var ErrInvalidTokenConfig = errors.New("invalid token config")

const defaultTokenTTL = 24 * time.Hour

type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

type Claims struct {
	UserID    int64  `json:"user_id"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type tokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func NewTokenManager(secret, issuer string, ttl time.Duration) *TokenManager {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}

	return &TokenManager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
		now:    time.Now,
	}
}

func (m *TokenManager) currentTime() time.Time {
	if m.now != nil {
		return m.now()
	}

	return time.Now()
}

func (m *TokenManager) Generate(userID int64) (string, error) {
	if len(m.secret) == 0 {
		return "", ErrInvalidTokenConfig
	}

	header := tokenHeader{
		Alg: "HS256",
		Typ: "JWT",
	}

	now := m.currentTime()
	claims := Claims{
		UserID:    userID,
		Issuer:    m.issuer,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(m.ttl).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerPart + "." + payloadPart

	signature := m.sign(signingInput)

	return signingInput + "." + signature, nil
}

func (m *TokenManager) ValidateAndParse(token string) (Claims, error) {
	if len(m.secret) == 0 {
		return Claims{}, ErrInvalidTokenConfig
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := m.sign(signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return Claims{}, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var header tokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return Claims{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Issuer != m.issuer {
		return Claims{}, ErrInvalidToken
	}

	if claims.ExpiresAt <= m.currentTime().Unix() {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func (m *TokenManager) sign(input string) string {
	hash := hmac.New(sha256.New, m.secret)
	hash.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}
