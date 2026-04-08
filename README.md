# Cloudera Cloud Factory MCP Server

A Model Context Protocol (MCP) server that provides tools for managing Cloudera Cloud Factory resources (formerly Taikun), including projects, virtual clusters, catalogs, and applications.

Note: The repository name remains `cloudera-cloud-factory-mcp` for compatibility; the binary is now `cloudera-cloud-factory-mcp`.

[![Release](https://img.shields.io/github/v/release/itera-io/cloudera-cloud-factory-mcp)](https://github.com/itera-io/cloudera-cloud-factory-mcp/releases)
[![CI](https://github.com/itera-io/cloudera-cloud-factory-mcp/workflows/CI/badge.svg)](https://github.com/itera-io/cloudera-cloud-factory-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/itera-io/cloudera-cloud-factory-mcp)](https://goreportcard.com/report/github.com/itera-io/cloudera-cloud-factory-mcp)

## Installation

### Option 1: Download Pre-built Binaries (Recommended)

Download the latest release for your platform from the [releases page](https://github.com/itera-io/cloudera-cloud-factory-mcp/releases).

#### Linux (x86_64)
```bash
curl -L https://github.com/itera-io/cloudera-cloud-factory-mcp/releases/latest/download/cloudera-cloud-factory-mcp_Linux_x86_64.tar.gz | tar xz
sudo mv cloudera-cloud-factory-mcp /usr/local/bin/
```

#### macOS (Intel)
```bash
curl -L https://github.com/itera-io/cloudera-cloud-factory-mcp/releases/latest/download/cloudera-cloud-factory-mcp_Darwin_x86_64.tar.gz | tar xz
sudo mv cloudera-cloud-factory-mcp /usr/local/bin/
```

#### macOS (Apple Silicon)
```bash
curl -L https://github.com/itera-io/cloudera-cloud-factory-mcp/releases/latest/download/cloudera-cloud-factory-mcp_Darwin_arm64.tar.gz | tar xz
sudo mv cloudera-cloud-factory-mcp /usr/local/bin/
```

#### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri "https://github.com/itera-io/cloudera-cloud-factory-mcp/releases/latest/download/cloudera-cloud-factory-mcp_Windows_x86_64.zip" -OutFile "cloudera-cloud-factory-mcp.zip"
Expand-Archive -Path "cloudera-cloud-factory-mcp.zip" -DestinationPath .
# Move cloudera-cloud-factory-mcp.exe to your PATH
```

### Option 2: Build from Source

#### Prerequisites
- Go 1.24 or later
- Cloudera Cloud Factory account with API access

```bash
git clone https://github.com/itera-io/cloudera-cloud-factory-mcp
cd cloudera-cloud-factory-mcp
go build -o cloudera-cloud-factory-mcp
```

### Option 3: Using Go Install

```bash
go install github.com/itera-io/cloudera-cloud-factory-mcp@latest
```

## Configuration

The server authenticates to the Cloudera Cloud Factory API with **Robot User** credentials. Use the legacy `TAIKUN_*` environment variable names expected by the upstream Go client.

```bash
export TAIKUN_ACCESS_KEY="your-robot-user-access-key"
export TAIKUN_SECRET_KEY="your-robot-user-secret-key"
export TAIKUN_API_HOST="api-latest.osc1.sjc.cloudera.com"  # Optional, defaults to api-latest.osc1.sjc.cloudera.com
export TAIKUN_DOMAIN_NAME=""  # Optional, only set if your environment requires it
```

### Environment File

You can also create a `.env` file:

```bash
TAIKUN_ACCESS_KEY=your-robot-user-access-key
TAIKUN_SECRET_KEY=your-robot-user-secret-key
TAIKUN_API_HOST=api-latest.osc1.sjc.cloudera.com
# Optional:
# TAIKUN_DOMAIN_NAME=your-domain-name
```

## Usage

### Starting the Server

```bash
./cloudera-cloud-factory-mcp
```

The server will start and listen for MCP requests via stdio transport.

When running with Robot User credentials, the server reads the current Robot User identity and assigned scopes from the API and exposes them via the `robot-user-capabilities` MCP tool. For tools with a known scope mapping, the server will fail fast with a clear JSON error if the Robot User lacks the required scope.

### Connecting from Claude Desktop

Add this configuration to your Claude Desktop config:

```json
{
  "mcpServers": {
    "cloudera-cloud-factory": {
      "command": "/path/to/cloudera-cloud-factory-mcp",
      "env": {
        "TAIKUN_ACCESS_KEY": "your-robot-user-access-key",
        "TAIKUN_SECRET_KEY": "your-robot-user-secret-key",
        "TAIKUN_API_HOST": "api-latest.osc1.sjc.cloudera.com"
      }
    }
  }
}
```

## Support

For issues and questions:
- Create an issue in this repository
- Check the [Cloudera Cloud Factory documentation](https://docs.taikun.cloud/)
- Review the [MCP specification](https://modelcontextprotocol.io/)
