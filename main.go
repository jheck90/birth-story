package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/index.html
var templateFS embed.FS

func main() {
	token     := mustEnv("DISCORD_TOKEN")
	appID     := mustEnv("DISCORD_APP_ID")
	guildID   := mustEnv("DISCORD_GUILD_ID")
	publicKey := mustEnv("DISCORD_PUBLIC_KEY")
	port      := getEnv("PORT", "8080")
	updateFile := os.Getenv("UPDATE_FILE")

	store := NewStore(updateFile)

	log.Println("Registering /update slash command with Discord...")
	if err := registerCommand(appID, guildID, token); err != nil {
		log.Fatalf("Failed to register Discord command: %v", err)
	}
	log.Println("Discord command registered.")

	tmpl := template.Must(template.ParseFS(templateFS, "templates/index.html"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", indexHandler(store, tmpl))
	mux.HandleFunc("GET /events", eventsHandler(store))
	mux.HandleFunc("POST /interactions", interactionsHandler(store, publicKey))

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Required env var %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
