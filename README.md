# MCP Milvus

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for Milvus vector database, providing comprehensive vector database operations.

## 🚀 Features

- **Complete Milvus Operations**: Full lifecycle management of databases, collections, and indexes
- **High-Performance Vector Search**: Support for similarity search, hybrid search, and more retrieval methods
- **Intelligent Session Management**: Efficient connection pooling based on Ristretto cache
- **Engineering Architecture**: Modular design for easy extension and maintenance
- **Middleware Support**: Built-in logging, authentication, and other middleware
- **Docker Support**: Complete containerized deployment solution
- **Type Safety**: Go's strong type system ensures API safety

## 📋 Supported Tools

### Database Management
- `milvus_create_database` - Create database
- `milvus_list_databases` - List all databases
- `milvus_use_database` - Switch database

### Collection Management
- `milvus_create_collection` - Create collection
- `milvus_drop_collection` - Drop collection
- `milvus_list_collections` - List collections
- `milvus_get_collection_info` - Get collection information
- `milvus_rename_collection` - Rename collection
- `milvus_load_collection` - Load collection into memory
- `milvus_release_collection` - Release collection from memory

### Index Management
- `milvus_create_index` - Create index
- `milvus_drop_index` - Drop index

### Data Operations
- `milvus_insert_data` - Insert data
- `milvus_upsert` - Insert or update data
- `milvus_delete_entities` - Delete entities
- `milvus_query` - Conditional query
- `milvus_vector_search` - Vector similarity search

### Connection Management
- `milvus_connector` - Establish Milvus connection

## 🛠️ Installation and Usage

### Prerequisites

- Go 1.24 or higher
- Running Milvus instance

### Install from Source

```bash
git clone https://github.com/tailabs/mcp-milvus.git
cd mcp-milvus
make deps
make build
```

Or using Go directly:
```bash
git clone https://github.com/tailabs/mcp-milvus.git
cd mcp-milvus
go mod download
go build -o mcp-milvus ./cmd/mcp-milvus
```

### Docker Deployment

```bash
# Build image
docker build -t mcp-milvus .

# Run container
docker run -p 8080:8080 mcp-milvus
```

### Usage

1. **Start the server**
```bash
# Using Makefile
make run

# Or directly
./build/mcp-milvus
```

2. **Connect to Milvus**
Use the `milvus_connector` tool to establish connection:
```json
{
  "address": "localhost:19530",
  "token": "username:password",
  "db_name": "default"
}
```

3. **Perform operations**
After connection is established, you can use other tools for database operations.

## 🏗️ Project Structure

```
mcp-milvus/
├── cmd/mcp-milvus/          # Main application entry
├── internal/
│   ├── middleware/          # Middleware (logging, auth, etc.)
│   ├── registry/            # Tool registry
│   ├── schema/              # Schema builder
│   ├── session/             # Session management
│   └── tools/               # Milvus tool implementations
├── Dockerfile               # Docker build file
├── go.mod                   # Go module definition
└── README.md               # Project documentation
```

## 🔧 Configuration

### Environment Variables

- `LOG_LEVEL`: Log level (debug/info/warn/error), default: info
- `PORT`: Service port, default: 8080

### Connection Configuration

Supports the following connection parameters:
- `address`: Milvus service address
- `token`: Authentication token (format: username:password)
- `db_name`: Database name

## 🤝 Contributing

We welcome all forms of contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork this repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Set up development environment (`make deps && make tools`)
4. Make your changes and test (`make test && make lint`)
5. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
6. Push to the branch (`git push origin feature/AmazingFeature`)
7. Open a Pull Request

### Available Make Commands

Run `make help` to see all available commands:

```bash
make help          # Show all available commands
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make fmt           # Format code
make run           # Build and run
make dev           # Run with live reload
make docker        # Build Docker image
make build-all     # Build for all platforms
make release       # Prepare release
make clean         # Clean build artifacts
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Links

- [Milvus Official Website](https://milvus.io/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/mark3labs/mcp-go)

## 📞 Support

If you encounter any issues or have questions:

1. Check [Issues](https://github.com/tailabs/mcp-milvus/issues) to see if similar issues exist
2. Create a new Issue describing your problem
3. Join discussions in Discussions

## 🙏 Acknowledgments

- [Milvus](https://github.com/milvus-io/milvus) - Excellent vector database
- [MCP Go](https://github.com/mark3labs/mcp-go) - Go implementation of MCP
- All contributors and users

---

**⭐ If this project helps you, please give it a Star!**
