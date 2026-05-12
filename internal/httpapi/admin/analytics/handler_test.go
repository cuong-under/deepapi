package analytics

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/chathistory"
	"ds2api/internal/config"
	"ds2api/internal/database"
)

func TestScopeItemsFiltersNonAdminByUserID(t *testing.T) {
	h := &Handler{}
	items := []chathistory.SummaryEntry{
		{ID: "one", UserID: 1},
		{ID: "two", UserID: 2},
		{ID: "legacy", UserID: 0},
	}

	userCtx := auth.SetUserContext(t.Context(), 1, "user1", "user")
	userReq := httptest.NewRequest("GET", "/admin/analytics/overview", nil).WithContext(userCtx)
	userItems := h.scopeItemsForRequest(items, userReq)
	if len(userItems) != 1 || userItems[0].ID != "one" {
		t.Fatalf("expected only user 1 item, got %#v", userItems)
	}

	adminCtx := auth.SetUserContext(t.Context(), 99, "admin", "admin")
	adminReq := httptest.NewRequest("GET", "/admin/analytics/overview", nil).WithContext(adminCtx)
	adminItems := h.scopeItemsForRequest(items, adminReq)
	if len(adminItems) != len(items) {
		t.Fatalf("expected admin to see all items, got %d", len(adminItems))
	}
}

func TestAggregateStatsGroupsByUser(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	h := &Handler{ChatHistory: store}

	first := addCompletedHistoryEntry(t, store, 1, 100, 40)
	addCompletedHistoryEntry(t, store, 1, 20, 10)
	addCompletedHistoryEntry(t, store, 2, 50, 25)

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	day := time.UnixMilli(first.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "user", "", "", "")
	if len(stats) != 2 {
		t.Fatalf("expected stats for 2 users, got %d: %#v", len(stats), stats)
	}

	byUser := map[int64]TokenUsageStats{}
	for _, stat := range stats {
		byUser[stat.UserID] = stat
		if stat.AccountID != "" || stat.CallerID != "" || stat.Model != "" {
			t.Fatalf("user grouped stat should not expose account/caller/model identity: %#v", stat)
		}
	}

	if byUser[1].TotalTokens != 170 || byUser[1].RequestCount != 2 {
		t.Fatalf("bad user 1 aggregate: %#v", byUser[1])
	}
	if byUser[2].TotalTokens != 75 || byUser[2].RequestCount != 1 {
		t.Fatalf("bad user 2 aggregate: %#v", byUser[2])
	}
}

func TestAggregateStatsGroupsLegacyUserByAPIKeyOwner(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	db, err := database.Open(filepath.Join(t.TempDir(), "ds2api.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	user, err := db.CreateUser("user1", "user1@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = db.CreateAPIKey(user.ID, "user-key", "test key", "")
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}
	h := &Handler{ChatHistory: store, DB: db}

	entry := addCompletedHistoryEntryWithCaller(t, store, 0, callerTokenIDForAnalytics("user-key"), 100, 40)

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	day := time.UnixMilli(entry.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "user", "", "", "")
	if len(stats) != 1 {
		t.Fatalf("expected legacy api key owner stat, got %d: %#v", len(stats), stats)
	}
	if stats[0].UserID != user.ID || stats[0].UserLabel != "user1@example.com" || stats[0].TotalTokens != 140 {
		t.Fatalf("bad legacy aggregate: %#v", stats[0])
	}
}

func TestAggregateStatsCountsInMemoryIntegerUsage(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	h := &Handler{ChatHistory: store}

	entry, err := store.Start(chathistory.StartParams{
		CallerID:  "caller:test",
		AccountID: "account",
		UserID:    1,
		Model:     "deepseek-chat",
		UserInput: "hello",
	})
	if err != nil {
		t.Fatalf("start history entry: %v", err)
	}
	entry, err = store.Update(entry.ID, chathistory.UpdateParams{
		Status: "success",
		Usage: map[string]any{
			"prompt_tokens":     int64(11),
			"completion_tokens": 7,
			"total_tokens":      int(18),
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": uint(3),
			},
		},
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update history entry: %v", err)
	}

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	day := time.UnixMilli(entry.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "", "", "", "")
	if len(stats) != 1 {
		t.Fatalf("expected one aggregate, got %d: %#v", len(stats), stats)
	}
	if stats[0].PromptTokens != 11 || stats[0].CompletionTokens != 7 || stats[0].TotalTokens != 18 || stats[0].ReasoningTokens != 3 {
		t.Fatalf("bad integer usage aggregate: %#v", stats[0])
	}
}

func TestAggregateStatsCountsInputOutputUsageAliases(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	h := &Handler{ChatHistory: store}

	entry, err := store.Start(chathistory.StartParams{
		CallerID:  "caller:test",
		AccountID: "account",
		UserID:    1,
		Model:     "deepseek-chat",
		UserInput: "hello",
	})
	if err != nil {
		t.Fatalf("start history entry: %v", err)
	}
	entry, err = store.Update(entry.ID, chathistory.UpdateParams{
		Status: "success",
		Usage: map[string]any{
			"input_tokens":  int64(13),
			"output_tokens": int64(5),
		},
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update history entry: %v", err)
	}

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	day := time.UnixMilli(entry.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "", "", "", "")
	if len(stats) != 1 {
		t.Fatalf("expected one aggregate, got %d: %#v", len(stats), stats)
	}
	if stats[0].PromptTokens != 13 || stats[0].CompletionTokens != 5 || stats[0].TotalTokens != 18 {
		t.Fatalf("bad input/output usage aggregate: %#v", stats[0])
	}
}

func TestAggregateStatsUsesBasePricingForSearchModel(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	h := &Handler{
		ChatHistory: store,
		pricing: config.PricingConfig{
			Currency: "USD",
			Models: map[string]config.ModelPrice{
				"deepseek-v4-flash": {
					InputPricePer1M:  0.14,
					OutputPricePer1M: 0.28,
				},
			},
		},
	}

	entry, err := store.Start(chathistory.StartParams{
		CallerID:  "caller:test",
		AccountID: "account",
		UserID:    1,
		Model:     "deepseek-v4-flash-search",
		UserInput: "hello",
	})
	if err != nil {
		t.Fatalf("start history entry: %v", err)
	}
	entry, err = store.Update(entry.ID, chathistory.UpdateParams{
		Status: "success",
		Usage: map[string]any{
			"prompt_tokens":     int64(1_000_000),
			"completion_tokens": int64(1_000_000),
			"total_tokens":      int64(2_000_000),
		},
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update history entry: %v", err)
	}

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	day := time.UnixMilli(entry.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "", "", "", "")
	if len(stats) != 1 {
		t.Fatalf("expected one aggregate, got %d: %#v", len(stats), stats)
	}
	if stats[0].TotalCost < 0.4199 || stats[0].TotalCost > 0.4201 {
		t.Fatalf("expected base flash search cost 0.42, got %#v", stats[0])
	}
}

func TestAggregateStatsBillsTotalOnlyUsageAsInputTokens(t *testing.T) {
	store := chathistory.New(filepath.Join(t.TempDir(), "history.json"))
	h := &Handler{
		ChatHistory: store,
		pricing: config.PricingConfig{
			Currency: "USD",
			Models: map[string]config.ModelPrice{
				"deepseek-v4-flash": {
					InputPricePer1M:  0.14,
					OutputPricePer1M: 0.28,
				},
			},
		},
	}

	entry, err := store.Start(chathistory.StartParams{
		CallerID:  "caller:test",
		AccountID: "account",
		UserID:    1,
		Model:     "deepseek-v4-flash",
		UserInput: "hello",
	})
	if err != nil {
		t.Fatalf("start history entry: %v", err)
	}
	entry, err = store.Update(entry.ID, chathistory.UpdateParams{
		Status: "success",
		Usage: map[string]any{
			"total_tokens": int64(1_000_000),
		},
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update history entry: %v", err)
	}

	file, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	day := time.UnixMilli(entry.CreatedAt).UTC().Format("2006-01-02")
	stats := h.aggregateStats(file.Items, day, day, "", "", "", "")
	if len(stats) != 1 {
		t.Fatalf("expected one aggregate, got %d: %#v", len(stats), stats)
	}
	if stats[0].TotalCost < 0.1399 || stats[0].TotalCost > 0.1401 {
		t.Fatalf("expected total-only usage to bill as input tokens, got %#v", stats[0])
	}
}

func addCompletedHistoryEntry(t *testing.T, store *chathistory.Store, userID, promptTokens, completionTokens int64) chathistory.Entry {
	t.Helper()
	return addCompletedHistoryEntryWithCaller(t, store, userID, "key", promptTokens, completionTokens)
}

func addCompletedHistoryEntryWithCaller(t *testing.T, store *chathistory.Store, userID int64, callerID string, promptTokens, completionTokens int64) chathistory.Entry {
	t.Helper()

	entry, err := store.Start(chathistory.StartParams{
		CallerID:  callerID,
		AccountID: "account",
		UserID:    userID,
		Model:     "deepseek-chat",
		UserInput: "hello",
	})
	if err != nil {
		t.Fatalf("start history entry: %v", err)
	}

	entry, err = store.Update(entry.ID, chathistory.UpdateParams{
		Status: "success",
		Usage: map[string]any{
			"prompt_tokens":     float64(promptTokens),
			"completion_tokens": float64(completionTokens),
			"total_tokens":      float64(promptTokens + completionTokens),
		},
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update history entry: %v", err)
	}

	return entry
}
