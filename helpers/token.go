package helpers

import (
	"errors"
	"time"

	"cashmate-api/config"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token tidak valid")
	ErrTokenExpired = errors.New("token telah kedaluwarsa")
)

// secretAccess mengambil secret untuk Access Token dari konfigurasi.
func secretAccess() []byte {
	return []byte(config.JWTSecret())
}

// secretRefresh mengambil secret untuk Refresh Token dari konfigurasi.
func secretRefresh() []byte {
	return []byte(config.JWTRefreshSecret())
}

// JWTClaims adalah klaim standar JWT ditambah user_id, email & role.
type JWTClaims struct {
	UserID      uint   `json:"user_id"`
	BusinessID  uint   `json:"business_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	AuthVersion uint64 `json:"auth_version"`
	jwt.RegisteredClaims
}

// GenerateAccessToken membuat Access Token JWT yang berlaku singkat.
func GenerateAccessToken(userID, businessID uint, email, role string, authVersion uint64) (string, error) {
	ttl := time.Minute * time.Duration(config.AccessTokenTTL())

	claims := JWTClaims{
		UserID:      userID,
		BusinessID:  businessID,
		Email:       email,
		Role:        role,
		AuthVersion: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "cashmate-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretAccess())
}

// GenerateRefreshToken membuat Refresh Token JWT yang berlaku lama.
func GenerateRefreshToken(userID, businessID uint, email, role string, authVersion uint64) (string, error) {
	ttl := time.Hour * time.Duration(config.RefreshTokenTTL())

	claims := JWTClaims{
		UserID:      userID,
		BusinessID:  businessID,
		Email:       email,
		Role:        role,
		AuthVersion: authVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "cashmate-api-refresh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretRefresh())
}

// ParseAccessToken memvalidasi & menguraikan Access Token JWT.
func ParseAccessToken(tokenString string) (*JWTClaims, error) {
	return parse(tokenString, secretAccess(), "cashmate-api")
}

// ParseRefreshToken memvalidasi & menguraikan Refresh Token JWT.
func ParseRefreshToken(tokenString string) (*JWTClaims, error) {
	return parse(tokenString, secretRefresh(), "cashmate-api-refresh")
}

func parse(tokenString string, secret []byte, issuer string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return secret, nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	if claims.Issuer != issuer {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
