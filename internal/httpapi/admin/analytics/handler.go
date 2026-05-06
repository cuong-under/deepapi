package analytics

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"ds2api/internal/chathistory"
	"ds2api/internal/config"
)

type Handler struct {
	ChatHistory *chathistory.Store
	Store       *config.Store
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
	Period          string  `json:"period"`           // Date in YYYY-MM-DD format
	AccountID       string  `json:"account_id"`       // Empty for aggregated stats
	CallerID        string  `json:"caller_id"`        // Empty for aggregated stats
	Model           string  `json:"model"`            // Empty for aggregated stats
	PromptTokens    int64   `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	TotalTokens     int64   `json:"total_tokens"`
	RequestCount    int64   `json:"request_count"`
	SuccessCount    int64   `json:"success_count"`
	ErrorCount      int64   `json:"error_count"`
	TotalCost       float64 `json:"total_cost"`       // Calculated cost in configured currency
}

// OverviewStats represents high-level overview statistics
type OverviewStats struct {
	Today              TokenUsageStats   `json:"today"`
	Yesterday          TokenUsageStats   `json:"yesterday"`
	Last7Days          TokenUsageStats   `json:"last_7_days"`
	Last30Days         TokenUsageStats   `json:"last_30_days"`
	TopAccounts        []TokenUsageStats `json:"top_accounts"`
	TopCallers         []TokenUsageStats `json:"top_callers"`
	TopModels          []TokenUsageStats `json:"top_models"`
	Currency           string            `json:"currency"`
}

// GetTokenUsage handles GET /admin/analytics/token-usage
func (h *Handler) GetTokenUsage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse query parameters
	startDate := query.Get("start_date") // YYYY-MM-DD
	endDate := query.Get("end_date")     // YYYY-MM-DD
	groupBy := query.Get("group_by")     // day, account, caller, model
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

	// Aggregate statistics
	stats := h.aggregateStats(file.Items, startDate, endDate, groupBy, accountID, callerID, model)

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
	todayStats := h.aggregateStats(file.Items, today, today, "", "", "", "")
	if len(todayStats) > 0 {
		overview.Today = todayStats[0]
	}

	// Yesterday stats
	yesterdayStats := h.aggregateStats(file.Items, yesterday, yesterday, "", "", "", "")
	if len(yesterdayStats) > 0 {
		overview.Yesterday = yesterdayStats[0]
	}

	// Last 7 days stats
	last7Stats := h.aggregateStats(file.Items, sevenDaysAgo, today, "", "", "", "")
	if len(last7Stats) > 0 {
		overview.Last7Days = last7Stats[0]
	}

	// Last 30 days stats
	last30Stats := h.aggregateStats(file.Items, thirtyDaysAgo, today, "", "", "", "")
	if len(last30Stats) > 0 {
		overview.Last30Days = last30Stats[0]
	}

	// Top accounts (last 30 days)
	topAccounts := h.aggregateStats(file.Items, thirtyDaysAgo, today, "account", "", "", "")
	sort.Slice(topAccounts, func(i, j int) bool {
		return topAccounts[i].TotalTokens > topAccounts[j].TotalTokens
	})
	if len(topAccounts) > 10 {
		topAccounts = topAccounts[:10]
	}
	overview.TopAccounts = topAccounts

	// Top callers (last 30 days)
	topCallers := h.aggregateStats(file.Items, thirtyDaysAgo, today, "caller", "", "", "")
	sort.Slice(topCallers, func(i, j int) bool {
		return topCallers[i].TotalTokens > topCallers[j].TotalTokens
	})
	if len(topCallers) > 10 {
		topCallers = topCallers[:10]
	}
	overview.TopCallers = topCallers

	// Top models (last 30 days)
	topModels := h.aggregateStats(file.Items, thirtyDaysAgo, today, "model", "", "", "")
	sort.Slice(topModels, func(i, j int) bool {
		return topModels[i].TotalTokens > topModels[j].TotalTokens
	})
	if len(topModels) > 10 {
		topModels = topModels[:10]
	}
	overview.TopModels = topModels

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
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
		switch groupBy {
		case "day":
			key = itemTime.Format("2006-01-02")
		case "account":
			key = "account:" + item.AccountID
		case "caller":
			key = "caller:" + item.CallerID
		case "model":
			key = "model:" + item.Model
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
			case "day":
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
			default:
				statsMap[key].Period = startDate + " to " + endDate
				statsMap[key].AccountID = ""
				statsMap[key].CallerID = ""
				statsMap[key].Model = ""
			}
		}

		stat := statsMap[key]

		// Extract usage from detail entry
		usage := detail.Usage
		var entryPromptTokens, entryCompletionTokens int64
		if usage != nil {
			if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
				entryPromptTokens = int64(promptTokens)
				stat.PromptTokens += entryPromptTokens
			}
			if completionTokens, ok := usage["completion_tokens"].(float64); ok {
				entryCompletionTokens = int64(completionTokens)
				stat.CompletionTokens += entryCompletionTokens
			}
			if totalTokens, ok := usage["total_tokens"].(float64); ok {
				stat.TotalTokens += int64(totalTokens)
			}

			// Extract reasoning tokens from completion_tokens_details
			if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
				if reasoningTokens, ok := details["reasoning_tokens"].(float64); ok {
					stat.ReasoningTokens += int64(reasoningTokens)
				}
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
