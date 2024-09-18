package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
)

func HomeHandler(info *nip11.RelayInformationDocument, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Handling request for Home page", "remote_addr", r.RemoteAddr)

		data := struct {
			Title string
			Info  nip11.RelayInformationDocument
		}{
			Title: "Home",
			Info:  *info,
		}

		if err := templates["homePage"].ExecuteTemplate(w, "base.html", data); err != nil {
			slog.Error("Error executing template", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Home page rendered successfully")
	}
}

func SaveNoteHandler(relay *khatru.Relay, templates map[string]*template.Template) http.HandlerFunc {
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
		if event.Kind != 31234 || event.Content == "" {
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
		err = templates["note_card"].ExecuteTemplate(w, "note_card", event)
		if err != nil {
			http.Error(w, "Error rendering note card", http.StatusInternalServerError)
			return
		}
	}
}

func FetchNotes(relay *khatru.Relay, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pubkey := r.URL.Query().Get("pubkey")
		if pubkey == "" {
			http.Error(w, "Pubkey is required", http.StatusBadRequest)
			return
		}

		// Fetch notes for the given pubkey
		notesChan, err := relay.QueryEvents[0](context.Background(), nostr.Filter{
			Kinds:   []int{31234},
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

		tmpl := templates["note_list"]

		// Render the notes using the note_list template
		err = tmpl.ExecuteTemplate(w, "note_list", allNotes)
		if err != nil {
			fmt.Println(err)
			slog.Error("Failed to render notes", "error", err)
			http.Error(w, "Failed to render notes", http.StatusInternalServerError)
			return
		}
	}
}

func FetchSingleNote(relay *khatru.Relay, templates map[string]*template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID := strings.TrimPrefix(r.URL.Path, "/note/")

		filter := nostr.Filter{
			IDs: []string{eventID},
		}

		events, err := relay.QueryEvents[0](context.Background(), filter)
		if err != nil {
			http.Error(w, "Failed to fetch note", http.StatusInternalServerError)
			return
		}

		var allEvents []*nostr.Event
		for event := range events {
			allEvents = append(allEvents, event)
		}

		if len(allEvents) == 0 {
			http.NotFound(w, r)
			return
		}

		data := struct {
			Title string
			Note  *nostr.Event
		}{
			Title: "Note",
			Note:  allEvents[0],
		}
		err = templates["notePage"].ExecuteTemplate(w, "base.html", data)
		if err != nil {
			slog.Error("Failed to render template", "error", err)
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	}
}
