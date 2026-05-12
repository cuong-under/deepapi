package client

import (
	"bufio"
	"bytes"
	"context"
	dsprotocol "ds2api/internal/deepseek/protocol"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	trans "ds2api/internal/deepseek/transport"
	"ds2api/internal/util"
)

func (c *Client) CallCompletion(ctx context.Context, a *auth.RequestAuth, payload map[string]any, powResp string, maxAttempts int) (*http.Response, error) {
	_ = maxAttempts
	clients := c.requestClientsForAuth(ctx, a)
	headers := c.authHeaders(a.DeepSeekToken)
	headers["x-ds-pow-response"] = powResp
	captureSession := c.capture.Start("deepseek_completion", dsprotocol.DeepSeekCompletionURL, a.AccountID, payload)
	resp, err := c.streamPostOnce(ctx, clients.stream, dsprotocol.DeepSeekCompletionURL, headers, payload)
	if err != nil {
		return nil, err
	}
	resp = normalizeCompletionJSONError(resp)
	if captureSession != nil {
		resp.Body = captureSession.WrapBody(resp.Body, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusOK {
		resp = c.wrapCompletionWithAutoContinue(ctx, a, payload, powResp, resp)
	}
	return resp, nil
}

func normalizeCompletionJSONError(resp *http.Response) *http.Response {
	if resp == nil || resp.Body == nil || resp.StatusCode != http.StatusOK {
		return resp
	}
	reader := bufio.NewReader(resp.Body)
	first, err := reader.Peek(1)
	if err != nil {
		resp.Body = readCloser{Reader: reader, closer: resp.Body}
		return resp
	}
	if len(first) == 0 || first[0] != '{' {
		resp.Body = readCloser{Reader: reader, closer: resp.Body}
		return resp
	}
	body, err := io.ReadAll(reader)
	closeErr := resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}
	if closeErr != nil {
		config.Logger.Warn("[deepseek_completion] response body close failed after json error read", "error", closeErr)
	}
	status, message, ok := completionJSONError(body)
	if !ok {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}
	resp.StatusCode = status
	resp.Status = http.StatusText(status)
	resp.Body = io.NopCloser(strings.NewReader(message))
	resp.ContentLength = int64(len(message))
	return resp
}

type readCloser struct {
	*bufio.Reader
	closer io.Closer
}

func (r readCloser) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

func completionJSONError(body []byte) (int, string, bool) {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, "", false
	}
	data, _ := parsed["data"].(map[string]any)
	bizCode := util.IntFrom(data["biz_code"])
	if bizCode == 0 {
		return 0, "", false
	}
	bizMsg := strings.TrimSpace(asString(data["biz_msg"]))
	if bizMsg == "" {
		bizMsg = strings.TrimSpace(asString(parsed["msg"]))
	}
	if bizMsg == "" {
		bizMsg = "DeepSeek upstream returned business error"
	}
	status := http.StatusBadGateway
	if bizCode == 5 || strings.Contains(strings.ToLower(bizMsg), "muted") {
		status = http.StatusTooManyRequests
	}
	return status, bizMsg, true
}

func (c *Client) streamPost(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any) (*http.Response, error) {
	return c.streamPostWithFallback(ctx, doer, url, headers, payload, true)
}

func (c *Client) streamPostOnce(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any) (*http.Response, error) {
	return c.streamPostWithFallback(ctx, doer, url, headers, payload, false)
}

func (c *Client) streamPostWithFallback(ctx context.Context, doer trans.Doer, url string, headers map[string]string, payload any, allowFallback bool) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers = c.jsonHeaders(headers)
	clients := c.requestClientsFromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := doer.Do(req)
	if err != nil {
		if allowFallback {
			config.Logger.Warn("[deepseek] fingerprint stream request failed, fallback to std transport", "url", url, "error", err)
			req2, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
			if reqErr != nil {
				return nil, reqErr
			}
			for k, v := range headers {
				req2.Header.Set(k, v)
			}
			return clients.fallbackS.Do(req2)
		}
		return nil, err
	}
	return resp, nil
}
