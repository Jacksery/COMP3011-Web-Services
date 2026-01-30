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
	_ = godotenv.Load()
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
	defer d.Close()

	r := gin.Default()
	handlers.RegisterRoutes(r, d)

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
