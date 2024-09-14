package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"nts/internal/handlers"

	"github.com/fiatjaf/eventstore/sqlite3"
	"github.com/fiatjaf/khatru"
)

func main() {
	// Create the relay instance
	relay := khatru.NewRelay()

	// Set up basic properties
	relay.Info.Name = "My Khatru Relay"
	relay.Info.Description = "A custom Nostr relay using Khatru and SQLite"
	relay.Info.PubKey = "replace_with_your_public_key"

	// Initialize SQLite database
	db := sqlite3.SQLite3Backend{DatabaseURL: "khatru.db"}
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}

	// Set up storage functions
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)

	// Parse templates
	tmpl := template.Must(template.ParseGlob("templates/**/*.tmpl"))

	// Create a router
	mux := relay.Router()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Add the homepage
	mux.HandleFunc("/", handlers.HomeHandler(tmpl, relay.Info))

	// Start the server
	port := ":8081"
	fmt.Printf("Starting Khatru relay on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
