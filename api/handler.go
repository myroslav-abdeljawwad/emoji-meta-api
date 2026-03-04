package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"

	"emoji-meta-api/models"
	"emoji-meta-api/server"
)

// EmojiMetaHandler serves the emoji metadata API endpoints.
// Version: 1.0 - Myroslav Mokhammad Abdeljawwad
type EmojiMetaHandler struct {
	mu      sync.RWMutex
	emojis  map[string]models.Emoji // key is unicode symbol
	dataDir string
}

// NewEmojiMetaHandler initializes the handler by loading emoji data from disk.
func NewEmojiMetaHandler() (*EmojiMetaHandler, error) {
	dir := os.Getenv("EMOJI_DATA_DIR")
	if dir == "" {
		dir = "./data"
	}

	emojis, err := loadEmojisFromDir(dir)
	if err != nil {
		return nil, err
	}
	return &EmojiMetaHandler{
		emojis:  emojis,
		dataDir: dir,
	}, nil
}

// loadEmojisFromDir reads all JSON files in the directory and unmarshals them into Emoji structs.
func loadEmojisFromDir(dir string) (map[string]models.Emoji, error) {
	m := make(map[string]models.Emoji)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, f.Name())
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var e models.Emoji
		if err := json.Unmarshal(b, &e); err != nil {
			return nil, err
		}
		m[e.Symbol] = e
	}

	return m, nil
}

// ServeHTTP implements http.Handler to expose the API.
func (h *EmojiMetaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Use(server.AuthMiddleware)
	r.Use(server.RateLimiterMiddleware)

	router := chi.NewRouter()
	router.Get("/emoji/{symbol}", h.handleGetEmoji)

	// CORS support
	c := cors.AllowAll()
	handler := c.Handler(router)

	handler.ServeHTTP(w, r)
}

// handleGetEmoji retrieves the metadata for a requested emoji symbol.
func (h *EmojiMetaHandler) handleGetEmoji(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		http.Error(w, `{"error":"emoji symbol required"}`, http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	e, ok := h.emojis[symbol]
	h.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"emoji not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(e); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

// ValidateInput ensures the emoji symbol is valid Unicode and not empty.
func ValidateInput(symbol string) error {
	if strings.TrimSpace(symbol) == "" {
		return errors.New("empty symbol")
	}
	// Basic Unicode check: ensure at least one rune
	if len([]rune(symbol)) == 0 {
		return errors.New("invalid unicode")
	}
	return nil
}