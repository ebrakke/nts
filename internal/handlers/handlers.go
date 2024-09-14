package handlers

import (
	"html/template"
	"net/http"

	"github.com/nbd-wtf/go-nostr/nip11"
)

func HomeHandler(tmpl *template.Template, info *nip11.RelayInformationDocument) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Info  nip11.RelayInformationDocument
		}{
			Title: "Home",
			Info:  *info,
		}

		if err := tmpl.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
