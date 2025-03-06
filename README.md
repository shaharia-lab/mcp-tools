# MCP (Model Context Protocol) Tools

A list of tools to provide different capabilities to LLMs (Language Model Models).
The tools adhere to the Model Context Protocol (MCP) specification to provide a common interface for LLMs to interact with.

```json
{
  name: string;          // Unique identifier for the tool
  description?: string;  // Human-readable description
  inputSchema: {         // JSON Schema for the tool's parameters
    type: "object",
    properties: { ... }  // Tool-specific parameters
  }
}
```

[Here](https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure) is the definition speicification for the MCP Tools.
