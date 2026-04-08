package main

import (
	"strings"
	"testing"

	taikuncore "github.com/itera-io/taikungoclient/client"
)

func TestNormalizeCreateKubernetesProfileCommandDefaultsLoadBalancers(t *testing.T) {
	command := taikuncore.NewCreateKubernetesProfileCommand()

	if errorResp := normalizeCreateKubernetesProfileCommand(command); errorResp != nil {
		t.Fatalf("expected defaulting to succeed, got response %+v", errorResp)
	}

	if !command.HasOctaviaEnabled() || !command.GetOctaviaEnabled() {
		t.Fatalf("expected octaviaEnabled to default to true, got %+v", command)
	}
	if !command.HasTaikunLBEnabled() || command.GetTaikunLBEnabled() {
		t.Fatalf("expected taikunLBEnabled to default to false, got %+v", command)
	}
}

func TestNormalizeCreateKubernetesProfileCommandRejectsConflictingLoadBalancers(t *testing.T) {
	command := taikuncore.NewCreateKubernetesProfileCommand()
	command.SetOctaviaEnabled(true)
	command.SetTaikunLBEnabled(true)

	response := normalizeCreateKubernetesProfileCommand(command)
	result := decodeToolResponseJSON[ErrorResponse](t, response)

	if !strings.Contains(result.Error, "cannot both be true") {
		t.Fatalf("expected clear validation error, got %+v", result)
	}
	if !strings.Contains(result.Details, "safe defaults") {
		t.Fatalf("expected remediation guidance, got %+v", result)
	}
}

func TestNormalizeCreateKubernetesProfileCommandPreservesExplicitSingleMode(t *testing.T) {
	command := taikuncore.NewCreateKubernetesProfileCommand()
	command.SetTaikunLBEnabled(true)

	if errorResp := normalizeCreateKubernetesProfileCommand(command); errorResp != nil {
		t.Fatalf("expected explicit single load balancer mode to pass, got %+v", errorResp)
	}

	if command.HasOctaviaEnabled() {
		t.Fatalf("expected octaviaEnabled to remain unset when user only set taikunLBEnabled, got %+v", command)
	}
	if !command.HasTaikunLBEnabled() || !command.GetTaikunLBEnabled() {
		t.Fatalf("expected explicit taikunLBEnabled=true to be preserved, got %+v", command)
	}
}
