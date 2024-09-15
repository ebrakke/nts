package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
)

func HomeHandler(tmpl *template.Template, info *nip11.RelayInformationDocument) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Handling request for Home page", "remote_addr", r.RemoteAddr)

		data := struct {
			Title string
			Info  nip11.RelayInformationDocument
		}{
			Title: "Home",
			Info:  *info,
		}

		if err := tmpl.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			slog.Error("Error executing template", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Home page rendered successfully")
	}
}

func SaveNoteHandler(tmpl *template.Template, relay *khatru.Relay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Handling SaveNote request", "remote_addr", r.RemoteAddr)

		var event *nostr.Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			slog.Error("Error decoding request body", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		slog.Info("Received note event", "event", event)

		// Validate the event
		if event.Kind != 1990 || event.Content == "" {
			slog.Warn("Invalid note event", "kind", event.Kind, "content", event.Content)
			http.Error(w, "Invalid note event", http.StatusBadRequest)
			return
		}

		// Add the event to the relay
		_, err = relay.AddEvent(context.Background(), event)
		if err != nil {
			slog.Error("Failed to save note to relay", "error", err)
			http.Error(w, "Failed to save note", http.StatusInternalServerError)
			return
		}

		slog.Info("Note saved successfully to relay")

		// Render only the new note card
		err = tmpl.ExecuteTemplate(w, "note_card", event)
		if err != nil {
			http.Error(w, "Error rendering note card", http.StatusInternalServerError)
			return
		}
	}
}

func FetchNotes(tmpl *template.Template, relay *khatru.Relay) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pubkey := r.URL.Query().Get("pubkey")
		if pubkey == "" {
			http.Error(w, "Pubkey is required", http.StatusBadRequest)
			return
		}

		// Fetch notes for the given pubkey
		notesChan, err := relay.QueryEvents[0](context.Background(), nostr.Filter{
			Kinds:   []int{1990},
			Authors: []string{pubkey},
			Limit:   20,
		})

		if err != nil {
			slog.Error("Failed to fetch notes", "error", err)
			http.Error(w, "Failed to fetch notes", http.StatusInternalServerError)
			return
		}

		// Collect all events from the channel
		var allNotes []*nostr.Event
		for note := range notesChan {
			allNotes = append(allNotes, note)
		}

		// Render the notes using the note_list template
		err = tmpl.ExecuteTemplate(w, "note_list", allNotes)
		if err != nil {
			slog.Error("Failed to render notes", "error", err)
			http.Error(w, "Failed to render notes", http.StatusInternalServerError)
			return
		}
	}
}
