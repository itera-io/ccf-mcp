package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
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

func resetPendingProjectCommitChangesForTest(t *testing.T) {
	t.Helper()

	projectPendingCommitChangesMu.Lock()
	defer projectPendingCommitChangesMu.Unlock()

	projectPendingCommitChanges = map[int32]pendingProjectCommitChanges{}
}

func TestPendingProjectCommitModeStandaloneVMCreateOnly(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(101)
	recordPendingStandaloneVMCreate(projectID)

	if got := pendingProjectCommitMode(projectID); got != projectCommitModeVM {
		t.Fatalf("expected %q, got %q", projectCommitModeVM, got)
	}
}

func TestPendingProjectCommitModeServerAddOnly(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(102)
	recordPendingServerAdd(projectID)

	if got := pendingProjectCommitMode(projectID); got != projectCommitModeProject {
		t.Fatalf("expected %q, got %q", projectCommitModeProject, got)
	}
}

func TestPendingProjectCommitModeMixedChangesPreferProject(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(103)
	recordPendingStandaloneVMCreate(projectID)
	recordPendingServerAdd(projectID)

	if got := pendingProjectCommitMode(projectID); got != projectCommitModeProject {
		t.Fatalf("expected %q for mixed changes, got %q", projectCommitModeProject, got)
	}
}

func TestExecuteProjectCommitModeClearsPendingStateAfterSuccessfulVMCommit(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(104)
	recordPendingStandaloneVMCreate(projectID)

	called := ""
	result, errorInfo := executeProjectCommitMode(
		projectID,
		pendingProjectCommitMode(projectID),
		func() (projectCommitResult, *apiErrorInfo) {
			called = "project"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "fallback"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "vm"
			return projectCommitResult{Mode: "vm", Message: "ok"}, nil
		},
	)
	if errorInfo != nil {
		t.Fatalf("expected successful VM commit, got error %+v", errorInfo)
	}
	if called != "vm" {
		t.Fatalf("expected vm commit path, got %q", called)
	}
	if result.Mode != "vm" {
		t.Fatalf("expected vm result mode, got %q", result.Mode)
	}
	if got := pendingProjectCommitMode(projectID); got != projectCommitModeAuto {
		t.Fatalf("expected pending state to clear after success, got %q", got)
	}
}

func TestExecuteProjectCommitModeClearsPendingStateAfterSuccessfulProjectCommit(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(105)
	recordPendingStandaloneVMCreate(projectID)
	recordPendingServerAdd(projectID)

	called := ""
	result, errorInfo := executeProjectCommitMode(
		projectID,
		pendingProjectCommitMode(projectID),
		func() (projectCommitResult, *apiErrorInfo) {
			called = "project"
			return projectCommitResult{Mode: "project", Message: "ok"}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "fallback"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "vm"
			return projectCommitResult{}, nil
		},
	)
	if errorInfo != nil {
		t.Fatalf("expected successful project commit, got error %+v", errorInfo)
	}
	if called != "project" {
		t.Fatalf("expected project commit path, got %q", called)
	}
	if result.Mode != "project" {
		t.Fatalf("expected project result mode, got %q", result.Mode)
	}
	if got := pendingProjectCommitMode(projectID); got != projectCommitModeAuto {
		t.Fatalf("expected pending state to clear after success, got %q", got)
	}
}

func TestExecuteProjectCommitModePreservesPendingStateAfterFailedVMCommit(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(106)
	recordPendingStandaloneVMCreate(projectID)

	called := ""
	_, errorInfo := executeProjectCommitMode(
		projectID,
		pendingProjectCommitMode(projectID),
		func() (projectCommitResult, *apiErrorInfo) {
			called = "project"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "fallback"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "vm"
			return projectCommitResult{}, &apiErrorInfo{Message: "vm commit failed"}
		},
	)
	if errorInfo == nil {
		t.Fatal("expected VM commit failure")
	}
	if called != "vm" {
		t.Fatalf("expected vm commit path, got %q", called)
	}
	if got := pendingProjectCommitMode(projectID); got != projectCommitModeVM {
		t.Fatalf("expected pending VM state to remain after failure, got %q", got)
	}
}

func TestExecuteProjectCommitModeUsesFallbackWhenNoTrackedChanges(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(107)
	called := ""
	result, errorInfo := executeProjectCommitMode(
		projectID,
		pendingProjectCommitMode(projectID),
		func() (projectCommitResult, *apiErrorInfo) {
			called = "project"
			return projectCommitResult{}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "fallback"
			return projectCommitResult{Mode: "project", Message: "ok"}, nil
		},
		func() (projectCommitResult, *apiErrorInfo) {
			called = "vm"
			return projectCommitResult{}, nil
		},
	)
	if errorInfo != nil {
		t.Fatalf("expected successful fallback commit path, got error %+v", errorInfo)
	}
	if called != "fallback" {
		t.Fatalf("expected fallback commit path, got %q", called)
	}
	if result.Mode != "project" {
		t.Fatalf("expected project result mode from fallback path, got %q", result.Mode)
	}
}

func TestCommitProjectWithFallbackPreservesReactiveFallbackInTrackedProjectMode(t *testing.T) {
	resetPendingProjectCommitChangesForTest(t)

	projectID := int32(108)
	recordPendingServerAdd(projectID)

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch callCount {
		case 1:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"title":"Bad request","detail":"You need at least one worker, an odd number of master(s) and one bastion to commit changes."}`))
		case 2:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected extra request %d to %s", callCount, r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := taikuncore.NewConfiguration()
	cfg.Scheme = "http"
	cfg.Host = strings.TrimPrefix(server.URL, "http://")
	client := &taikungoclient.Client{
		Client: taikuncore.NewAPIClient(cfg),
	}

	result, errorInfo := commitProjectWithFallback(client, projectID)
	if errorInfo != nil {
		t.Fatalf("expected reactive fallback to recover tracked project-mode commit, got error %+v", errorInfo)
	}
	if result.Mode != "vm" {
		t.Fatalf("expected VM commit result after fallback, got %q", result.Mode)
	}
	if callCount != 2 {
		t.Fatalf("expected project commit plus VM fallback, got %d request(s)", callCount)
	}
	if got := pendingProjectCommitMode(projectID); got != projectCommitModeAuto {
		t.Fatalf("expected pending state to clear after successful fallback, got %q", got)
	}
}
