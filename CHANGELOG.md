# Changelog

## [1.0.0] - 2025-01-15

### Added
- Initial Go implementation of mcp-kroki
- Support for all Kroki diagram types (Mermaid, PlantUML, GraphViz, etc.)
- Two MCP tools:
  - `generate_diagram_url`: Generate URLs for diagrams
  - `download_diagram`: Download diagrams as image files
- SVG scaling support
- **New Feature**: `KROKI_SERVER_URL` environment variable to configure custom Kroki server
- Comprehensive error handling with Japanese error messages
- Support for multiple output formats (SVG, PNG, PDF, JPEG, base64)

### Technical Details
- Built with official `modelcontextprotocol/go-sdk` v0.2.0
- Requires Go 1.23.2+ (uses Go toolchain auto-download)
- Single binary executable with no runtime dependencies
- Cross-platform support (Linux, macOS, Windows)

### Differences from TypeScript Version
1. Added `KROKI_SERVER_URL` environment variable support
2. Native Go performance and lower memory usage
3. Single binary distribution
4. Easier cross-platform builds
