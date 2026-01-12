package jwtutils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type JwtCustomClaims struct {
	UserID uint      `json:"user_id"`
	Role   string    `json:"role"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type JWTService struct {
	SecretKey     string
	Issuer        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func NewJWTService(secret string, issuer string) *JWTService {
	return &JWTService{
		SecretKey:     secret,
		Issuer:        issuer,
		AccessExpiry:  15 * time.Minute,   // อายุสั้น (เช่น 15 นาที)
		RefreshExpiry: 7 * 24 * time.Hour, // อายุยาว (เช่น 7 วัน)
	}
}

func (j *JWTService) GenerateToken(userID uint, role string, tokenType TokenType) (string, error) {
	expiry := j.AccessExpiry
	if tokenType == RefreshToken {
		expiry = j.RefreshExpiry
	}

	claims := &JwtCustomClaims{
		UserID: userID,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			Issuer:    j.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.SecretKey))
}

func (j *JWTService) ValidateToken(tokenString string) (*JwtCustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.SecretKey), nil
	})

	if claims, ok := token.Claims.(*JwtCustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
