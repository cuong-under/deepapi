package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"ds2api/internal/auth"
)

func TestCallCompletionDoesNotFallbackForNonIdempotentCompletion(t *testing.T) {
	var fallbackCalled bool
	client := &Client{
		stream: doerFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("ambiguous completion write failure")
		}),
		fallbackS: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			fallbackCalled = true
			return &http.Response{StatusCode: http.StatusOK}, nil
		})},
	}
	_, err := client.CallCompletion(
		context.Background(),
		&auth.RequestAuth{DeepSeekToken: "token"},
		map[string]any{"prompt": "hello"},
		"pow",
		3,
	)
	if err == nil {
		t.Fatal("expected completion error")
	}
	if fallbackCalled {
		t.Fatal("completion fallback should not be called for a non-idempotent request")
	}
}

func TestCallCompletionMapsMutedJSONBusinessError(t *testing.T) {
	client := &Client{
		stream: doerFunc(func(*http.Request) (*http.Response, error) {
			body := `{"code":0,"msg":"","data":{"biz_code":5,"biz_msg":"user is muted","biz_data":{"is_muted":1}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}
	resp, err := client.CallCompletion(
		context.Background(),
		&auth.RequestAuth{DeepSeekToken: "token"},
		map[string]any{"prompt": "hello"},
		"pow",
		3,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for muted account, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if got := string(body); got != "user is muted" {
		t.Fatalf("unexpected body %q", got)
	}
}
