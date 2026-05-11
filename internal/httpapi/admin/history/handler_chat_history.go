package history

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/chathistory"
)

func (h *Handler) getChatHistory(w http.ResponseWriter, r *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}

	// Get user context (multi-user mode)
	userCtx, hasUserCtx := auth.GetUserContext(r.Context())
	isAdmin := hasUserCtx && userCtx.Role == "admin"

	ifNoneMatch := strings.TrimSpace(r.Header.Get("If-None-Match"))
	if ifNoneMatch != "" {
		revision, err := store.Revision()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"detail": err.Error(),
				"path":   store.Path(),
			})
			return
		}
		etag := chathistory.ListETag(revision)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		if ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"detail": err.Error(),
			"path":   store.Path(),
		})
		return
	}

	// Filter items by UserID (if not admin)
	items := snapshot.Items
	if hasUserCtx && !isAdmin {
		filteredItems := make([]chathistory.SummaryEntry, 0)
		for _, item := range snapshot.Items {
			// Only show items that belong to this user (skip legacy items without user_id)
			if item.UserID == userCtx.UserID && item.UserID != 0 {
				filteredItems = append(filteredItems, item)
			}
		}
		items = filteredItems
	} else if hasUserCtx && isAdmin {
		// Admin can see all, but filter out legacy items without user_id in multi-user mode
		filteredItems := make([]chathistory.SummaryEntry, 0)
		for _, item := range snapshot.Items {
			if item.UserID != 0 {
				filteredItems = append(filteredItems, item)
			}
		}
		items = filteredItems
	}

	etag := chathistory.ListETag(snapshot.Revision)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if ifNoneMatch == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":  snapshot.Version,
		"limit":    snapshot.Limit,
		"revision": snapshot.Revision,
		"items":    items,
		"path":     store.Path(),
	})
}

func (h *Handler) getChatHistoryItem(w http.ResponseWriter, r *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}

	// Get user context (multi-user mode)
	userCtx, hasUserCtx := auth.GetUserContext(r.Context())
	isAdmin := hasUserCtx && userCtx.Role == "admin"

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "history id is required"})
		return
	}
	ifNoneMatch := strings.TrimSpace(r.Header.Get("If-None-Match"))
	if ifNoneMatch != "" {
		revision, err := store.DetailRevision(id)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"detail": err.Error()})
			return
		}
		etag := chathistory.DetailETag(id, revision)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		if ifNoneMatch == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	item, err := store.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"detail": err.Error()})
		return
	}

	// Check ownership (if not admin)
	if hasUserCtx && !isAdmin && item.UserID != userCtx.UserID {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "access denied"})
		return
	}

	etag := chathistory.DetailETag(item.ID, item.Revision)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if ifNoneMatch == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"item": item,
	})
}

func (h *Handler) clearChatHistory(w http.ResponseWriter, _ *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}
	if err := store.Clear(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": err.Error(), "path": store.Path()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) deleteChatHistoryItem(w http.ResponseWriter, r *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}

	// Get user context (multi-user mode)
	userCtx, hasUserCtx := auth.GetUserContext(r.Context())
	isAdmin := hasUserCtx && userCtx.Role == "admin"

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "history id is required"})
		return
	}

	// Check ownership before deleting (if not admin)
	if hasUserCtx && !isAdmin {
		item, err := store.Get(id)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"detail": err.Error()})
			return
		}
		if item.UserID != userCtx.UserID {
			writeJSON(w, http.StatusForbidden, map[string]any{"detail": "access denied"})
			return
		}
	}

	if err := store.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) updateChatHistorySettings(w http.ResponseWriter, r *http.Request) {
	store := h.ChatHistory
	if store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "chat history store is not configured"})
		return
	}
	var body struct {
		Limit int `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid json"})
		return
	}
	snapshot, err := store.SetLimit(body.Limit)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"limit":    snapshot.Limit,
		"revision": snapshot.Revision,
		"items":    snapshot.Items,
	})
}
