package main

import "testing"

func TestResolveInstallAppTimeoutDefaultsToTenMinutes(t *testing.T) {
	timeout, defaulted := resolveInstallAppTimeout(0)
	if timeout != defaultInstallAppTimeoutSeconds {
		t.Fatalf("expected default install timeout %d, got %d", defaultInstallAppTimeoutSeconds, timeout)
	}
	if !defaulted {
		t.Fatalf("expected timeout to be marked defaulted")
	}
}

func TestResolveInstallAppTimeoutPreservesExplicitValue(t *testing.T) {
	timeout, defaulted := resolveInstallAppTimeout(900)
	if timeout != 900 {
		t.Fatalf("expected explicit install timeout 900, got %d", timeout)
	}
	if defaulted {
		t.Fatalf("expected explicit timeout to not be marked defaulted")
	}
}

func TestResolveInstallAppTTLDefaultsToTenMinutes(t *testing.T) {
	ttl, defaulted, validationError := resolveInstallAppTTL(0)
	if ttl != defaultInstallAppTTLMinutes {
		t.Fatalf("expected default install ttl %d, got %d", defaultInstallAppTTLMinutes, ttl)
	}
	if !defaulted {
		t.Fatalf("expected ttl to be marked defaulted")
	}
	if validationError != "" {
		t.Fatalf("expected no ttl validation error, got %q", validationError)
	}
}

func TestResolveInstallAppTTLPreservesExplicitValue(t *testing.T) {
	ttl, defaulted, validationError := resolveInstallAppTTL(30)
	if ttl != 30 {
		t.Fatalf("expected explicit install ttl 30, got %d", ttl)
	}
	if defaulted {
		t.Fatalf("expected explicit ttl to not be marked defaulted")
	}
	if validationError != "" {
		t.Fatalf("expected no ttl validation error, got %q", validationError)
	}
}

func TestResolveInstallAppTTLRejectsOutOfRangeValue(t *testing.T) {
	ttl, defaulted, validationError := resolveInstallAppTTL(5)
	if ttl != 0 {
		t.Fatalf("expected rejected ttl to return 0, got %d", ttl)
	}
	if defaulted {
		t.Fatalf("expected rejected ttl to not be marked defaulted")
	}
	if validationError == "" {
		t.Fatalf("expected ttl validation error for out of range value")
	}
}
