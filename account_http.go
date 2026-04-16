package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type createAccountCommand struct {
	Name               string `json:"name,omitempty"`
	Email              string `json:"email,omitempty"`
	CreateOrganization *bool  `json:"createOrganization,omitempty"`
}

type updateAccountCommand struct {
	ID    int32  `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type accountListItem struct {
	ID                 int32  `json:"id"`
	Name               string `json:"name,omitempty"`
	OrganizationsCount int64  `json:"organizationsCount,omitempty"`
	UsersCount         int64  `json:"usersCount,omitempty"`
	GroupsCount        int64  `json:"groupsCount,omitempty"`
	ProjectsCount      int64  `json:"projectsCount,omitempty"`
}

type accountListCursorPaginatedResponse struct {
	Data       []accountListItem `json:"data"`
	Limit      int32             `json:"limit"`
	HasMore    bool              `json:"hasMore"`
	TotalCount int64             `json:"totalCount"`
	NextCursor *int32            `json:"nextCursor,omitempty"`
}

type accountDetailsDTO struct {
	ID                  int32                              `json:"id"`
	Name                string                             `json:"name,omitempty"`
	Email               string                             `json:"email,omitempty"`
	OrganizationsCount  int64                              `json:"organizationsCount,omitempty"`
	UsersCount          int64                              `json:"usersCount,omitempty"`
	GroupsCount         int64                              `json:"groupsCount,omitempty"`
	ProjectsCount       int64                              `json:"projectsCount,omitempty"`
	CreatedAt           string                             `json:"createdAt,omitempty"`
	Is2FAEnabled        bool                               `json:"is2FAEnabled"`
	UserWithGlobalRoles []taikuncore.UserWithGlobalRoleDto `json:"userWithGlobalRoles,omitempty"`
}

func performAuthenticatedRequest(client *taikungoclient.Client, method, apiPath string, query url.Values, body any) (*http.Response, error) {
	cfg := client.Client.GetConfig()
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}

	endpoint := (&url.URL{
		Scheme:   scheme,
		Host:     cfg.Host,
		Path:     apiPath,
		RawQuery: query.Encode(),
	}).String()

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return cfg.HTTPClient.Do(req)
}

func performAuthenticatedJSONRequest[T any](client *taikungoclient.Client, method, apiPath string, query url.Values, body any) (*T, *http.Response, error) {
	httpResponse, err := performAuthenticatedRequest(client, method, apiPath, query, body)
	if err != nil {
		return nil, httpResponse, err
	}
	if httpResponse == nil || httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return nil, httpResponse, nil
	}

	defer func() {
		if err := httpResponse.Body.Close(); err != nil {
			logger.Printf("Failed to close HTTP response body: %v", err)
		}
	}()

	var result T
	if err := json.NewDecoder(httpResponse.Body).Decode(&result); err != nil {
		return nil, httpResponse, err
	}
	return &result, httpResponse, nil
}

func translateDomainIDPayload(payload string) (map[string]interface{}, *mcp_golang.ToolResponse) {
	command, errorResp := decodePayload[map[string]interface{}](payload)
	if errorResp != nil {
		return nil, errorResp
	}

	if command == nil {
		return map[string]interface{}{}, nil
	}

	if domainID, ok := (*command)["domainId"]; ok {
		if _, exists := (*command)["accountId"]; !exists {
			(*command)["accountId"] = domainID
		}
		delete(*command, "domainId")
	}

	return *command, nil
}
