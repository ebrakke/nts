package main

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"nts/internal/handlers"
	"nts/internal/middleware"

	"github.com/fiatjaf/eventstore/sqlite3"
	"github.com/fiatjaf/khatru"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
)

//go:embed templates
var viewFS embed.FS

var templates map[string]*template.Template
var funcMap = template.FuncMap{
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

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("Error loading .env file", "error", err)
	}

	layoutTemplate := template.Must(template.New("").Funcs(funcMap).ParseFS(viewFS, "templates/layout/*.html"))
	layoutTemplate = template.Must(layoutTemplate.ParseFS(viewFS, "templates/components/*.html"))

	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["homePage"] = template.Must(template.Must(layoutTemplate.Clone()).ParseFS(viewFS, "templates/pages/index.html"))
	templates["notePage"] = template.Must(template.Must(layoutTemplate.Clone()).ParseFS(viewFS, "templates/pages/note.html"))
	templates["note_list"] = layoutTemplate
	templates["note_card"] = layoutTemplate
}

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create the relay instance
	relay := khatru.NewRelay()

	// Set up basic properties from environment variables
	relay.Info.Name = os.Getenv("RELAY_NAME")
	relay.Info.Description = os.Getenv("RELAY_DESCRIPTION")
	relay.Info.PubKey = os.Getenv("RELAY_PUBKEY")

	// Initialize SQLite database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "khatru.db"
	}
	db := sqlite3.SQLite3Backend{DatabaseURL: dbURL}
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

	// Create a router
	mux := relay.Router()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Add the homepage with request logging
	mux.Handle("/", middleware.LogRequests(handlers.HomeHandler(relay.Info, templates)))

	// Add save-note route with request logging
	mux.Handle("/save-note", middleware.LogRequests(handlers.SaveNoteHandler(relay, templates)))

	// Add fetch-notes route
	mux.Handle("/fetch-notes", middleware.LogRequests(handlers.FetchNotes(relay, templates)))

	// Add fetch-single-note route
	mux.Handle("/note/", middleware.LogRequests(handlers.FetchSingleNote(relay, templates)))

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	slog.Info("Starting Khatru relay", "address", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
