package usecase

import (
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type JWTManager interface {
	GenerateToken(userID string) (string, error)
}

type jwtManager struct {
	secretKey     string
	tokenDuration time.Duration
}

func NewJWTManager(secretKey string, duration time.Duration) JWTManager {
	return &jwtManager{
		secretKey:     secretKey,
		tokenDuration: duration,
	}
}

func (j *jwtManager) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(j.tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}
