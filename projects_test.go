package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

func fakeHTTPErrorResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func decodeToolResponseJSON[T any](t *testing.T, response *mcp_golang.ToolResponse) T {
	t.Helper()

	if response == nil {
		t.Fatal("expected response, got nil")
	}
	if len(response.Content) == 0 || response.Content[0].TextContent == nil {
		t.Fatalf("expected text content in response, got %+v", response)
	}

	var result T
	if err := json.Unmarshal([]byte(response.Content[0].TextContent.Text), &result); err != nil {
		t.Fatalf("failed to decode tool response JSON: %v", err)
	}
	return result
}

func TestListProjectsErrorResponseNormalizesNotFoundToEmptyList(t *testing.T) {
	response := listProjectsErrorResponse(
		fakeHTTPErrorResponse(http.StatusNotFound, `{"detail":"There is no available data can be found"}`),
		fmt.Errorf("not found"),
	)

	result := decodeToolResponseJSON[ProjectListResponse](t, response)
	if result.Total != 0 {
		t.Fatalf("expected zero projects, got %+v", result)
	}
	if len(result.Projects) != 0 {
		t.Fatalf("expected empty projects list, got %+v", result)
	}
	if result.Message != "No projects found" {
		t.Fatalf("expected normalized empty-list message, got %+v", result)
	}
}

func TestWaitForProjectLookupErrorResponseTreatsNotFoundAsDeleted(t *testing.T) {
	response := waitForProjectLookupErrorResponse(
		true,
		42,
		fakeHTTPErrorResponse(http.StatusNotFound, `{"detail":"Project not found"}`),
		fmt.Errorf("not found"),
	)

	result := decodeToolResponseJSON[SuccessResponse](t, response)
	if !result.Success {
		t.Fatalf("expected deletion wait to succeed, got %+v", result)
	}
	if !strings.Contains(result.Message, "42") || !strings.Contains(strings.ToLower(result.Message), "deleted") {
		t.Fatalf("expected deletion success message, got %+v", result)
	}
}

func TestWaitForProjectLookupErrorResponsePreservesErrorsWhenNotWaitingForDelete(t *testing.T) {
	response := waitForProjectLookupErrorResponse(
		false,
		42,
		fakeHTTPErrorResponse(http.StatusNotFound, `{"detail":"Project not found"}`),
		fmt.Errorf("not found"),
	)

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "Project not found") {
		t.Fatalf("expected original API error to be preserved, got %+v", result)
	}
}

func TestDeleteProjectErrorResponseTranslatesNonEmptyProject(t *testing.T) {
	response := deleteProjectErrorResponse(
		946,
		fakeHTTPErrorResponse(http.StatusBadRequest, `{"detail":"You can not delete non empty project"}`),
		fmt.Errorf("bad request"),
	)

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "Project 946 cannot be deleted until all servers are removed") {
		t.Fatalf("expected clearer non-empty project validation error, got %+v", result)
	}
	if !strings.Contains(result.Details, "You can not delete non empty project") {
		t.Fatalf("expected original API error in details, got %+v", result)
	}
}

func TestDeleteProjectErrorResponsePreservesOtherErrors(t *testing.T) {
	response := deleteProjectErrorResponse(
		946,
		fakeHTTPErrorResponse(http.StatusBadRequest, `{"detail":"something else went wrong"}`),
		fmt.Errorf("bad request"),
	)

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "something else went wrong") {
		t.Fatalf("expected unrelated delete errors to pass through, got %+v", result)
	}
	if result.Details != "" {
		t.Fatalf("expected passthrough delete errors to avoid extra details wrapping, got %+v", result)
	}
}
