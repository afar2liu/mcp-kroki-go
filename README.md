# mcp-kroki-go

A Model Context Protocol (MCP) server for converting diagrams (Mermaid, PlantUML, GraphViz, etc.) to images using Kroki.io, implemented in Go.

## Features

- Generate diagram URLs for various diagram types (Mermaid, PlantUML, GraphViz, C4, etc.)
- Download diagrams as image files (SVG, PNG, PDF, JPEG)
- SVG scaling support
- Configurable Kroki server URL via environment variable
- Built with [mcp-go SDK](https://github.com/modelcontextprotocol/go-sdk)

## Installation

### Prerequisites

- Go 1.24 or higher

### Build from source

```bash
git clone https://github.com/afar2liu/mcp-kroki-go.git
cd mcp-kroki-go
go mod tidy
go build -o mcp-kroki-go
```

## Configuration

### Environment Variables

- `KROKI_SERVER_URL`: Custom Kroki server URL (default: `https://kroki.io`)

Example:
```bash
export KROKI_SERVER_URL=https://your-kroki-server.com
```

### MCP Client Configuration

Add to your MCP client configuration (e.g., Claude Desktop, Windsurf):

```json
{
  "mcpServers": {
    "kroki": {
      "command": "/absolute/path/to/mcp-kroki-go/mcp-kroki-go",
      "env": {
        "KROKI_SERVER_URL": "https://kroki.io"
      }
    }
  }
}
```

**Important:** 
- Use the **full absolute path** to the executable file (not the directory)
- Example: `/home/user/mcp-kroki-go/mcp-kroki-go` (not `/home/user/mcp-kroki-go`)
- Ensure the file has execute permissions: `chmod +x /path/to/mcp-kroki-go`

## Usage

The server provides two tools:

### 1. generate_diagram_url

Generate a URL for a diagram that can be embedded in documents or viewed in browsers.

**Parameters:**
- `type` (required): Diagram type (e.g., "mermaid", "plantuml", "graphviz")
- `content` (required): Diagram source code
- `outputFormat` (optional): Output format - "svg" (default), "png", "pdf", "jpeg", or "base64"

**Example:**
```
Generate a Mermaid diagram URL:
type: mermaid
content: graph TD; A-->B; B-->C;
outputFormat: svg
```

### 2. download_diagram

Download a diagram as an image file to your local system.

**Parameters:**
- `type` (required): Diagram type
- `content` (required): Diagram source code
- `outputPath` (required): Full path where the file should be saved
- `outputFormat` (optional): Output format (if not specified, derived from file extension)
- `scale` (optional): Scaling factor for SVG output (default: 1.0)

**Example:**
```
Download a PlantUML diagram:
type: plantuml
content: @startuml\nAlice -> Bob: Hello\n@enduml
outputPath: /home/user/diagram.svg
scale: 2.0
```

## Supported Diagram Types

- mermaid
- plantuml
- graphviz
- c4plantuml
- excalidraw
- erd
- svgbob
- nomnoml
- wavedrom
- blockdiag
- seqdiag
- actdiag
- nwdiag
- packetdiag
- rackdiag
- umlet
- ditaa
- vega
- vegalite

## Supported Output Formats

- svg (Scalable Vector Graphics)
- png (Portable Network Graphics)
- pdf (Portable Document Format)
- jpeg (JPEG Image)
- base64 (Base64-encoded SVG for HTML embedding)

## Error Handling

The server provides detailed error messages in Japanese for:
- Invalid diagram syntax
- Decoding errors
- HTTP errors from Kroki API
- File system errors

## Development

### Project Structure

```
mcp-kroki-go/
├── main.go          # Main server implementation
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
└── README.md        # This file
```

### Testing

```bash
# Build the project
go build

# Run the server (for testing with MCP client)
./mcp-kroki-go
```

## Differences from TypeScript Version

This Go implementation is functionally equivalent to the original TypeScript version with the following enhancements:

1. **Environment Variable Support**: Added `KROKI_SERVER_URL` environment variable to configure custom Kroki server
2. **Native Performance**: Go implementation provides better performance and lower memory usage
3. **Single Binary**: Compiles to a single executable with no runtime dependencies
4. **Cross-platform**: Easy to build for different platforms (Linux, macOS, Windows)

## License

MIT

## Credits

- Original TypeScript implementation: [mcp-kroki](https://github.com/tkoba1974/mcp-kroki)
- Kroki.io: https://kroki.io/
- MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk
