package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

func enableAutoscaling(client *taikungoclient.Client, args JSONPayloadArgs) (*mcp_golang.ToolResponse, error) {
	command, errorResp := decodePayload[taikuncore.EnableAutoscalingCommand](args.Payload)
	if errorResp != nil {
		return errorResp, nil
	}

	httpResponse, err := client.Client.AutoscalingAPI.AutoscalingEnable(context.Background()).
		EnableAutoscalingCommand(*command).
		Execute()
	return finalizeAction(httpResponse, err, "enable autoscaling", "Autoscaling enabled successfully")
}

func updateAutoscaling(client *taikungoclient.Client, args JSONPayloadArgs) (*mcp_golang.ToolResponse, error) {
	command, errorResp := decodePayload[taikuncore.EditAutoscalingCommand](args.Payload)
	if errorResp != nil {
		return errorResp, nil
	}

	httpResponse, err := client.Client.AutoscalingAPI.AutoscalingEdit(context.Background()).
		EditAutoscalingCommand(*command).
		Execute()
	return finalizeAction(httpResponse, err, "update autoscaling", "Autoscaling updated successfully")
}

func disableAutoscaling(client *taikungoclient.Client, args JSONPayloadArgs) (*mcp_golang.ToolResponse, error) {
	command, errorResp := decodePayload[taikuncore.DisableAutoscalingCommand](args.Payload)
	if errorResp != nil {
		return errorResp, nil
	}

	httpResponse, err := client.Client.AutoscalingAPI.AutoscalingDisable(context.Background()).
		DisableAutoscalingCommand(*command).
		Execute()
	return finalizeAction(httpResponse, err, "disable autoscaling", "Autoscaling disabled successfully")
}

func getAutoscalingStatus(client *taikungoclient.Client, args ProjectIDArgs) (*mcp_golang.ToolResponse, error) {
	result, httpResponse, err := client.Client.ProjectsAPI.ProjectsList(context.Background()).
		Id(args.ProjectID).
		Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}
	if errorResp := checkResponse(httpResponse, "get autoscaling status"); errorResp != nil {
		return errorResp, nil
	}
	if result == nil || len(result.Data) == 0 {
		return createJSONResponse(ErrorResponse{
			Error: fmt.Sprintf("project %d not found", args.ProjectID),
		}), nil
	}

	project := result.Data[0]
	return createJSONResponse(map[string]interface{}{
		"projectId":                project.GetId(),
		"projectName":              project.GetName(),
		"isAutoscalingEnabled":     project.GetIsAutoscalingEnabled(),
		"isAutoscalingSpotEnabled": project.GetIsAutoscalingSpotEnabled(),
		"status":                   project.GetStatus(),
		"health":                   project.GetHealth(),
		"message":                  fmt.Sprintf("Loaded autoscaling status for project %d", args.ProjectID),
		"success":                  true,
	}), nil
}
