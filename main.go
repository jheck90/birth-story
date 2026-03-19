package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed templates/index.html
var templateFS embed.FS

func main() {
	token     := mustEnv("DISCORD_TOKEN")
	appID     := mustEnv("DISCORD_APP_ID")
	guildID   := mustEnv("DISCORD_GUILD_ID")
	publicKey := mustEnv("DISCORD_PUBLIC_KEY")
	port       := getEnv("PORT", "8080")
	updateFile := os.Getenv("UPDATE_FILE")
	timezone   := getEnv("TZ", "America/Denver")
	familyName := getEnv("FAMILY_NAME", "Our")
	allowedUsers := parseAllowedUsers(os.Getenv("ALLOWED_DISCORD_IDS"))

	store := NewStore(updateFile, timezone)

	log.Println("Registering /update slash command with Discord...")
	if err := registerCommand(appID, guildID, token); err != nil {
		log.Fatalf("Failed to register Discord command: %v", err)
	}
	log.Println("Discord command registered.")

	tmpl := template.Must(template.ParseFS(templateFS, "templates/index.html"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", indexHandler(store, tmpl, familyName))
	mux.HandleFunc("GET /events", eventsHandler(store))
	mux.HandleFunc("POST /interactions", interactionsHandler(store, publicKey, allowedUsers))

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

// parseAllowedUsers converts a comma-separated string of Discord user IDs into
// a set. Returns an empty set if the env var is unset (no restriction applied).
func parseAllowedUsers(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, id := range strings.Split(raw, ",") {
		if id := strings.TrimSpace(id); id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
