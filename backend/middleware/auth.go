package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/cinema-booking/backend/models"
	"github.com/cinema-booking/backend/repository"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	firebaseAuth *auth.Client
	userRepo     *repository.UserRepository
}

func NewAuthMiddleware(firebaseAuth *auth.Client, userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{firebaseAuth: firebaseAuth, userRepo: userRepo}
}

// Authenticate verifies Firebase ID token and injects user into context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		idToken := strings.TrimPrefix(header, "Bearer ")

		token, err := m.firebaseAuth.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Upsert user in MongoDB
		user, err := m.userRepo.UpsertByFirebaseUID(context.Background(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "user sync failed"})
			return
		}

		c.Set("user", user)
		c.Set("firebase_uid", token.UID)
		c.Next()
	}
}

// RequireRole ensures the authenticated user has the expected role
func RequireRole(role models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		u := user.(*models.User)
		if u.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
