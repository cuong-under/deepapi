package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/chathistory"
	"ds2api/internal/config"
	"ds2api/internal/database"
)

type Handler struct {
	ChatHistory *chathistory.Store
	Store       *config.Store
	DB          *database.DB
	pricing     config.PricingConfig // Cached pricing config
}

// SetPricing sets the pricing configuration
func (h *Handler) SetPricing(p config.PricingConfig) {
	h.pricing = p
	config.Logger.Info("[analytics] pricing initialized",
		"currency", p.Currency,
		"models_count", len(p.Models))
	for modelKey := range p.Models {
		config.Logger.Info("[analytics] pricing model loaded", "model", modelKey)
	}
}

// TokenUsageStats represents aggregated token usage statistics
type TokenUsageStats struct {
	Period           string  `json:"period"`     // Date in YYYY-MM-DD format
	AccountID        string  `json:"account_id"` // Empty for aggregated stats
	CallerID         string  `json:"caller_id"`  // Empty for aggregated stats
	Model            string  `json:"model"`      // Empty for aggregated stats
	UserID           int64   `json:"user_id"`    // Empty for aggregated stats
	UserLabel        string  `json:"user_label,omitempty"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	ReasoningTokens  int64   `json:"reasoning_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	ErrorCount       int64   `json:"error_count"`
	TotalCost        float64 `json:"total_cost"` // Calculated cost in configured currency
}

// OverviewStats represents high-level overview statistics
type OverviewStats struct {
	Today       TokenUsageStats   `json:"today"`
	Yesterday   TokenUsageStats   `json:"yesterday"`
	Last7Days   TokenUsageStats   `json:"last_7_days"`
	Last30Days  TokenUsageStats   `json:"last_30_days"`
	TopAccounts []TokenUsageStats `json:"top_accounts"`
	TopCallers  []TokenUsageStats `json:"top_callers"`
	TopModels   []TokenUsageStats `json:"top_models"`
	TopUsers    []TokenUsageStats `json:"top_users"`
	Currency    string            `json:"currency"`
}

// GetTokenUsage handles GET /admin/analytics/token-usage
func (h *Handler) GetTokenUsage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse query parameters
	startDate := query.Get("start_date") // YYYY-MM-DD
	endDate := query.Get("end_date")     // YYYY-MM-DD
	groupBy := query.Get("group_by")     // day, account, caller, model, user
	accountID := query.Get("account_id")
	callerID := query.Get("caller_id")
	model := query.Get("model")

	// Default to last 7 days if no date range specified
	now := time.Now()
	if endDate == "" {
		endDate = now.Format("2006-01-02")
	}
	if startDate == "" {
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	}

	// Load chat history
	file, err := h.ChatHistory.Snapshot()
	if err != nil {
		http.Error(w, "Failed to load chat history", http.StatusInternalServerError)
		return
	}

	items := h.scopeItemsForRequest(file.Items, r)

	// Aggregate statistics
	stats := h.aggregateStats(items, startDate, endDate, groupBy, accountID, callerID, model)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"stats":      stats,
		"start_date": startDate,
		"end_date":   endDate,
		"group_by":   groupBy,
	})
}

// GetOverview handles GET /admin/analytics/overview
func (h *Handler) GetOverview(w http.ResponseWriter, r *http.Request) {
	// Load chat history
	file, err := h.ChatHistory.Snapshot()
	if err != nil {
		http.Error(w, "Failed to load chat history", http.StatusInternalServerError)
		return
	}

	// Check if user is authenticated (multi-user mode)
	userID, hasUserID := auth.GetUserID(r.Context())
	isAdmin := auth.IsAdmin(r.Context())

	items := h.scopeItems(file.Items, userID, hasUserID, isAdmin)

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	sevenDaysAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
	thirtyDaysAgo := now.AddDate(0, 0, -30).Format("2006-01-02")

	overview := OverviewStats{
		Currency: "USD",
	}

	if h.pricing.Currency != "" {
		overview.Currency = h.pricing.Currency
	}

	// Today stats
	todayStats := h.aggregateStats(items, today, today, "", "", "", "")
	if len(todayStats) > 0 {
		overview.Today = todayStats[0]
	}

	// Yesterday stats
	yesterdayStats := h.aggregateStats(items, yesterday, yesterday, "", "", "", "")
	if len(yesterdayStats) > 0 {
		overview.Yesterday = yesterdayStats[0]
	}

	// Last 7 days stats
	last7Stats := h.aggregateStats(items, sevenDaysAgo, today, "", "", "", "")
	if len(last7Stats) > 0 {
		overview.Last7Days = last7Stats[0]
	}

	// Last 30 days stats
	last30Stats := h.aggregateStats(items, thirtyDaysAgo, today, "", "", "", "")
	if len(last30Stats) > 0 {
		overview.Last30Days = last30Stats[0]
	}

	// Top accounts (last 30 days)
	topAccounts := h.aggregateStats(items, thirtyDaysAgo, today, "account", "", "", "")
	sort.Slice(topAccounts, func(i, j int) bool {
		return topAccounts[i].TotalTokens > topAccounts[j].TotalTokens
	})
	if len(topAccounts) > 10 {
		topAccounts = topAccounts[:10]
	}
	overview.TopAccounts = topAccounts

	// Top callers (last 30 days)
	topCallers := h.aggregateStats(items, thirtyDaysAgo, today, "caller", "", "", "")
	sort.Slice(topCallers, func(i, j int) bool {
		return topCallers[i].TotalTokens > topCallers[j].TotalTokens
	})
	if len(topCallers) > 10 {
		topCallers = topCallers[:10]
	}
	overview.TopCallers = topCallers

	// Top models (last 30 days)
	topModels := h.aggregateStats(items, thirtyDaysAgo, today, "model", "", "", "")
	sort.Slice(topModels, func(i, j int) bool {
		return topModels[i].TotalTokens > topModels[j].TotalTokens
	})
	if len(topModels) > 10 {
		topModels = topModels[:10]
	}
	overview.TopModels = topModels

	topUsers := h.aggregateStats(items, thirtyDaysAgo, today, "user", "", "", "")
	sort.Slice(topUsers, func(i, j int) bool {
		return topUsers[i].TotalTokens > topUsers[j].TotalTokens
	})
	if len(topUsers) > 10 {
		topUsers = topUsers[:10]
	}
	overview.TopUsers = topUsers

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func (h *Handler) scopeItemsForRequest(items []chathistory.SummaryEntry, r *http.Request) []chathistory.SummaryEntry {
	userID, hasUserID := auth.GetUserID(r.Context())
	return h.scopeItems(items, userID, hasUserID, auth.IsAdmin(r.Context()))
}

func (h *Handler) scopeItems(items []chathistory.SummaryEntry, userID int64, hasUserID, isAdmin bool) []chathistory.SummaryEntry {
	if !hasUserID || isAdmin {
		return items
	}
	filteredItems := make([]chathistory.SummaryEntry, 0)
	for _, item := range items {
		if item.UserID == userID {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

func (h *Handler) aggregateStats(
	items []chathistory.SummaryEntry,
	startDate, endDate, groupBy, accountFilter, callerFilter, modelFilter string,
) []TokenUsageStats {
	// Parse date range in UTC
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	// Convert to UTC and set to start/end of day
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, time.UTC)

	// Map to aggregate stats
	statsMap := make(map[string]*TokenUsageStats)
	users := h.userDirectory()

	config.Logger.Info("[analytics] aggregating stats",
		"total_items", len(items),
		"start", start.Format(time.RFC3339),
		"end", end.Format(time.RFC3339))

	matchedCount := 0
	for _, item := range items {
		// Parse created_at timestamp (stored as milliseconds)
		itemTime := time.UnixMilli(item.CreatedAt).UTC()

		// Filter by date range
		if itemTime.Before(start) || itemTime.After(end) {
			config.Logger.Info("[analytics] item filtered out by date",
				"id", item.ID,
				"item_time", itemTime.Format(time.RFC3339),
				"start", start.Format(time.RFC3339),
				"end", end.Format(time.RFC3339))
			continue
		}
		matchedCount++
		config.Logger.Info("[analytics] item matched date filter",
			"id", item.ID,
			"item_time", itemTime.Format(time.RFC3339))

		// Filter by account
		if accountFilter != "" && item.AccountID != accountFilter {
			continue
		}

		// Filter by caller
		if callerFilter != "" && item.CallerID != callerFilter {
			continue
		}

		// Filter by model
		if modelFilter != "" && !strings.EqualFold(item.Model, modelFilter) {
			continue
		}

		// Load detail entry to get usage data
		detail, err := h.ChatHistory.Get(item.ID)
		if err != nil {
			config.Logger.Warn("[analytics] failed to load detail", "id", item.ID, "error", err)
			continue
		}

		config.Logger.Info("[analytics] loaded detail successfully", "id", item.ID, "has_usage", detail.Usage != nil)

		// Determine grouping key
		var key string
		userID, userLabel := h.resolveItemUser(item, users)
		switch groupBy {
		case "day":
			key = itemTime.Format("2006-01-02")
		case "account":
			key = "account:" + item.AccountID
		case "caller":
			key = "caller:" + item.CallerID
		case "model":
			key = "model:" + item.Model
		case "user":
			if userID == 0 {
				continue
			}
			key = "user:" + strconv.FormatInt(userID, 10)
		default:
			key = "total"
		}

		// Initialize stats if not exists
		if statsMap[key] == nil {
			statsMap[key] = &TokenUsageStats{
				Period:    itemTime.Format("2006-01-02"),
				AccountID: item.AccountID,
				CallerID:  item.CallerID,
				Model:     item.Model,
				UserID:    userID,
				UserLabel: userLabel,
			}

			// Set appropriate fields based on groupBy
			switch groupBy {
			case "account":
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
			case "caller":
				statsMap[key].AccountID = ""
				statsMap[key].Model = ""
			case "model":
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].UserID = 0
				statsMap[key].UserLabel = ""
			case "user":
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
			case "day":
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
				statsMap[key].UserID = 0
				statsMap[key].UserLabel = ""
			default:
				statsMap[key].Period = startDate + " to " + endDate
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
				statsMap[key].UserID = 0
				statsMap[key].UserLabel = ""
			}
		}

		stat := statsMap[key]

		// Extract usage from detail entry
		usage := detail.Usage
		var entryPromptTokens, entryCompletionTokens, entryTotalTokens int64
		if usage != nil {
			entryPromptTokens = usageInt64(usage, "prompt_tokens", "input_tokens")
			entryCompletionTokens = usageInt64(usage, "completion_tokens", "output_tokens")
			entryTotalTokens = usageInt64(usage, "total_tokens")
			if entryTotalTokens == 0 {
				entryTotalTokens = entryPromptTokens + entryCompletionTokens
			}

			stat.PromptTokens += entryPromptTokens
			stat.CompletionTokens += entryCompletionTokens
			stat.TotalTokens += entryTotalTokens

			// Extract reasoning tokens from completion_tokens_details
			if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
				stat.ReasoningTokens += usageInt64(details, "reasoning_tokens")
			}
		}

		config.Logger.Info("[analytics] extracted tokens",
			"entry_id", item.ID,
			"model", item.Model,
			"prompt_tokens", entryPromptTokens,
			"completion_tokens", entryCompletionTokens,
			"has_store", h.Store != nil)

		// Count requests
		stat.RequestCount++
		if item.Status == "success" {
			stat.SuccessCount++
		} else if item.Status == "error" {
			stat.ErrorCount++
		}

		// Calculate cost for this entry and add to total
		config.Logger.Info("[analytics] checking cost calculation",
			"entry_id", item.ID,
			"entryPromptTokens", entryPromptTokens,
			"has_pricing_models", h.pricing.Models != nil,
			"pricing_models_count", len(h.pricing.Models))

		if entryPromptTokens > 0 && h.pricing.Models != nil {
			modelKey := strings.ToLower(item.Model)
			if pricing, ok := h.pricing.Models[modelKey]; ok {
				inputCost := float64(entryPromptTokens) / 1_000_000 * pricing.InputPricePer1M
				outputCost := float64(entryCompletionTokens) / 1_000_000 * pricing.OutputPricePer1M
				entryCost := inputCost + outputCost
				stat.TotalCost += entryCost
				config.Logger.Info("[analytics] calculated entry cost",
					"entry_id", item.ID,
					"model", item.Model,
					"model_key", modelKey,
					"prompt_tokens", entryPromptTokens,
					"completion_tokens", entryCompletionTokens,
					"entry_cost", entryCost,
					"accumulated_cost", stat.TotalCost)
			} else {
				config.Logger.Warn("[analytics] no pricing for model",
					"model", item.Model,
					"model_key", modelKey)
			}
		}
	}

	// Convert map to slice
	result := make([]TokenUsageStats, 0, len(statsMap))
	for _, stat := range statsMap {
		result = append(result, *stat)
	}

	config.Logger.Info("[analytics] aggregation complete",
		"matched_items", matchedCount,
		"result_count", len(result))

	// Sort by period/key
	sort.Slice(result, func(i, j int) bool {
		return result[i].Period < result[j].Period
	})

	return result
}

type userDirectory struct {
	byID     map[int64]string
	byCaller map[string]int64
}

func (h *Handler) userDirectory() userDirectory {
	dir := userDirectory{
		byID:     map[int64]string{},
		byCaller: map[string]int64{},
	}
	if h == nil || h.DB == nil {
		return dir
	}
	users, _, err := h.DB.ListUsers(1000, 0)
	if err != nil {
		config.Logger.Warn("[analytics] failed to list users for labels", "error", err)
		return dir
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		label := strings.TrimSpace(user.Email)
		if label == "" {
			label = strings.TrimSpace(user.Username)
		}
		if label != "" {
			dir.byID[user.ID] = label
		}
	}
	keys, err := h.DB.GetAllAPIKeys()
	if err != nil {
		config.Logger.Warn("[analytics] failed to list api keys for user mapping", "error", err)
		return dir
	}
	for _, key := range keys {
		if key == nil {
			continue
		}
		callerID := callerTokenIDForAnalytics(key.APIKey)
		if callerID != "" {
			dir.byCaller[callerID] = key.UserID
		}
	}
	return dir
}

func (h *Handler) resolveItemUser(item chathistory.SummaryEntry, dir userDirectory) (int64, string) {
	userID := item.UserID
	if userID == 0 && item.CallerID != "" {
		userID = dir.byCaller[item.CallerID]
	}
	if userID == 0 {
		return 0, ""
	}
	return userID, dir.byID[userID]
}

func callerTokenIDForAnalytics(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return "caller:" + hex.EncodeToString(sum[:8])
}

func usageInt64(usage map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := usage[key]; ok {
			if n, ok := anyInt64(value); ok {
				return n
			}
		}
	}
	return 0
}

func anyInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return n, true
		}
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return int64(f), true
	default:
		return 0, false
	}
}
