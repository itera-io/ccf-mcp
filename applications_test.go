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
