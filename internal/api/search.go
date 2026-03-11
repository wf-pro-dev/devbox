package api

import (
	"log"
	"net/http"

	"github.com/wf-pro-dev/devbox/internal/search"
)

type searchHandler struct {
	searcher *search.Searcher
}

// GET /search?q=<query>
func (h *searchHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		jsonError(w, "'q' query parameter is required", http.StatusBadRequest)
		return
	}

	results, err := h.searcher.Search(r.Context(), q)
	if err != nil {
		jsonError(w, "search failed", http.StatusInternalServerError)
		log.Printf("search %q: %v", q, err)
		return
	}

	jsonOK(w, results)
}
