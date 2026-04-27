package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/itera-io/taikungoclient"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

const (
	mcpLockOrgIDsEnv     = "MCP_LOCK_ORGANIZATION_IDS"
	mcpLockProjectIDsEnv = "MCP_LOCK_PROJECT_IDS"
	mcpLockOrgIDsArg     = "--mcp-lock-organization-ids"
	mcpLockProjectIDsArg = "--mcp-lock-project-ids"
)

type MCPLockArgs struct {
	OrganizationIDs []int32 `json:"organizationIds,omitempty" jsonschema:"description=Allowed organization IDs for this MCP session"`
	ProjectIDs      []int32 `json:"projectIds,omitempty" jsonschema:"description=Allowed project IDs for this MCP session"`
}

type MCPLockStatusArgs struct{}

type MCPLockClearArgs struct{}

type mcpLockScope struct {
	OrganizationIDs []int32 `json:"organizationIds,omitempty"`
	ProjectIDs      []int32 `json:"projectIds,omitempty"`
	Source          string  `json:"source"`
	UpdatedAt       string  `json:"updatedAt"`
}

type mcpLockState struct {
	mu           sync.RWMutex
	envScope     *mcpLockScope
	runtimeScope *mcpLockScope
}

type mcpLockTargets struct {
	OrganizationIDs []int32
	ProjectIDs      []int32
}

var globalMCPLockState mcpLockState
var resolveMCPLockOrgsForProjectsFn = resolveMCPLockOrganizationIDsFromProjects

func initMCPLockFromEnv(getenv func(string) string) error {
	return initMCPLockFromConfig(getenv, nil)
}

func initMCPLockFromConfig(getenv func(string) string, startupArgs []string) error {
	envOrgIDs, err := parseMCPLockIDList(getenv(mcpLockOrgIDsEnv))
	if err != nil {
		return fmt.Errorf("invalid %s: %w", mcpLockOrgIDsEnv, err)
	}
	envProjectIDs, err := parseMCPLockIDList(getenv(mcpLockProjectIDsEnv))
	if err != nil {
		return fmt.Errorf("invalid %s: %w", mcpLockProjectIDsEnv, err)
	}

	argOrgIDsRaw, argProjectIDsRaw, err := parseMCPLockIDsFromArgs(startupArgs)
	if err != nil {
		return err
	}
	argOrgIDs, err := parseMCPLockIDList(argOrgIDsRaw)
	if err != nil {
		return fmt.Errorf("invalid %s value %q: %w", mcpLockOrgIDsArg, argOrgIDsRaw, err)
	}
	argProjectIDs, err := parseMCPLockIDList(argProjectIDsRaw)
	if err != nil {
		return fmt.Errorf("invalid %s value %q: %w", mcpLockProjectIDsArg, argProjectIDsRaw, err)
	}

	orgIDs := envOrgIDs
	if argOrgIDsRaw != "" {
		orgIDs = argOrgIDs
	}
	projectIDs := envProjectIDs
	if argProjectIDsRaw != "" {
		projectIDs = argProjectIDs
	}

	if len(orgIDs) == 0 && len(projectIDs) == 0 {
		globalMCPLockState.mu.Lock()
		globalMCPLockState.envScope = nil
		globalMCPLockState.mu.Unlock()
		return nil
	}

	scope := newMCPLockScope(orgIDs, projectIDs, "env")
	globalMCPLockState.mu.Lock()
	globalMCPLockState.envScope = &scope
	globalMCPLockState.mu.Unlock()
	if argOrgIDsRaw != "" || argProjectIDsRaw != "" {
		logger.Printf("Initialized MCP lock from startup arguments (%d org IDs, %d project IDs)", len(scope.OrganizationIDs), len(scope.ProjectIDs))
	} else {
		logger.Printf("Initialized MCP lock from environment (%d org IDs, %d project IDs)", len(scope.OrganizationIDs), len(scope.ProjectIDs))
	}
	return nil
}

func mcpLock(args MCPLockArgs) (*mcp_golang.ToolResponse, error) {
	if len(args.OrganizationIDs) == 0 && len(args.ProjectIDs) == 0 {
		return createJSONResponse(ErrorResponse{
			Error:   "mcp-lock requires at least one scope constraint",
			Details: "Provide organizationIds and/or projectIds. Use mcp-lock-clear to remove runtime lock.",
		}), nil
	}

	scope := newMCPLockScope(args.OrganizationIDs, args.ProjectIDs, "runtime")
	globalMCPLockState.mu.Lock()
	globalMCPLockState.runtimeScope = &scope
	globalMCPLockState.mu.Unlock()

	return createJSONResponse(map[string]interface{}{
		"success": true,
		"message": "MCP runtime lock updated successfully",
		"lock":    scope,
	}), nil
}

func mcpLockStatus(args MCPLockStatusArgs) (*mcp_golang.ToolResponse, error) {
	snapshot := getMCPLockSnapshot()
	return createJSONResponse(map[string]interface{}{
		"success":     true,
		"message":     "MCP lock status loaded",
		"active":      snapshot.Active,
		"envLock":     snapshot.EnvScope,
		"runtimeLock": snapshot.RuntimeScope,
		"effective":   snapshot.Effective,
	}), nil
}

func mcpLockClear(args MCPLockClearArgs) (*mcp_golang.ToolResponse, error) {
	globalMCPLockState.mu.Lock()
	globalMCPLockState.runtimeScope = nil
	globalMCPLockState.mu.Unlock()

	effective, active := getEffectiveMCPLockScope()
	message := "Runtime MCP lock cleared; no active lock remains"
	if active {
		message = "Runtime MCP lock cleared; environment lock remains active"
	}

	return createJSONResponse(map[string]interface{}{
		"success":   true,
		"message":   message,
		"active":    active,
		"effective": effective,
	}), nil
}

func enforceMCPLock(toolName string, args interface{}) *mcp_golang.ToolResponse {
	if strings.HasPrefix(toolName, "mcp-lock") {
		return nil
	}

	scope, active := getEffectiveMCPLockScope()
	if !active {
		return nil
	}

	targets, hasTargets, err := extractMCPLockTargets(args)
	if err != nil {
		return createJSONResponse(ErrorResponse{
			Error:   fmt.Sprintf("Unable to evaluate MCP lock for tool %q", toolName),
			Details: err.Error(),
		})
	}
	if !hasTargets {
		return nil
	}

	if len(scope.OrganizationIDs) > 0 && len(targets.OrganizationIDs) == 0 && len(targets.ProjectIDs) > 0 {
		resolvedOrgIDs, resolveErr := resolveMCPLockOrgsForProjectsFn(targets.ProjectIDs)
		if resolveErr != nil {
			return createJSONResponse(ErrorResponse{
				Error:   fmt.Sprintf("MCP lock blocked tool %q", toolName),
				Details: fmt.Sprintf("unable to resolve organization scope for project IDs %v: %v", targets.ProjectIDs, resolveErr),
			})
		}
		targets.OrganizationIDs = normalizeMCPLockIDs(append(targets.OrganizationIDs, resolvedOrgIDs...))
	}

	var outOfScopeProjects []int32
	if len(scope.ProjectIDs) > 0 && len(targets.ProjectIDs) > 0 {
		allowedProjects := toMCPLockSet(scope.ProjectIDs)
		for _, projectID := range targets.ProjectIDs {
			if _, ok := allowedProjects[projectID]; !ok {
				outOfScopeProjects = append(outOfScopeProjects, projectID)
			}
		}
	}

	var outOfScopeOrgs []int32
	if len(scope.OrganizationIDs) > 0 && len(targets.OrganizationIDs) > 0 {
		allowedOrgs := toMCPLockSet(scope.OrganizationIDs)
		for _, orgID := range targets.OrganizationIDs {
			if _, ok := allowedOrgs[orgID]; !ok {
				outOfScopeOrgs = append(outOfScopeOrgs, orgID)
			}
		}
	}

	if len(outOfScopeProjects) == 0 && len(outOfScopeOrgs) == 0 {
		return nil
	}

	detailsParts := []string{}
	if len(outOfScopeOrgs) > 0 {
		detailsParts = append(detailsParts, fmt.Sprintf("organization IDs out of scope: %v", outOfScopeOrgs))
	}
	if len(outOfScopeProjects) > 0 {
		detailsParts = append(detailsParts, fmt.Sprintf("project IDs out of scope: %v", outOfScopeProjects))
	}
	detailsParts = append(detailsParts, fmt.Sprintf("active lock: org=%v project=%v", scope.OrganizationIDs, scope.ProjectIDs))

	return createJSONResponse(ErrorResponse{
		Error:   fmt.Sprintf("MCP lock blocked tool %q", toolName),
		Details: strings.Join(detailsParts, "; "),
	})
}

func extractMCPLockTargets(args interface{}) (mcpLockTargets, bool, error) {
	raw, err := json.Marshal(args)
	if err != nil {
		return mcpLockTargets{}, false, fmt.Errorf("marshal args: %w", err)
	}

	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return mcpLockTargets{}, false, fmt.Errorf("decode args json: %w", err)
	}

	targets := mcpLockTargets{}
	walkMCPLockTargets(decoded, &targets)

	targets.OrganizationIDs = normalizeMCPLockIDs(targets.OrganizationIDs)
	targets.ProjectIDs = normalizeMCPLockIDs(targets.ProjectIDs)
	return targets, len(targets.OrganizationIDs) > 0 || len(targets.ProjectIDs) > 0, nil
}

func walkMCPLockTargets(value interface{}, targets *mcpLockTargets) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, nested := range typed {
			switch normalizeMCPLockKey(key) {
			case "organizationid":
				if id, ok := toMCPLockInt32(nested); ok && id > 0 {
					targets.OrganizationIDs = append(targets.OrganizationIDs, id)
				}
			case "organizationids":
				targets.OrganizationIDs = append(targets.OrganizationIDs, toMCPLockInt32Slice(nested)...)
			case "projectid":
				if id, ok := toMCPLockInt32(nested); ok && id > 0 {
					targets.ProjectIDs = append(targets.ProjectIDs, id)
				}
			case "projectids":
				targets.ProjectIDs = append(targets.ProjectIDs, toMCPLockInt32Slice(nested)...)
			case "payload":
				payloadString, ok := nested.(string)
				if ok && strings.TrimSpace(payloadString) != "" {
					var payload interface{}
					if err := json.Unmarshal([]byte(payloadString), &payload); err == nil {
						walkMCPLockTargets(payload, targets)
					}
				}
			default:
				walkMCPLockTargets(nested, targets)
			}
		}
	case []interface{}:
		for _, item := range typed {
			walkMCPLockTargets(item, targets)
		}
	}
}

func parseMCPLockIDList(raw string) ([]int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parts := strings.Split(trimmed, ",")
	ids := make([]int32, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		parsed, err := strconv.Atoi(item)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", item)
		}
		if parsed <= 0 {
			return nil, fmt.Errorf("ID must be positive: %d", parsed)
		}
		ids = append(ids, int32(parsed))
	}
	return normalizeMCPLockIDs(ids), nil
}

func parseMCPLockIDsFromArgs(startupArgs []string) (orgIDs string, projectIDs string, err error) {
	if len(startupArgs) == 0 {
		return "", "", nil
	}

	for i := 0; i < len(startupArgs); i++ {
		current := strings.TrimSpace(startupArgs[i])
		if current == "" {
			continue
		}

		switch {
		case current == mcpLockOrgIDsArg:
			if i+1 >= len(startupArgs) {
				return "", "", fmt.Errorf("missing value for %s", mcpLockOrgIDsArg)
			}
			i++
			orgIDs = strings.TrimSpace(startupArgs[i])
		case strings.HasPrefix(current, mcpLockOrgIDsArg+"="):
			orgIDs = strings.TrimSpace(strings.TrimPrefix(current, mcpLockOrgIDsArg+"="))
		case current == mcpLockProjectIDsArg:
			if i+1 >= len(startupArgs) {
				return "", "", fmt.Errorf("missing value for %s", mcpLockProjectIDsArg)
			}
			i++
			projectIDs = strings.TrimSpace(startupArgs[i])
		case strings.HasPrefix(current, mcpLockProjectIDsArg+"="):
			projectIDs = strings.TrimSpace(strings.TrimPrefix(current, mcpLockProjectIDsArg+"="))
		}
	}

	return orgIDs, projectIDs, nil
}

func newMCPLockScope(orgIDs []int32, projectIDs []int32, source string) mcpLockScope {
	return mcpLockScope{
		OrganizationIDs: normalizeMCPLockIDs(orgIDs),
		ProjectIDs:      normalizeMCPLockIDs(projectIDs),
		Source:          source,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func getEffectiveMCPLockScope() (mcpLockScope, bool) {
	globalMCPLockState.mu.RLock()
	defer globalMCPLockState.mu.RUnlock()

	if globalMCPLockState.runtimeScope != nil {
		return *globalMCPLockState.runtimeScope, true
	}
	if globalMCPLockState.envScope != nil {
		return *globalMCPLockState.envScope, true
	}
	return mcpLockScope{}, false
}

type mcpLockSnapshot struct {
	EnvScope    *mcpLockScope `json:"envLock,omitempty"`
	RuntimeScope *mcpLockScope `json:"runtimeLock,omitempty"`
	Effective   mcpLockScope  `json:"effective"`
	Active      bool          `json:"active"`
}

func getMCPLockSnapshot() mcpLockSnapshot {
	globalMCPLockState.mu.RLock()
	defer globalMCPLockState.mu.RUnlock()

	snapshot := mcpLockSnapshot{
		EnvScope:     cloneMCPLockScope(globalMCPLockState.envScope),
		RuntimeScope: cloneMCPLockScope(globalMCPLockState.runtimeScope),
	}

	if globalMCPLockState.runtimeScope != nil {
		snapshot.Effective = *cloneMCPLockScope(globalMCPLockState.runtimeScope)
		snapshot.Active = true
		return snapshot
	}
	if globalMCPLockState.envScope != nil {
		snapshot.Effective = *cloneMCPLockScope(globalMCPLockState.envScope)
		snapshot.Active = true
		return snapshot
	}

	snapshot.Effective = mcpLockScope{}
	snapshot.Active = false
	return snapshot
}

func resolveMCPLockOrganizationIDsFromProjects(projectIDs []int32) ([]int32, error) {
	if taikunClient == nil || taikunClient.Client == nil {
		return nil, fmt.Errorf("Cloudera Cloud Factory client is not initialized")
	}

	resolved := make([]int32, 0, len(projectIDs))
	for _, projectID := range normalizeMCPLockIDs(projectIDs) {
		projectList, httpResponse, err := taikunClient.Client.ProjectsAPI.ProjectsList(context.Background()).
			Id(projectID).
			Execute()
		if err != nil {
			return nil, taikungoclient.CreateError(httpResponse, err)
		}
		if httpResponse == nil {
			return nil, fmt.Errorf("no response received while resolving project %d", projectID)
		}
		if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("unexpected status %d while resolving project %d", httpResponse.StatusCode, projectID)
		}
		if projectList == nil || len(projectList.Data) == 0 {
			return nil, fmt.Errorf("project %d not found", projectID)
		}

		organizationID := projectList.Data[0].GetOrganizationId()
		if organizationID <= 0 {
			return nil, fmt.Errorf("project %d has no organization ID", projectID)
		}
		resolved = append(resolved, organizationID)
	}

	return normalizeMCPLockIDs(resolved), nil
}

func cloneMCPLockScope(scope *mcpLockScope) *mcpLockScope {
	if scope == nil {
		return nil
	}

	cloned := *scope
	cloned.OrganizationIDs = append([]int32(nil), scope.OrganizationIDs...)
	cloned.ProjectIDs = append([]int32(nil), scope.ProjectIDs...)
	return &cloned
}

func normalizeMCPLockIDs(ids []int32) []int32 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int32]struct{}, len(ids))
	normalized := make([]int32, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized
}

func normalizeMCPLockKey(key string) string {
	var builder strings.Builder
	builder.Grow(len(key))
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return strings.ToLower(builder.String())
}

func toMCPLockInt32(value interface{}) (int32, bool) {
	switch typed := value.(type) {
	case float64:
		return int32(typed), true
	case int:
		return int32(typed), true
	case int32:
		return typed, true
	case int64:
		return int32(typed), true
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	default:
		return 0, false
	}
}

func toMCPLockInt32Slice(value interface{}) []int32 {
	switch typed := value.(type) {
	case []interface{}:
		ids := make([]int32, 0, len(typed))
		for _, item := range typed {
			if id, ok := toMCPLockInt32(item); ok && id > 0 {
				ids = append(ids, id)
			}
		}
		return ids
	case []int32:
		return typed
	case []int:
		ids := make([]int32, 0, len(typed))
		for _, item := range typed {
			if item > 0 {
				ids = append(ids, int32(item))
			}
		}
		return ids
	default:
		if id, ok := toMCPLockInt32(value); ok && id > 0 {
			return []int32{id}
		}
	}
	return nil
}

func toMCPLockSet(ids []int32) map[int32]struct{} {
	result := make(map[int32]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result
}
