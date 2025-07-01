# Contributing Guide

Thank you for considering contributing to the MCP Milvus project! We welcome all forms of contributions, including but not limited to:

- 🐛 Bug reports
- ✨ Feature suggestions
- 📖 Documentation improvements
- 🧪 Test cases
- 💻 Code contributions

## Development Environment Setup

### Prerequisites

- Go 1.24+
- Git
- Docker (optional, for containerized testing)

### Local Development Setup

1. **Fork and clone the repository**
```bash
git clone https://github.com/YOUR_USERNAME/mcp-milvus.git
cd mcp-milvus
```

2. **Install dependencies**
```bash
go mod download
```

3. **Run tests to ensure everything works**
```bash
go test ./...
```

4. **Build the project**
```bash
go build -o mcp-milvus ./cmd/mcp-milvus
```

## Code Standards

### Go Code Style

- Use `gofmt` to format code
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names
- Add appropriate comments, especially for public APIs

### Commit Standards

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types include:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code formatting (no functional impact)
- `refactor`: Code refactoring
- `test`: Add tests
- `chore`: Build process or auxiliary tool changes

Example:
```
feat(tools): add new vector search capabilities

Add support for hybrid search with both semantic and keyword matching.
This includes new parameters for search configuration and result ranking.

Closes #123
```

## Development Workflow

### 1. Create Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number
```

### 2. Development and Testing

- Write code
- Add/update tests
- Ensure all tests pass
- Update relevant documentation

### 3. Commit Code

```bash
git add .
git commit -m "feat: add awesome new feature"
```

### 4. Push and Create PR

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Testing Guide

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/schema

# Run tests with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

- Write unit tests for new features
- Ensure test coverage is not less than 80%
- Use table-driven test patterns
- Include boundary conditions and error cases

Example test:
```go
func TestNewTool(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "test_result",
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := NewTool(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Adding New Tools

If you want to add new Milvus tools, please follow these steps:

### 1. Create Tool File

Create a new file in the `internal/tools/` directory, for example `milvus_new_feature.go`:

```go
package tools

import (
    "context"
    "github.com/tailabs/mcp-milvus/internal/registry"
    "github.com/tailabs/mcp-milvus/internal/session"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// NewFeatureTool represents the new feature tool
type NewFeatureTool struct{}

// Tool registrar
func (t *NewFeatureTool) GetTool() mcp.Tool {
    return mcp.Tool{
        Name:        "milvus_new_feature",
        Description: "Description of the new feature",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type":        "string",
                    "description": "Parameter description",
                },
            },
            "required": []string{"param1"},
        },
    }
}

func (t *NewFeatureTool) GetHandler() server.ToolHandlerFunc {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Implementation here
        return &mcp.CallToolResult{
            Content: []interface{}{
                map[string]interface{}{
                    "type": "text",
                    "text": "Success result",
                },
            },
        }, nil
    }
}

// Auto-register tool
func init() {
    registry.RegisterTool(&NewFeatureTool{})
}
```

### 2. Add Tests

Create corresponding test file `milvus_new_feature_test.go`.

### 3. Update Documentation

Add the new tool description to README.md.

## Issues and Bug Reports

### Reporting Bugs

Please use the [Bug Report template](https://github.com/tailabs/mcp-milvus/issues/new?template=bug_report.md) to report bugs, including:

- Detailed problem description
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment information (Go version, OS, etc.)
- Relevant logs and error messages

### Feature Requests

Use the [Feature Request template](https://github.com/tailabs/mcp-milvus/issues/new?template=feature_request.md) to suggest new features.

## Pull Request Guidelines

### PR Checklist

Before submitting a PR, ensure:

- [ ] Code passes all tests
- [ ] Code follows project coding standards
- [ ] Commit messages are clear and meaningful
- [ ] Includes necessary test cases
- [ ] Updates relevant documentation
- [ ] PR description clearly explains changes

### PR Template

Our PR template includes:

```markdown
## Change Type
- [ ] Bug fix
- [ ] New feature
- [ ] Refactoring
- [ ] Documentation update
- [ ] Other

## Change Description
<!-- Describe your changes in detail -->

## Testing
<!-- Describe how you tested these changes -->

## Checklist
- [ ] All tests pass
- [ ] Code follows project standards
- [ ] Updated documentation
- [ ] Added test cases
```

## Community

- 💬 [GitHub Discussions](https://github.com/tailabs/mcp-milvus/discussions) - General discussions
- 🐛 [GitHub Issues](https://github.com/tailabs/mcp-milvus/issues) - Bug reports and feature requests
- 📧 Email: For sensitive issues, you can send email to [maintainer email]

## Code of Conduct

We are committed to providing a friendly, safe, and welcoming environment for everyone. Please:

- Use friendly and inclusive language
- Respect different viewpoints and experiences
- Gracefully accept constructive criticism
- Focus on what is best for the community
- Show empathy towards other community members

## License

By contributing code, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

Thanks again for your contribution! 🎉 