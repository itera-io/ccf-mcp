package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestApplyOffsetLimit(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	got := applyOffsetLimit(items, 1, 2)
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("expected [2 3], got %#v", got)
	}

	got = applyOffsetLimit(items, 3, 0)
	if len(got) != 2 || got[0] != 4 || got[1] != 5 {
		t.Fatalf("expected [4 5], got %#v", got)
	}

	got = applyOffsetLimit(items, 10, 5)
	if len(got) != 0 {
		t.Fatalf("expected empty slice for out-of-range offset, got %#v", got)
	}
}

func TestSortAndPaginateStrings(t *testing.T) {
	values := []string{"zebra", "alpha", "delta", "beta"}

	got := sortAndPaginateStrings(values, 1, 2)
	if len(got) != 2 || got[0] != "beta" || got[1] != "delta" {
		t.Fatalf("expected [beta delta], got %#v", got)
	}
}

func TestCheckResponseReturnsJSONError(t *testing.T) {
	response := checkResponse(fakeHTTPErrorResponse(http.StatusBadGateway, `{"detail":"upstream failed"}`), "list projects")

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "upstream failed") || !strings.Contains(result.Error, "HTTP 502") {
		t.Fatalf("expected detailed JSON API error, got %+v", result)
	}
}

func TestDeleteKubernetesResourceInvalidKindReturnsJSONError(t *testing.T) {
	response, err := deleteKubernetesResource(nil, DeleteKubernetesResourceArgs{
		ProjectID: 1,
		Kind:      "NotARealKind",
		Name:      "demo",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	result := decodeToolResponseJSON[ErrorResponse](t, response)
	if !strings.Contains(result.Error, "Invalid resource kind") {
		t.Fatalf("expected JSON invalid-kind error, got %+v", result)
	}
	if !strings.Contains(result.Details, "Pod") || !strings.Contains(result.Details, "Deployment") {
		t.Fatalf("expected allowed operation kinds in details, got %+v", result)
	}
}
