// Command backend serves the jobs API on :8080.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"backend/db"
	"backend/handlers"
	"backend/sessions"
)

// Matches the runner's default so local development works without configuration.
const devRunnerToken = "dev-runner-token"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := db.Connect(ctx)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	store, err := sessions.Connect(ctx)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer store.Close()

	runnerToken := os.Getenv("RUNNER_TOKEN")
	if runnerToken == "" {
		log.Printf("RUNNER_TOKEN not set, using the insecure development token %q", devRunnerToken)
		runnerToken = devRunnerToken
	}

	h := &handlers.Handlers{
		DB:          conn,
		Sessions:    store,
		RunnerToken: runnerToken,
		AdminEmail:  strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
	}
	log.Println("Server starting on :8080")
	return http.ListenAndServe(":8080", h.Routes())
}
