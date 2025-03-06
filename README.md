# MCP (Model Context Protocol) Tools

A list of tools to provide different capabilities to LLMs (Language Model Models).
The tools adhere to the Model Context Protocol (MCP) specification to provide a common interface for LLMs to interact with.

```
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

## Available Tools

| Tool          | Description                              | Use-cases                               |
|---------------|------------------------------------------|-----------------------------------------|
| DataExtractor | Extracts structured data from input.     | Parsing documents, forms, or emails.    |
| TextAnalyzer  | Analyzes text for sentiment or keywords. | Sentiment analysis, keyword extraction. |
| Translator    | Translates text between languages.       | Multilingual support for applications.  |

### GetWeather (Dummy Tool)

This is a dummy tool that returns a fixed weather forecast for a given location.
It is used for testing purposes only.

```json
{
  "name": "GetWeather",
  "description": "Returns a fixed weather forecast for a given location.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "location": {
        "type": "string",
        "description": "The location to get the weather forecast for."
      }
    },
    "required": ["location"]
  }
}
```

## Contributing
Contributions to this open-source package are welcome! If you'd like to contribute, please start by reviewing
the [MCP Tools documentation](https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure) and ensure
adherence to the MCP specification. You can contribute by suggesting new tools, reporting bugs, improving existing
implementations, or enhancing documentation. To contribute, fork the repository, create a new branch for your changes,
and submit a pull request with a detailed description of your contribution. Please follow the project's coding standards
and ensure that all changes include appropriate tests and documentation updates. Thank you for supporting this project!

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
