package utils

import (
	"errors"
	"time"

	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // unix timestamp
}

type tokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	Type   string `json:"type"` // "access" | "refresh"
	jwt.RegisteredClaims
}

func GenerateTokenPair(claims model.JWTClaims, secret string, expireHours, refreshExpHours int) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(time.Duration(expireHours) * time.Hour)
	refreshExp := now.Add(time.Duration(refreshExpHours) * time.Hour)

	// Access token
	accessToken, err := generateToken(claims, secret, accessExp, "access")
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshToken, err := generateToken(claims, secret, refreshExp, "refresh")
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp.Unix(),
	}, nil
}

func generateToken(claims model.JWTClaims, secret string, exp time.Time, tokenType string) (string, error) {
	c := tokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		Name:   claims.Name,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString, secret string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return &model.JWTClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		Name:   claims.Name,
	}, nil
}