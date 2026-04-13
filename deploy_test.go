package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCommitProjectNeedsVMEndpoint(t *testing.T) {
	if !commitProjectNeedsVMEndpoint(errors.New("Taikun Error: (TITLE Bad request) (DETAIL You need at least one worker, an odd number of master(s) and one bastion to commit changes.)")) {
		t.Fatal("expected VM commit fallback for Kubernetes layout validation error")
	}

	if commitProjectNeedsVMEndpoint(errors.New("some other error")) {
		t.Fatal("did not expect VM commit fallback for unrelated errors")
	}
}

func TestCommitProjectFallbackDecisionUsesResponseBody(t *testing.T) {
	httpResponse := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"title":"Bad request","detail":"You need at least one worker, an odd number of master(s) and one bastion to commit changes."}`)),
	}

	shouldUseVMCommit, errorInfo := commitProjectFallbackDecision(httpResponse, nil)
	if errorInfo.Message == "" {
		t.Fatal("expected error info for non-2xx response")
	}
	if !shouldUseVMCommit {
		t.Fatal("expected VM commit fallback when response body contains cluster layout validation error")
	}
}
