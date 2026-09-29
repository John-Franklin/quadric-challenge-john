package main

import (
	"backend/db"
	"backend/handlers"
	"context"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	cancel()

	h := handlers.New(conn)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/runner", h.RunnerHandler)
	mux.HandleFunc("GET /api/jobs/acquire", h.AcquireJobsHandler)
	mux.HandleFunc("GET /api/jobs", h.ListJobsHandler)
	mux.HandleFunc("POST /api/jobs", h.CreateJobHandler)
	mux.HandleFunc("POST /api/jobs/{id}/logs", h.AppendLogsHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
