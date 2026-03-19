package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"time"
)

type pageData struct {
	FamilyName string
	Updates    []Update
}

func indexHandler(store *Store, tmpl *template.Template, familyName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updates := store.All()
		// Reverse in place so newest appears first in the template.
		slices.Reverse(updates)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, pageData{FamilyName: familyName, Updates: updates})
	}
}

func eventsHandler(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // tell nginx not to buffer SSE

		ch := store.Subscribe()
		defer store.Unsubscribe(ch)

		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case u := <-ch:
				data, _ := json.Marshal(u)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}
