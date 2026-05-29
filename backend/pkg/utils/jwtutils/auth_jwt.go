package jwtutils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// JwtCustomClaims โครงสร้างของ Claims ที่เราจะเก็บใน JWT
type JwtCustomClaims struct {
	UserID uint      `json:"user_id"`
	Role   string    `json:"role"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTService จัดการการสร้างและตรวจสอบ JWT Tokens
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

// GenerateToken สร้าง JWT Token พร้อม Claims
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
			ID:        uuid.New().String(), // jti — ป้องกัน duplicate token เมื่อ login พร้อมกันหลาย request
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			Issuer:    j.Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.SecretKey))
}

// ValidateToken ตรวจสอบความถูกต้องของ Token และคืนค่า Claims
func (j *JWTService) ValidateToken(tokenString string) (*JwtCustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JwtCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.SecretKey), nil
	})

	if claims, ok := token.Claims.(*JwtCustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
