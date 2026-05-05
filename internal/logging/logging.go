package logging

import (
	"log"
	"os"
	"runtime/debug"
	"github.com/gin-gonic/gin"
)

func PrintEnv() {
	log.Println("--- Environment Variables ---")
	for _, e := range os.Environ() {
		log.Println(e)
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
