package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
    UserID string
    Email  string
}

func (j *JWTService) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var tokenString string

        if cookie, err := r.Cookie("auth_token"); err == nil {
            tokenString = cookie.Value
        }

        if tokenString == "" {
            authHeader := r.Header.Get("Authorization")
            if strings.HasPrefix(authHeader, "Bearer ") {
                tokenString = strings.TrimPrefix(authHeader, "Bearer ")
            }
        }

        if tokenString == "" {
            http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
            return
        }

        claims, err := j.ValidateToken(tokenString)
        if err != nil {
            http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), UserContextKey, &UserContext{
            UserID: claims.UserID,
            Email:  claims.Email,
        })

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func GetUserFromContext(ctx context.Context) *UserContext {
    user, ok := ctx.Value(UserContextKey).(*UserContext)
    if !ok {
        return nil
    }
    return user
}