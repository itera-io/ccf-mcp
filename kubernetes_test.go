package main

import (
	"strings"
	"testing"
)

func TestNormalizeKubeConfigRoleIDDefaultsAWSProjectsToRoleOne(t *testing.T) {
	roleID, errorResp := normalizeKubeConfigRoleID(953, "AWS", 0)
	if errorResp != nil {
		t.Fatalf("expected AWS kubeconfig role defaulting to succeed, got %+v", errorResp)
	}
	if roleID != awsKubeConfigRoleID {
		t.Fatalf("expected AWS projects to default to kubeConfigRoleId %d, got %d", awsKubeConfigRoleID, roleID)
	}
}

func TestNormalizeKubeConfigRoleIDRejectsNonDefaultAWSRole(t *testing.T) {
	_, response := normalizeKubeConfigRoleID(953, "AWS", 2)
	result := decodeToolResponseJSON[ErrorResponse](t, response)

	if !strings.Contains(result.Error, "only supports kubeConfigRoleId 1") {
		t.Fatalf("expected clear AWS kubeconfig role validation error, got %+v", result)
	}
	if !strings.Contains(result.Details, `cloudType "AWS"`) {
		t.Fatalf("expected AWS remediation guidance in details, got %+v", result)
	}
}

func TestNormalizeKubeConfigRoleIDPreservesNonAWSRoleSelection(t *testing.T) {
	roleID, errorResp := normalizeKubeConfigRoleID(77, "OpenStack", 7)
	if errorResp != nil {
		t.Fatalf("expected non-AWS kubeconfig role selection to pass through, got %+v", errorResp)
	}
	if roleID != 7 {
		t.Fatalf("expected non-AWS role selection to be preserved, got %d", roleID)
	}
}

func TestParseInt32RejectsOutOfRangeValues(t *testing.T) {
	if got := parseInt32("2147483647"); got != 2147483647 {
		t.Fatalf("expected max int32 to parse, got %d", got)
	}
	if got := parseInt32("2147483648"); got != 0 {
		t.Fatalf("expected out-of-range int32 parse to return 0, got %d", got)
	}
}

func TestParseReadyCountsRejectsOutOfRangeValues(t *testing.T) {
	ready, total := parseReadyCounts("12/2147483648")
	if ready != 0 || total != 0 {
		t.Fatalf("expected out-of-range ready counts to return 0/0, got %d/%d", ready, total)
	}
}
