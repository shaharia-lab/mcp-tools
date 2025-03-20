# MCP Tools

## Overview
A collection of tools for various integrations and utilities.

## New Features

### Jira Integration
- Search Jira issues
- Read Jira issue details
- Create new Jira issues

### Confluence Integration
- Search Confluence pages
- Read Confluence page details

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