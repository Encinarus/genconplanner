package logging

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func PrintEnv() {
	log.Println("--- Environment Variables ---")
	sensitiveKeys := map[string]bool{
		"BGG_API_KEY":       true,
		"FIREBASE_CONFIG":   true,
		"FIREBASE_API_KEY":  true,
		"POSTGRES_PASSWORD": true,
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		val := pair[1]

		if sensitiveKeys[key] {
			log.Printf("%s=[REDACTED]\n", key)
		} else if key == "DATABASE_URL" {
			// Redact password in URL: postgres://user:password@host:port/db
			redacted := val
			if atIdx := strings.LastIndex(val, "@"); atIdx != -1 {
				if protoIdx := strings.Index(val, "://"); protoIdx != -1 {
					prefix := val[:protoIdx+3]
					suffix := val[atIdx:]
					userInfo := val[protoIdx+3 : atIdx]
					if colonIdx := strings.Index(userInfo, ":"); colonIdx != -1 {
						user := userInfo[:colonIdx]
						redacted = prefix + user + ":[REDACTED]" + suffix
					}
				}
			}
			log.Printf("%s=%s\n", key, redacted)
		} else if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "secret") || strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "api") || strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "credentials") || strings.Contains(strings.ToLower(key), "url") || strings.Contains(strings.ToLower(key), "jwt") {
			// Catch-all for other potentially sensitive variables
			log.Printf("%s=[REDACTED]\n", key)
		} else {
			log.Println(e)
		}
	}
	log.Println("--- End Environment Variables ---")
}

func LogWithError(err error, message string) {
	log.Printf("%s: %v\n", message, err)
	debug.PrintStack()
}

func ErrorStackTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			log.Printf("Error: %v\n", err.Err)
			log.Printf("Stack trace:\n%s\n", string(debug.Stack()))
		}
	}
}

const (
	RequestIDKey = "RequestId"
	UserEmailKey = "userEmail"
)

func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
		}
		c.Set(RequestIDKey, reqID)
		c.Header("X-Request-Id", reqID)
		c.Next()
	}
}

func LogCtx(c *gin.Context, format string, args ...interface{}) {
	reqID, _ := c.Get(RequestIDKey)
	if reqID == nil || reqID == "" {
		reqID = "-"
	}
	email := ""
	if user, exists := c.Get(UserEmailKey); exists {
		email, _ = user.(string)
	}
	if email == "" {
		email = "anon"
	}
	prefix := strings.TrimRight(fmt.Sprintf("[req:%v] [user:%s] ", reqID, email), " ") + " "
	if len(args) == 0 {
		log.Print(prefix + format)
	} else {
		log.Printf(prefix+format, args...)
	}
}
