package proxies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/config"
	dsclient "ds2api/internal/deepseek/client"
)

var proxyConnectivityTester = func(ctx context.Context, proxy config.Proxy) map[string]any {
	return dsclient.TestProxyConnectivity(ctx, proxy)
}

func validateProxyMutation(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	if err := config.ValidateProxyConfig(cfg.Proxies); err != nil {
		return err
	}
	return config.ValidateAccountProxyReferences(cfg.Accounts, cfg.Proxies)
}

func proxyResponse(proxy config.Proxy) map[string]any {
	proxy = config.NormalizeProxy(proxy)
	return map[string]any{
		"id":           proxy.ID,
		"name":         proxy.Name,
		"type":         proxy.Type,
		"host":         proxy.Host,
		"port":         proxy.Port,
		"username":     proxy.Username,
		"has_password": strings.TrimSpace(proxy.Password) != "",
	}
}

func (h *Handler) listProxies(w http.ResponseWriter, _ *http.Request) {
	proxies := h.Store.Snapshot().Proxies
	items := make([]map[string]any, 0, len(proxies))
	for _, proxy := range proxies {
		proxy = config.NormalizeProxy(proxy)
		items = append(items, map[string]any{
			"id":           proxy.ID,
			"name":         proxy.Name,
			"type":         proxy.Type,
			"host":         proxy.Host,
			"port":         proxy.Port,
			"username":     proxy.Username,
			"has_password": strings.TrimSpace(proxy.Password) != "",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

func (h *Handler) addProxy(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	proxy := toProxy(req)
	err := h.Store.Update(func(c *config.Config) error {
		c.Proxies = append(c.Proxies, proxy)
		return validateProxyMutation(c)
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "proxy": proxyResponse(proxy)})
}

type proxyImportError struct {
	Line    int    `json:"line"`
	Content string `json:"content"`
	Error   string `json:"error"`
}

func (h *Handler) importProxies(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)

	raw := fieldString(req, "text")
	if raw == "" {
		raw = fieldString(req, "proxies")
	}
	proxyType := strings.ToLower(strings.TrimSpace(fieldString(req, "type")))
	if proxyType == "" {
		proxyType = "socks5h"
	}

	parsed, parseErrors := parseProxyImportLines(raw, proxyType)
	if len(parsed) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"detail": "no valid proxies to import",
			"errors": parseErrors,
		})
		return
	}

	var imported []map[string]any
	var skipped []map[string]any
	err := h.Store.Update(func(c *config.Config) error {
		existing := make(map[string]struct{}, len(c.Proxies)+len(parsed))
		for _, proxy := range c.Proxies {
			proxy = config.NormalizeProxy(proxy)
			existing[proxyImportDedupeKey(proxy)] = struct{}{}
		}

		for _, proxy := range parsed {
			proxy = config.NormalizeProxy(proxy)
			key := proxyImportDedupeKey(proxy)
			if _, ok := existing[key]; ok {
				skipped = append(skipped, proxyResponse(proxy))
				continue
			}
			c.Proxies = append(c.Proxies, proxy)
			existing[key] = struct{}{}
			imported = append(imported, proxyResponse(proxy))
		}

		return validateProxyMutation(c)
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error(), "errors": parseErrors})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"imported":       imported,
		"imported_count": len(imported),
		"skipped":        skipped,
		"skipped_count":  len(skipped),
		"errors":         parseErrors,
		"error_count":    len(parseErrors),
	})
}

func parseProxyImportLines(raw string, proxyType string) ([]config.Proxy, []proxyImportError) {
	var proxies []config.Proxy
	var errors []proxyImportError

	for index, line := range strings.Split(raw, "\n") {
		lineNo := index + 1
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		proxy, err := parseProxyImportLine(line, proxyType)
		if err != nil {
			errors = append(errors, proxyImportError{
				Line:    lineNo,
				Content: line,
				Error:   err.Error(),
			})
			continue
		}
		proxies = append(proxies, proxy)
	}

	return proxies, errors
}

func parseProxyImportLine(line string, proxyType string) (config.Proxy, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 4 {
		return config.Proxy{}, fmt.Errorf("expected host:port:username:password")
	}

	host := strings.TrimSpace(parts[0])
	portRaw := strings.TrimSpace(parts[1])
	username := strings.TrimSpace(parts[2])
	password := strings.TrimSpace(parts[3])
	if host == "" || portRaw == "" || username == "" || password == "" {
		return config.Proxy{}, fmt.Errorf("host, port, username and password are required")
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return config.Proxy{}, fmt.Errorf("invalid port")
	}

	proxy := config.NormalizeProxy(config.Proxy{
		Type:     proxyType,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	})
	if proxy.Name == "" {
		proxy.Name = fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)
	}
	if err := config.ValidateProxyConfig([]config.Proxy{proxy}); err != nil {
		return config.Proxy{}, err
	}

	return proxy, nil
}

func proxyImportDedupeKey(proxy config.Proxy) string {
	proxy = config.NormalizeProxy(proxy)
	return strings.Join([]string{
		proxy.Type,
		strings.ToLower(proxy.Host),
		strconv.Itoa(proxy.Port),
		proxy.Username,
	}, "|")
}

func (h *Handler) updateProxy(w http.ResponseWriter, r *http.Request) {
	proxyID := chi.URLParam(r, "proxyID")
	if decoded, err := url.PathUnescape(proxyID); err == nil {
		proxyID = decoded
	}
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	proxy := toProxy(req)
	proxy.ID = strings.TrimSpace(proxyID)

	err := h.Store.Update(func(c *config.Config) error {
		for i, existing := range c.Proxies {
			existing = config.NormalizeProxy(existing)
			if existing.ID != proxy.ID {
				continue
			}
			if proxy.Password == "" {
				proxy.Password = existing.Password
			}
			c.Proxies[i] = proxy
			return validateProxyMutation(c)
		}
		return newRequestError("代理不存在")
	})
	if err != nil {
		if detail, ok := requestErrorDetail(err); ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": detail})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "proxy": proxyResponse(proxy)})
}

func (h *Handler) deleteProxy(w http.ResponseWriter, r *http.Request) {
	proxyID := chi.URLParam(r, "proxyID")
	if decoded, err := url.PathUnescape(proxyID); err == nil {
		proxyID = decoded
	}
	err := h.Store.Update(func(c *config.Config) error {
		idx := -1
		for i, existing := range c.Proxies {
			existing = config.NormalizeProxy(existing)
			if existing.ID == strings.TrimSpace(proxyID) {
				idx = i
				break
			}
		}
		if idx < 0 {
			return newRequestError("代理不存在")
		}
		c.Proxies = append(c.Proxies[:idx], c.Proxies[idx+1:]...)
		for i := range c.Accounts {
			if strings.TrimSpace(c.Accounts[i].ProxyID) == strings.TrimSpace(proxyID) {
				c.Accounts[i].ProxyID = ""
			}
		}
		return validateProxyMutation(c)
	})
	if err != nil {
		if detail, ok := requestErrorDetail(err); ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": detail})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) testProxy(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	proxyID := fieldString(req, "proxy_id")

	var proxy config.Proxy
	if proxyID != "" {
		var ok bool
		proxy, ok = findProxyByID(h.Store.Snapshot(), proxyID)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "代理不存在"})
			return
		}
	} else {
		proxy = toProxy(req)
	}

	result := proxyConnectivityTester(r.Context(), proxy)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) updateAccountProxy(w http.ResponseWriter, r *http.Request) {
	identifier := chi.URLParam(r, "identifier")
	if decoded, err := url.PathUnescape(identifier); err == nil {
		identifier = decoded
	}
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	proxyID := fieldString(req, "proxy_id")

	err := h.Store.Update(func(c *config.Config) error {
		if proxyID != "" {
			if _, ok := findProxyByID(*c, proxyID); !ok {
				return newRequestError("代理不存在")
			}
		}
		for i, acc := range c.Accounts {
			if !accountMatchesIdentifier(acc, identifier) {
				continue
			}
			c.Accounts[i].ProxyID = proxyID
			return validateProxyMutation(c)
		}
		return newRequestError("账号不存在")
	})
	if err != nil {
		if detail, ok := requestErrorDetail(err); ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": detail})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
		return
	}
	h.Pool.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "proxy_id": proxyID})
}
