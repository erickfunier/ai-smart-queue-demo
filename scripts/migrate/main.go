package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/erickfunier/ai-smart-queue/internal/infrastructure/config"
	"github.com/jackc/pgx/v5"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	conn, err := pgx.Connect(context.Background(), cfg.Postgres.DSN)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	// Read all migration files
	entries, err := os.ReadDir("migrations")
	if err != nil {
		panic(err)
	}

	// Sort them alphabetically: 001, 002, 003...
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		fmt.Println("Running migration:", name)

		content, err := os.ReadFile("migrations/" + name)
		if err != nil {
			panic(err)
		}

		_, err = conn.Exec(context.Background(), string(content))
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("✅ All migrations applied")
}
