package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"nts/internal/handlers"
	"nts/internal/middleware"

	"github.com/fiatjaf/eventstore/sqlite3"
	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create the relay instance
	relay := khatru.NewRelay()

	// Set up basic properties
	relay.Info.Name = "My Khatru Relay"
	relay.Info.Description = "A custom Nostr relay using Khatru and SQLite"
	relay.Info.PubKey = "replace_with_your_public_key"

	// Initialize SQLite database
	db := sqlite3.SQLite3Backend{DatabaseURL: "khatru.db"}
	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize SQLite database", "error", err)
		os.Exit(1)
	}

	// Set up storage functions
	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)

	// Define template functions
	funcMap := template.FuncMap{
		"formatTime": func(timestamp nostr.Timestamp) string {
			t := time.Unix(int64(timestamp), 0)
			return t.Format("Jan 2, 2006 15:04")
		},
		"getTitle": func(s *nostr.Event) string {
			tags := s.Tags
			for _, tag := range tags {
				if tag[0] == "title" {
					return tag[1]
				}
			}
			return ""
		},
		"truncate": func(n int, s string) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
	}

	// Parse templates with function map
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/**/*.tmpl"))

	// Create a router
	mux := relay.Router()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Add the homepage with request logging
	mux.Handle("/", middleware.LogRequests(handlers.HomeHandler(tmpl, relay.Info)))

	// Add save-note route with request logging
	mux.Handle("/save-note", middleware.LogRequests(handlers.SaveNoteHandler(tmpl, relay)))

	// Add fetch-notes route
	mux.Handle("/fetch-notes", middleware.LogRequests(handlers.FetchNotes(tmpl, relay)))

	// Start the server
	port := ":8081"
	slog.Info("Starting Khatru relay", "address", "http://localhost"+port)
	if err := http.ListenAndServe(port, mux); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
