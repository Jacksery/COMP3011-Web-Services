package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var secret []byte

// Init initializes the JWT secret. If JWT_SECRET is set in the environment it will be used.
// Otherwise a random base64 secret is generated, set in the environment, and printed to the terminal
// (for development/testing convenience).
func Init() {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		secret = []byte(s)
		return
	}
	// generate 32 random bytes and base64 encode them (single-line)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("warning: failed to generate random JWT secret, falling back to devsecret: %v", err)
		secret = []byte("devsecret")
		if err := os.Setenv("JWT_SECRET", string(secret)); err != nil {
			log.Printf("warning: failed to set JWT_SECRET env var: %v", err)
		}
		return
	}
	s := base64.StdEncoding.EncodeToString(b)
	secret = []byte(s)
	if err := os.Setenv("JWT_SECRET", s); err != nil {
		log.Printf("warning: failed to set JWT_SECRET env var: %v", err)
	}
	// Print a visible warning — only when not running in production
	env := os.Getenv("ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "production" {
		log.Printf("WARNING: generated JWT secret but hiding it because ENV=production")
	} else {
		log.Printf("WARNING: generated JWT secret for dev/test: %s", s)
	}
}

func GenerateJWT(username string) (string, error) {
	if secret == nil {
		Init()
	}
	claims := jwt.MapClaims{
		"sub":   username,
		"admin": true,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ValidateCredentials(username, password string) bool {
	u := os.Getenv("ADMIN_USER")
	p := os.Getenv("ADMIN_PASSWORD")
	if u == "" || p == "" {
		// development default: admin/password
		return username == "admin" && password == "password"
	}
	return username == u && password == p
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth"})
			return
		}
		const prefix = "Bearer "
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth header"})
			return
		}
		tok := auth[len(prefix):]
		if secret == nil {
			Init()
		}
		_, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
			// ensure token uses HMAC SHA-256
			if t.Method == nil || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Method)
			}
			return secret, nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Next()
	}
}
