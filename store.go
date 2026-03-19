package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type Update struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type Store struct {
	mu     sync.RWMutex
	items  []Update
	file   string
	loc    *time.Location
	subsMu sync.Mutex
	subs   map[chan Update]struct{}
}

func NewStore(file, timezone string) *Store {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Printf("warn: unknown timezone %q, falling back to UTC: %v", timezone, err)
		loc = time.UTC
	}
	s := &Store{
		file: file,
		loc:  loc,
		subs: make(map[chan Update]struct{}),
	}
	if file != "" {
		s.load()
	}
	return s
}

func (s *Store) Add(text string) {
	u := Update{Text: text, Timestamp: time.Now().In(s.loc)}

	s.mu.Lock()
	s.items = append(s.items, u)
	s.mu.Unlock()

	if s.file != "" {
		s.save()
	}

	s.subsMu.Lock()
	for ch := range s.subs {
		select {
		case ch <- u:
		default: // drop if subscriber is slow
		}
	}
	s.subsMu.Unlock()
}

// All returns updates in chronological order (oldest first).
func (s *Store) All() []Update {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Update, len(s.items))
	copy(result, s.items)
	return result
}

func (s *Store) Subscribe() chan Update {
	ch := make(chan Update, 16)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	return ch
}

func (s *Store) Unsubscribe(ch chan Update) {
	s.subsMu.Lock()
	delete(s.subs, ch)
	s.subsMu.Unlock()
}

func (s *Store) load() {
	data, err := os.ReadFile(s.file)
	if err != nil {
		return // file may not exist yet
	}
	var updates []Update
	if err := json.Unmarshal(data, &updates); err != nil {
		log.Printf("warn: could not parse update file: %v", err)
		return
	}
	s.items = updates
	log.Printf("Loaded %d updates from %s", len(updates), s.file)
}

func (s *Store) save() {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.items, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		log.Printf("warn: could not marshal updates: %v", err)
		return
	}
	if err := os.WriteFile(s.file, data, 0644); err != nil {
		log.Printf("warn: could not write update file: %v", err)
	}
}
