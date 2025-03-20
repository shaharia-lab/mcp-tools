# MCP Tools

## Overview
A collection of tools for various integrations and utilities.

## Available Tools
- GitHub Repository Management
- GitHub Issues Management
- GitHub Pull Request Management
- Jira Integration
- Confluence Integration

## Installation

```bash
go get github.com/shaharia-lab/mcp-tools
```

## Usage

### Jira Client
```go
jiraClient := mcptools.NewJiraClient("https://your-jira-url.com", "your-auth-token")
issues, err := jiraClient.SearchJiraIssues("project = MyProject")
```

### Confluence Client
```go
confluenceClient := mcptools.NewConfluenceClient("https://your-confluence-url.com", "your-auth-token")
pages, err := confluenceClient.SearchConfluencePages("keyword")
```