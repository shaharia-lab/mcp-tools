# MCP (Model Context Protocol) Tools

A list of tools to provide different capabilities to LLMs (Language Model Models).
The tools adhere to the Model Context Protocol (MCP) specification to provide a common interface for LLMs to interact with.

**Note:** If you want to see the tools in action, you can use the [MCP Kit](https://github.com/shaharia-lab/mcp-kit) project.

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

[Here](https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure) is the definition specification for the MCP Tools.

## Available Tools

| Tool   | Name                   | Description                                                                     | Use-cases                                                                   |
|--------|------------------------|---------------------------------------------------------------------------------|-----------------------------------------------------------------------------|
| bash   | `bash`                 | Execute bash commands and shell scripts.                                        | System command execution, scripting, automation tasks.                      |
| cat    | `cat`                  | Read and display file contents.                                                 | File inspection, quick content viewing.                                     |
| cURL   | `curl`                 | A versatile tool for making HTTP requests and interacting with APIs.            | Fetching data from APIs, web scraping, testing endpoints.                   |
| docker | `docker`               | A tool for managing Docker containers and images.                               | Building, running, and deploying applications in containers.                |
| file_system | `file_system`       | Perform filesystem operations like list, read, write, create, delete files.    | File management, directory manipulation, content manipulation.              |
| git    | `git`                  | A tool for interacting with Git repositories.                                   | Managing code repositories, version control, collaboration.                 |
| github | `github_issues`        | Manages GitHub issues - create, list, update, comment on issues.                | Managing GitHub issues. Required `GITHUB_TOKEN` environment variable        |
| github | `github_pull_requests` | Manages GitHub pull requests - create, review, merge.                           | Managing GitHub pull requests. Required `GITHUB_TOKEN` environment variable |
| github | `github_repository`    | Manages GitHub repositories - create, delete, update, fork.                     | Repository management. Required `GITHUB_TOKEN` environment variable         |
| github | `github_search`        | Performs GitHub search operations across repositories, code, issues, and users. | Advanced GitHub searches. Required `GITHUB_TOKEN` environment variable      |
| gmail  | `gmail`                | Gmail operation to execute (list, send, read, delete).                          | Managing Gmail operations                                                   |
| grep   | `grep`                 | Search for text patterns in files or directories.                               | Text searching, log analysis, pattern matching.                             |
| postgresql | `postgresql`        | Interact with PostgreSQL databases.                                             | Database querying, data retrieval, database management.                     |
| sed    | `sed`                  | Stream editor for filtering and transforming text.                              | Text manipulation, regex-based stream editing.                              |
| weather| `get_weather`          | Retrieve current weather information.                                           | Weather data retrieval, location-based weather queries.                     |

## Contributing
Contributions to this open-source package are welcome! If you'd like to contribute, please start by reviewing
the [MCP Tools documentation](https://modelcontextprotocol.io/docs/concepts/tools#tool-definition-structure) and ensure
adherence to the MCP specification. You can contribute by suggesting new tools, reporting bugs, improving existing
implementations, or enhancing documentation. To contribute, fork the repository, create a new branch for your changes,
and submit a pull request with a detailed description of your contribution. Please follow the project's coding standards
and ensure that all changes include appropriate tests and documentation updates. Thank you for supporting this project!

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.