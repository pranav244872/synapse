package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pranav244872/synapse/api"
	"github.com/pranav244872/synapse/config"
	db "github.com/pranav244872/synapse/db/sqlc"
	"github.com/pranav244872/synapse/skillz"
)

func main() {
	// Step 1: Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("❌ could not load configuration: %v", err)
	}
	log.Println("✅ Configuration loaded successfully.")

	// Step 2: Establish database connection pool
	connPool, err := pgxpool.New(context.Background(), cfg.DBSource)
	if err != nil {
		log.Fatalf("❌ could not connect to the database: %v", err)
	}
	defer connPool.Close()
	log.Println("✅ Database connection pool established.")

	// Step 3: Initialize the database store
	store := db.NewStore(connPool)

	// Step 4: Load skill aliases from the database to build the alias map
	log.Println("🔄 Loading skill aliases from the database...")
	aliasRows, err := store.GetAllSkillAliases(context.Background())
	if err != nil {
		// It's a fatal error because the skillz processor depends on it.
		log.Fatalf("❌ could not load skill aliases: %v", err)
	}

	aliasMap := make(map[string]string)
	for _, row := range aliasRows {
		aliasMap[row.AliasName] = row.CanonicalName
	}
	log.Printf("✅ Loaded %d skill aliases.", len(aliasMap))

	// Step 5: Initialize the skill processing service with the loaded aliases
	geminiClient := skillz.NewGeminiLLMClient(cfg.GeminiAPIKey, cfg.GeminiAPIURL, &http.Client{})
	skillzProcessor := skillz.NewLLMProcessor(aliasMap, geminiClient)
	log.Println("✅ Skillz processor (Gemini) initialized.")

	// Step 6: Create a new API server instance
	server, err := api.NewServer(cfg, store, skillzProcessor)
	if err != nil {
		log.Fatalf("❌ could not create the server: %v", err)
	}
	log.Println("✅ API server created.")

	// Step 7: Start the HTTP server
	log.Printf("🚀 Starting server on %s", cfg.ServerAddress)
	if err := server.Start(cfg.ServerAddress); err != nil {
		log.Fatalf("❌ failed to start server: %v", err)
	}
}
