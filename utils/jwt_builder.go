package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	accessTokenTTL  = 30 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour

	ErrInvalidToken = errors.New("invalid token")
)

func jwtSecret() []byte {
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return []byte(v)
	}
	// Development fallback — DO NOT use in production. Set JWT_SECRET in env.
	return []byte("dev-insecure-default-secret-change-me")
}

// GenerateAccessToken issues a short-lived access token.
func GenerateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"typ":    "access",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(accessTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// GenerateRefreshToken issues a longer-lived refresh token.
// The `typ` claim prevents a refresh token from being accepted as an access token.
func GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"typ":    "refresh",
		"iat":    time.Now().Unix(),
		"exp":    time.Now().Add(refreshTokenTTL).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ParseJWT parses and validates signature/expiry for any token.
func ParseJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret(), nil
	})
}

// ParseAccessToken validates and enforces `typ=access`.
func ParseAccessToken(tokenString string) (jwt.MapClaims, error) {
	return parseWithType(tokenString, "access")
}

// ParseRefreshToken validates and enforces `typ=refresh`.
func ParseRefreshToken(tokenString string) (jwt.MapClaims, error) {
	return parseWithType(tokenString, "refresh")
}

func parseWithType(tokenString, wantTyp string) (jwt.MapClaims, error) {
	token, err := ParseJWT(tokenString)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	typ, _ := claims["typ"].(string)
	if typ != wantTyp {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GetUserIDFromClaims returns the userID string from token claims.
func GetUserIDFromClaims(claims jwt.MapClaims) (string, error) {
	uid, ok := claims["userID"].(string)
	if !ok || uid == "" {
		return "", ErrInvalidToken
	}
	return uid, nil
}
