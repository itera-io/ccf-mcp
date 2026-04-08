package main

import (
	"strings"
	"testing"
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
