package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestProjectBackupReadErrorResponseTranslatesMissingKubeconfig(t *testing.T) {
	response := projectBackupReadErrorResponse(
		77,
		fakeHTTPErrorResponse(http.StatusBadRequest, `{"detail":"There is no kubeconfig file for this project"}`),
		fmt.Errorf("bad request"),
	)

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "project 77") || !strings.Contains(strings.ToLower(result.Error), "kubeconfig") {
		t.Fatalf("expected clearer project backup prerequisite message, got %+v", result)
	}
	if !strings.Contains(result.Details, "There is no kubeconfig file for this project") {
		t.Fatalf("expected original API error in details, got %+v", result)
	}
}

func TestProjectBackupReadErrorResponsePreservesOtherErrors(t *testing.T) {
	response := projectBackupReadErrorResponse(
		77,
		fakeHTTPErrorResponse(http.StatusNotFound, `{"detail":"backup not found"}`),
		fmt.Errorf("not found"),
	)

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(strings.ToLower(result.Error), "backup not found") {
		t.Fatalf("expected non-kubeconfig errors to pass through, got %+v", result)
	}
	if result.Details != "" {
		t.Fatalf("expected passthrough errors to avoid extra details wrapping, got %+v", result)
	}
}
