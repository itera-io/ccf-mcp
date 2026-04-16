package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestReadResponseBodyPreservingBodyAllowsRepeatedReads(t *testing.T) {
	response := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{"detail":"still readable"}`)),
	}

	firstRead, err := readResponseBodyPreservingBody(response)
	if err != nil {
		t.Fatalf("expected first read to succeed, got error: %v", err)
	}
	if string(firstRead) != `{"detail":"still readable"}` {
		t.Fatalf("unexpected first read body: %q", string(firstRead))
	}

	secondRead, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("expected second read to succeed, got error: %v", err)
	}
	if string(secondRead) != `{"detail":"still readable"}` {
		t.Fatalf("expected body to remain readable after helper, got %q", string(secondRead))
	}
}

func TestAPIErrorInfoFromResponsePreservesBodyForLaterReaders(t *testing.T) {
	const body = `{"title":"Bad Gateway","detail":"upstream failed"}`
	requestURL, err := url.Parse("https://api.example.test/api/v1/projectapp/list")
	if err != nil {
		t.Fatalf("failed to parse request URL: %v", err)
	}

	response := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request: &http.Request{
			Method: http.MethodGet,
			URL:    requestURL,
		},
	}

	errorInfo := apiErrorInfoFromResponse(response, nil)
	if !strings.Contains(errorInfo.Message, "upstream failed") {
		t.Fatalf("expected error message to include API detail, got %q", errorInfo.Message)
	}

	bodyAfterErrorInfo, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("expected response body to remain readable, got error: %v", err)
	}
	if string(bodyAfterErrorInfo) != body {
		t.Fatalf("expected preserved body %q, got %q", body, string(bodyAfterErrorInfo))
	}
}
