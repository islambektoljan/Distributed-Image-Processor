package security

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type KeycloakClaims struct {
	Azp               string `json:"azp"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	jwt.RegisteredClaims
}

// AuthMiddleware creates a Gin middleware for JWT validation against Keycloak
func AuthMiddleware(jwksURL, clientID string) gin.HandlerFunc {
	// Create JWKS client with auto-refresh
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval:  time.Hour,
		RefreshTimeout:   10 * time.Second,
		RefreshRateLimit: time.Minute * 5,
		RefreshErrorHandler: func(err error) {
			log.Printf("Error refreshing JWKS: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Failed to create JWKS client: %v", err)
	}

	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check for Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT
		token, err := jwt.ParseWithClaims(tokenString, &KeycloakClaims{}, jwks.Keyfunc)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Invalid token: %v", err)})
			c.Abort()
			return
		}

		// Check if token is valid
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not valid"})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*KeycloakClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to extract claims"})
			c.Abort()
			return
		}

		// Validate audience (client ID)
		if claims.Azp != clientID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid audience"})
			c.Abort()
			return
		}

		// Validate expiration
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
			c.Abort()
			return
		}

		// Store claims in context for later use
		c.Set("user", claims.PreferredUsername)
		c.Set("email", claims.Email)
		c.Set("claims", claims)

		c.Next()
	}
}
