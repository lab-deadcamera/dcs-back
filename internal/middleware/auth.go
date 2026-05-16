package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Auth parses the JWT Bearer token and sets user claims in the context.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Extract standard claims
		if sub, ok := claims["sub"].(float64); ok {
			c.Set("userID", int64(sub))
		}
		if username, ok := claims["username"].(string); ok {
			c.Set("username", username)
		}

		// Extract role claims
		if roleLevel, ok := claims["role_level"].(float64); ok {
			c.Set("role_level", roleLevel)
		}
		if roleName, ok := claims["role_name"].(string); ok {
			c.Set("role_name", roleName)
		}
		if roleID, ok := claims["role_id"].(float64); ok {
			c.Set("role_id", int(roleID))
		}

		// Extract user profile claims
		if name, ok := claims["name"].(string); ok {
			c.Set("user_name", name)
		}
		if surname, ok := claims["surname"].(string); ok {
			c.Set("user_surname", surname)
		}
		if userName, ok := claims["user_name"].(string); ok {
			c.Set("user_display_name", userName)
		}
		if email, ok := claims["email"].(string); ok {
			c.Set("user_email", email)
		}

		c.Next()
	}
}

// RequireRole returns a middleware that rejects requests whose role level
// is greater than maxLevel (lower level = more privilege).
//   - RequireRole(0) = only SUPER_ADMIN
//   - RequireRole(1) = SUPER_ADMIN or ADMIN
//   - RequireRole(2) = SUPER_ADMIN, ADMIN, or SUPERVISOR
//   - RequireRole(3) = anyone authenticated
func RequireRole(maxLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("role_level")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: no role"})
			return
		}

		level, ok := raw.(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: invalid role"})
			return
		}

		if int(level) > maxLevel {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: insufficient role level"})
			return
		}

		c.Next()
	}
}
