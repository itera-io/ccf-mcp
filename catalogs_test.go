package main

import (
	"strings"
	"testing"
)

func TestResolveCatalogAppForRemovalMatchesUniqueApp(t *testing.T) {
	app, err := resolveCatalogAppForRemoval([]CatalogAppSummary{
		{ID: 11, Name: "nginx", Repository: "bitnami", CatalogID: 101},
		{ID: 12, Name: "redis", Repository: "bitnami", CatalogID: 101},
	}, RemoveAppFromCatalogArgs{
		CatalogID:   101,
		PackageName: "nginx",
	})
	if err != nil {
		t.Fatalf("expected unique app match, got error: %v", err)
	}
	if app.ID != 11 {
		t.Fatalf("expected app ID 11, got %d", app.ID)
	}
}

func TestResolveCatalogAppForRemovalUsesRepositoryToDisambiguate(t *testing.T) {
	app, err := resolveCatalogAppForRemoval([]CatalogAppSummary{
		{ID: 21, Name: "nginx", Repository: "bitnami", CatalogID: 202},
		{ID: 22, Name: "nginx", Repository: "other", CatalogID: 202},
	}, RemoveAppFromCatalogArgs{
		CatalogID:   202,
		PackageName: "nginx",
		Repository:  "other",
	})
	if err != nil {
		t.Fatalf("expected repository to disambiguate, got error: %v", err)
	}
	if app.ID != 22 {
		t.Fatalf("expected app ID 22, got %d", app.ID)
	}
}

func TestResolveCatalogAppForRemovalRejectsAmbiguousMatch(t *testing.T) {
	_, err := resolveCatalogAppForRemoval([]CatalogAppSummary{
		{ID: 31, Name: "nginx", Repository: "bitnami", CatalogID: 303},
		{ID: 32, Name: "nginx", Repository: "other", CatalogID: 303},
	}, RemoveAppFromCatalogArgs{
		CatalogID:   303,
		PackageName: "nginx",
	})
	if err == nil {
		t.Fatal("expected ambiguous match error, got nil")
	}
	if !strings.Contains(err.Error(), "specify repository") {
		t.Fatalf("expected disambiguation guidance, got %v", err)
	}
}

func TestResolveCatalogAppForRemovalRejectsMissingApp(t *testing.T) {
	_, err := resolveCatalogAppForRemoval([]CatalogAppSummary{
		{ID: 41, Name: "redis", Repository: "bitnami", CatalogID: 404},
	}, RemoveAppFromCatalogArgs{
		CatalogID:   404,
		PackageName: "nginx",
	})
	if err == nil {
		t.Fatal("expected missing app error, got nil")
	}
	if !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}
