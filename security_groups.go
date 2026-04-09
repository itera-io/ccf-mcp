package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

func createStandaloneProfileSecurityGroup(client *taikungoclient.Client, args JSONPayloadArgs) (*mcp_golang.ToolResponse, error) {
	command, errorResp := decodePayload[taikuncore.CreateSecurityGroupCommand](args.Payload)
	if errorResp != nil {
		return errorResp, nil
	}

	apiResp, httpResponse, err := client.Client.SecurityGroupAPI.SecuritygroupCreate(context.Background()).
		CreateSecurityGroupCommand(*command).
		Execute()
	return finalizeAPIOperation(apiResp, httpResponse, err, "create standalone profile security group", "Standalone profile security group created successfully")
}

func updateStandaloneProfileSecurityGroup(client *taikungoclient.Client, args JSONPayloadArgs) (*mcp_golang.ToolResponse, error) {
	command, errorResp := decodePayload[taikuncore.EditSecurityGroupCommand](args.Payload)
	if errorResp != nil {
		return errorResp, nil
	}

	httpResponse, err := client.Client.SecurityGroupAPI.SecuritygroupEdit(context.Background()).
		EditSecurityGroupCommand(*command).
		Execute()
	return finalizeAction(httpResponse, err, "update standalone profile security group", fmt.Sprintf("Standalone profile security group %d updated successfully", command.GetId()))
}

func deleteStandaloneProfileSecurityGroup(client *taikungoclient.Client, args IDArgs) (*mcp_golang.ToolResponse, error) {
	httpResponse, err := client.Client.SecurityGroupAPI.SecuritygroupDelete(context.Background(), args.ID).Execute()
	return finalizeAction(httpResponse, err, "delete standalone profile security group", fmt.Sprintf("Standalone profile security group %d deleted successfully", args.ID))
}
