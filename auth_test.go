package main

import (
	"strings"
	"testing"

	taikuncore "github.com/itera-io/taikungoclient/client"
)

func mapGetenv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestResolveRobotUserAuthConfigDefaultsHost(t *testing.T) {
	cfg, err := resolveRobotUserAuthConfig(mapGetenv(map[string]string{
		"TAIKUN_ACCESS_KEY": "robot-access",
		"TAIKUN_SECRET_KEY": "robot-secret",
	}))
	if err != nil {
		t.Fatalf("expected valid robot user config, got error: %v", err)
	}

	if cfg.APIHost != defaultAPIHost {
		t.Fatalf("expected default API host %q, got %q", defaultAPIHost, cfg.APIHost)
	}
	if cfg.AccessKey != "robot-access" || cfg.SecretKey != "robot-secret" {
		t.Fatalf("unexpected robot user credentials in config: %+v", cfg)
	}
}

func TestResolveRobotUserAuthConfigSupportsOptionalDomainName(t *testing.T) {
	cfg, err := resolveRobotUserAuthConfig(mapGetenv(map[string]string{
		"TAIKUN_ACCESS_KEY":  "robot-access",
		"TAIKUN_SECRET_KEY":  "robot-secret",
		"TAIKUN_DOMAIN_NAME": "example-domain",
		"TAIKUN_API_HOST":    "api.example.test",
	}))
	if err != nil {
		t.Fatalf("expected valid robot user config, got error: %v", err)
	}

	if cfg.DomainName != "example-domain" {
		t.Fatalf("expected domain name to be preserved, got %q", cfg.DomainName)
	}
	if cfg.APIHost != "api.example.test" {
		t.Fatalf("expected API host override to be preserved, got %q", cfg.APIHost)
	}
}

func TestResolveRobotUserAuthConfigRejectsIncompleteCredentials(t *testing.T) {
	_, err := resolveRobotUserAuthConfig(mapGetenv(map[string]string{
		"TAIKUN_ACCESS_KEY": "robot-access",
	}))
	if err == nil {
		t.Fatal("expected incomplete robot user credentials to fail")
	}
	if !strings.Contains(err.Error(), "TAIKUN_ACCESS_KEY") || !strings.Contains(err.Error(), "TAIKUN_SECRET_KEY") {
		t.Fatalf("expected error to mention both robot user env vars, got: %v", err)
	}
}

func TestResolveRobotUserAuthConfigRejectsLegacyEmailPassword(t *testing.T) {
	_, err := resolveRobotUserAuthConfig(mapGetenv(map[string]string{
		"TAIKUN_EMAIL":    "user@example.com",
		"TAIKUN_PASSWORD": "super-secret",
	}))
	if err == nil {
		t.Fatal("expected legacy email/password auth to be rejected")
	}
	if !strings.Contains(err.Error(), "no longer supported") {
		t.Fatalf("expected legacy auth rejection message, got: %v", err)
	}
}

func TestEvaluateToolScopeAccessAllowed(t *testing.T) {
	access := evaluateToolScopeAccess("create-project", []string{"scope:projects:write"})
	if access.Status != "allowed" {
		t.Fatalf("expected allowed status, got %+v", access)
	}
}

func TestEvaluateToolScopeAccessBlocked(t *testing.T) {
	access := evaluateToolScopeAccess("create-project", []string{"scope:projects:read"})
	if access.Status != "blocked" {
		t.Fatalf("expected blocked status, got %+v", access)
	}
	if len(access.MissingScopes) != 1 || access.MissingScopes[0] != "scope:projects:write" {
		t.Fatalf("unexpected missing scopes: %+v", access.MissingScopes)
	}
}

func TestEvaluateToolScopeAccessUnknown(t *testing.T) {
	access := evaluateToolScopeAccess("unmapped-tool", nil)
	if access.Status != "unknown" {
		t.Fatalf("expected unknown status, got %+v", access)
	}
}

func TestEvaluateToolScopeAccessAllowsNoScopeTools(t *testing.T) {
	access := evaluateToolScopeAccess("robot-user-capabilities", nil)
	if access.Status != "allowed" {
		t.Fatalf("expected allowed status, got %+v", access)
	}
	if len(access.RequiredScopes) != 0 {
		t.Fatalf("expected no required scopes, got %+v", access.RequiredScopes)
	}
}

func TestRobotUserContextFromDetailsPopulatesAccountFields(t *testing.T) {
	details := taikuncore.NewRobotUsersListDto(
		"user-1",
		7,
		"domain-fallback",
		"access-key",
		"creator",
		"robot-name",
		[]string{"scope:projects:read"},
		true,
		"2026-04-08T00:00:00Z",
	)
	details.AdditionalProperties = map[string]interface{}{
		"accountId":   float64(42),
		"accountName": "ccf-account",
	}

	ctx := robotUserContextFromDetails(details)
	if ctx.AccountID != 42 {
		t.Fatalf("expected account id 42, got %+v", ctx)
	}
	if ctx.AccountName != "ccf-account" {
		t.Fatalf("expected account name to come from accountName, got %+v", ctx)
	}
}

func TestAuthorizeToolDeniesScopedToolWhenScopeDiscoveryFails(t *testing.T) {
	setRobotUserContext(RobotUserContext{ScopeDiscoveryError: "boom"})
	t.Cleanup(func() {
		setRobotUserContext(RobotUserContext{})
	})

	denied := authorizeTool("create-project")
	if denied == nil {
		t.Fatal("expected scoped tool to be denied when scope discovery fails")
	}
}

func TestAuthorizeToolAllowsNoScopeToolWhenScopeDiscoveryFails(t *testing.T) {
	setRobotUserContext(RobotUserContext{ScopeDiscoveryError: "boom"})
	t.Cleanup(func() {
		setRobotUserContext(RobotUserContext{})
	})

	denied := authorizeTool("robot-user-capabilities")
	if denied != nil {
		t.Fatal("expected no-scope tool to remain allowed when scope discovery fails")
	}
}

func TestNewRefreshTaikunClientResponseSuccessWhenScopeDiscoverySucceeds(t *testing.T) {
	resp := newRefreshTaikunClientResponse(RobotUserContext{
		Name:             "robot",
		OrganizationName: "org",
		Scopes:           []string{"scope:projects:read"},
	})

	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
}

func TestNewRefreshTaikunClientResponseFailsWhenScopeDiscoveryFails(t *testing.T) {
	resp := newRefreshTaikunClientResponse(RobotUserContext{
		Name:                "robot",
		ScopeDiscoveryError: "unable to load scopes",
	})

	if resp.Success {
		t.Fatalf("expected failed response when scope discovery fails, got %+v", resp)
	}
	if resp.ScopeDiscoveryError == "" {
		t.Fatalf("expected scope discovery error to be preserved, got %+v", resp)
	}
}
