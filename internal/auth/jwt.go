package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token expired")
)

type JWTClaims struct {
    UserID    string `json:"user_id"`
    Email     string `json:"email"`
    ExpiresAt int64  `json:"exp"`
    IssuedAt  int64  `json:"iat"`
}

type JWTService struct {
    secret []byte
}

func NewJWTService(secret string) *JWTService {
    return &JWTService{secret: []byte(secret)}
}

func (j *JWTService) GenerateToken(userID, email string, duration time.Duration) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    userID,
        "email":      email,
        "iat":        time.Now().Unix(),
        "exp":        time.Now().Add(duration).Unix(),
	})

	tokenString, err := token.SignedString(j.secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }

		return j.secret, nil
	})

	if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrExpiredToken
        }
        return nil, ErrInvalidToken
    }

	if !token.Valid {
        return nil, ErrInvalidToken
    }

	claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, ErrInvalidToken
    }

	jwtClaims := &JWTClaims{}

	if userID, ok := claims["user_id"].(string); ok {
        jwtClaims.UserID = userID
    } else {
        return nil, ErrInvalidToken
    }
	if email, ok := claims["email"].(string); ok {
        jwtClaims.Email = email
    } else {
        return nil, ErrInvalidToken
    }
	if exp, ok := claims["exp"].(float64); ok {
        jwtClaims.ExpiresAt = int64(exp)
    }
	if iat, ok := claims["iat"].(float64); ok {
        jwtClaims.IssuedAt = int64(iat)
    }

	return jwtClaims, nil
}