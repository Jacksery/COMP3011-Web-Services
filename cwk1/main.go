package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"retaildb-service/internal/auth"
	"retaildb-service/internal/db"
	"retaildb-service/internal/handlers"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// Not fatal if there is no .env file; log other errors
		if !os.IsNotExist(err) {
			log.Printf("warning: failed to load .env file: %v", err)
		}
	}
	// initialize JWT secret (generates one for dev if missing and prints it)
	auth.Init()
	dbPath := os.Getenv("RETAILDB_PATH")
	if dbPath == "" {
		dbPath = "./retailDB.sqlite"
	}
	d, err := db.OpenDB(dbPath)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer func() {
		if err := d.Close(); err != nil {
			log.Printf("warning: failed to close db: %v", err)
		}
	}()

	r := gin.Default()

	// Lightweight CORS middleware to allow Swagger UI and browser requests
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	})

	handlers.RegisterRoutes(r, d)

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
