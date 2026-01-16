package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultKrokiURL = "https://kroki.io"
)

var (
	validDiagramTypes = []string{
		"mermaid", "plantuml", "graphviz", "c4plantuml",
		"excalidraw", "erd", "svgbob", "nomnoml", "wavedrom",
		"blockdiag", "seqdiag", "actdiag", "nwdiag", "packetdiag",
		"rackdiag", "umlet", "ditaa", "vega", "vegalite",
		"bpmn", "bytefield", "d2", "dbml", "pikchr",
		"structurizr", "symbolator", "tikz", "wireviz",
	}
	validOutputFormats = []string{"svg", "png", "pdf", "jpeg", "base64"}
)

type KrokiServer struct {
	baseURL string
}

func NewKrokiServer() *KrokiServer {
	baseURL := os.Getenv("KROKI_SERVER_URL")
	if baseURL == "" {
		baseURL = defaultKrokiURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &KrokiServer{baseURL: baseURL}
}

func (ks *KrokiServer) validateDiagramType(diagramType string) error {
	for _, valid := range validDiagramTypes {
		if diagramType == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid diagram type. Must be one of: %s", strings.Join(validDiagramTypes, ", "))
}

func (ks *KrokiServer) validateOutputFormat(format string) error {
	for _, valid := range validOutputFormats {
		if format == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid output format. Must be one of: %s", strings.Join(validOutputFormats, ", "))
}

func (ks *KrokiServer) encodeContent(content string) (string, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	return encoded, nil
}

func (ks *KrokiServer) getDiagramData(diagramType, content, outputFormat string, scale float64) ([]byte, error) {
	if err := ks.validateDiagramType(diagramType); err != nil {
		return nil, err
	}
	if err := ks.validateOutputFormat(outputFormat); err != nil {
		return nil, err
	}

	encodedContent, err := ks.encodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to encode content: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", ks.baseURL, diagramType, outputFormat, encodedContent)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch diagram from Kroki: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(contentType, "text/html") || strings.HasPrefix(string(data), "<html") || strings.HasPrefix(string(data), "<!DOCTYPE html") {
			htmlContent := string(data)
			titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
			bodyRegex := regexp.MustCompile(`<body>([\s\S]*?)</body>`)

			extractedMessage := "Unknown error (HTML response)"
			if matches := titleRegex.FindStringSubmatch(htmlContent); len(matches) > 1 {
				extractedMessage = strings.TrimSpace(matches[1])
			}

			if bodyMatches := bodyRegex.FindStringSubmatch(htmlContent); len(bodyMatches) > 1 {
				if strings.Contains(strings.ToLower(bodyMatches[1]), "unable to decode") {
					extractedMessage = "Decode error: Kroki could not decode the source. Please check the format and content."
					preRegex := regexp.MustCompile(`<pre>([\s\S]*?)</pre>`)
					if preMatches := preRegex.FindStringSubmatch(bodyMatches[1]); len(preMatches) > 1 {
						extractedMessage += fmt.Sprintf("\nDetails:\n---\n%s\n---", strings.TrimSpace(preMatches[1]))
					}
				}
			}
			return nil, fmt.Errorf("Kroki error: %s", extractedMessage)
		}

		if resp.StatusCode == http.StatusBadRequest {
			krokiMessage := string(data)
			if len(krokiMessage) < 500 {
				return nil, fmt.Errorf("There appears to be an error in the diagram description (Kroki HTTP 400).\nDetails from Kroki:\n---\n%s\n---\nPlease check the diagram content.", strings.TrimSpace(krokiMessage))
			}
			return nil, fmt.Errorf("There appears to be an error in the diagram description (Kroki HTTP 400). Please check the diagram content.")
		}

		return nil, fmt.Errorf("Kroki API request failed (status: %d)", resp.StatusCode)
	}

	if (outputFormat == "svg" || outputFormat == "base64") && len(data) > 0 {
		svgContent := string(data)

		errorTextRegex := regexp.MustCompile(`<text[^>]*class="error"[^>]*>([\s\S]*?)</text>|<text[^>]*fill="red"[^>]*>([\s\S]*?)</text>`)
		if matches := errorTextRegex.FindStringSubmatch(svgContent); len(matches) > 0 {
			rawErrorMessage := ""
			if matches[1] != "" {
				rawErrorMessage = matches[1]
			} else if matches[2] != "" {
				rawErrorMessage = matches[2]
			}
			rawErrorMessage = strings.TrimSpace(rawErrorMessage)
			decodedErrorMessage := strings.ReplaceAll(rawErrorMessage, "&lt;", "<")
			decodedErrorMessage = strings.ReplaceAll(decodedErrorMessage, "&gt;", ">")
			decodedErrorMessage = strings.ReplaceAll(decodedErrorMessage, "&amp;", "&")
			decodedErrorMessage = strings.ReplaceAll(decodedErrorMessage, "<br/>", "\n")
			return nil, fmt.Errorf("Diagram generation error (in SVG):\n%s\n\nPlease check the diagram content.", decodedErrorMessage)
		}

		if scale > 1.0 && outputFormat == "svg" {
			svgContent = ks.scaleSVG(svgContent, scale)
			data = []byte(svgContent)
			fmt.Fprintf(os.Stderr, "[KrokiServer] Applied scale %.2f to SVG.\n", scale)
		}
	}

	return data, nil
}

func (ks *KrokiServer) scaleSVG(svgContent string, scale float64) string {
	svgRegex := regexp.MustCompile(`<svg([^>]*)>`)
	return svgRegex.ReplaceAllStringFunc(svgContent, func(match string) string {
		attributes := match[4 : len(match)-1]

		widthRegex := regexp.MustCompile(`width="([^"]+)"`)
		heightRegex := regexp.MustCompile(`height="([^"]+)"`)

		attributes = widthRegex.ReplaceAllStringFunc(attributes, func(widthMatch string) string {
			value := widthRegex.FindStringSubmatch(widthMatch)[1]
			numRegex := regexp.MustCompile(`^([0-9.]+)(.*)$`)
			if numMatches := numRegex.FindStringSubmatch(value); len(numMatches) > 1 {
				if widthVal, err := strconv.ParseFloat(numMatches[1], 64); err == nil {
					unit := numMatches[2]
					if unit == "" {
						unit = "px"
					}
					return fmt.Sprintf(`width="%.2f%s"`, widthVal*scale, unit)
				}
			}
			return widthMatch
		})

		attributes = heightRegex.ReplaceAllStringFunc(attributes, func(heightMatch string) string {
			value := heightRegex.FindStringSubmatch(heightMatch)[1]
			numRegex := regexp.MustCompile(`^([0-9.]+)(.*)$`)
			if numMatches := numRegex.FindStringSubmatch(value); len(numMatches) > 1 {
				if heightVal, err := strconv.ParseFloat(numMatches[1], 64); err == nil {
					unit := numMatches[2]
					if unit == "" {
						unit = "px"
					}
					return fmt.Sprintf(`height="%.2f%s"`, heightVal*scale, unit)
				}
			}
			return heightMatch
		})

		return "<svg" + attributes + ">"
	})
}

func (ks *KrokiServer) generateDiagramURL(diagramType, content, outputFormat string) (string, error) {
	if err := ks.validateDiagramType(diagramType); err != nil {
		return "", err
	}
	if err := ks.validateOutputFormat(outputFormat); err != nil {
		return "", err
	}

	checkFormat := outputFormat
	if outputFormat == "base64" {
		checkFormat = "svg"
	}

	if _, err := ks.getDiagramData(diagramType, content, checkFormat, 1.0); err != nil {
		return "", err
	}

	encodedContent, err := ks.encodeContent(content)
	if err != nil {
		return "", fmt.Errorf("failed to encode content: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s/%s", ks.baseURL, diagramType, outputFormat, encodedContent), nil
}

type GenerateDiagramURLInput struct {
	Type         string `json:"type" jsonschema:"Diagram type (e.g. mermaid plantuml graphviz c4plantuml). See Kroki.io documentation for all supported formats."`
	Content      string `json:"content" jsonschema:"The diagram content in the specified format."`
	OutputFormat string `json:"outputFormat,omitempty" jsonschema:"Output format: svg (default) png pdf jpeg or base64."`
}

type GenerateDiagramURLOutput struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

func GenerateDiagramURL(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GenerateDiagramURLInput]) (*mcp.CallToolResultFor[GenerateDiagramURLOutput], error) {
	krokiServer := NewKrokiServer()

	input := params.Arguments
	outputFormat := input.OutputFormat
	if outputFormat == "" {
		outputFormat = "svg"
	}

	url, err := krokiServer.generateDiagramURL(input.Type, input.Content, outputFormat)
	if err != nil {
		return &mcp.CallToolResultFor[GenerateDiagramURLOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to generate diagram URL: %v", err)}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResultFor[GenerateDiagramURLOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Diagram URL generated and validated successfully. No errors found.\nURL: %s", url)}},
		StructuredContent: GenerateDiagramURLOutput{
			Message: "Diagram URL generated and validated successfully. No errors found.",
			URL:     url,
		},
	}, nil
}

type DownloadDiagramInput struct {
	Type         string  `json:"type" jsonschema:"Diagram type (e.g. mermaid plantuml graphviz). Supports the same diagram types as Kroki.io."`
	Content      string  `json:"content" jsonschema:"The diagram content in the specified format."`
	OutputPath   string  `json:"outputPath" jsonschema:"Complete file path where the diagram should be saved."`
	OutputFormat string  `json:"outputFormat,omitempty" jsonschema:"Output format (svg png pdf jpeg). If unspecified derived from file extension."`
	Scale        float64 `json:"scale,omitempty" jsonschema:"Scaling factor for SVG output (default 1.0)."`
}

type DownloadDiagramOutput struct {
	Message string `json:"message"`
}

func DownloadDiagram(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DownloadDiagramInput]) (*mcp.CallToolResultFor[DownloadDiagramOutput], error) {
	krokiServer := NewKrokiServer()

	input := params.Arguments
	outputFormat := input.OutputFormat
	if outputFormat == "" {
		ext := filepath.Ext(input.OutputPath)
		if len(ext) > 1 {
			outputFormat = ext[1:]
		} else {
			outputFormat = "svg"
		}
	}

	scale := input.Scale
	if scale == 0 {
		scale = 1.0
	}

	data, err := krokiServer.getDiagramData(input.Type, input.Content, outputFormat, scale)
	if err != nil {
		return &mcp.CallToolResultFor[DownloadDiagramOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to download diagram to %s: %v", input.OutputPath, err)}},
			IsError: true,
		}, nil
	}

	dir := filepath.Dir(input.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &mcp.CallToolResultFor[DownloadDiagramOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to create directory %s: %v", dir, err)}},
			IsError: true,
		}, nil
	}

	if err := os.WriteFile(input.OutputPath, data, 0644); err != nil {
		return &mcp.CallToolResultFor[DownloadDiagramOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Failed to write file %s: %v", input.OutputPath, err)}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResultFor[DownloadDiagramOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Diagram saved to %s", input.OutputPath)}},
		StructuredContent: DownloadDiagramOutput{
			Message: fmt.Sprintf("Diagram saved to %s", input.OutputPath),
		},
	}, nil
}

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "kroki-server",
			Version: "1.0.0",
		},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_diagram_url",
		Description: "Generate a URL for a diagram using Kroki.io. This tool takes Mermaid diagram code or other supported diagram formats and returns a URL to the rendered diagram. The URL can be used to display the diagram in web browsers or embedded in documents. Supported diagram types: mermaid, plantuml, graphviz, c4plantuml, excalidraw, erd, svgbob, nomnoml, wavedrom, blockdiag, seqdiag, actdiag, nwdiag, packetdiag, rackdiag, umlet, ditaa, vega, vegalite, bpmn, bytefield, d2, dbml, pikchr, structurizr, symbolator, tikz, wireviz. Supported output formats: svg, png, pdf, jpeg, base64.",
	}, GenerateDiagramURL)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "download_diagram",
		Description: "Download a diagram image to a local file. This tool converts diagram code (such as Mermaid) into an image file and saves it to the specified location. Useful for generating diagrams for presentations, documentation, or other offline use. Includes an option to scale SVG output. Supported diagram types: mermaid, plantuml, graphviz, c4plantuml, excalidraw, erd, svgbob, nomnoml, wavedrom, blockdiag, seqdiag, actdiag, nwdiag, packetdiag, rackdiag, umlet, ditaa, vega, vegalite, bpmn, bytefield, d2, dbml, pikchr, structurizr, symbolator, tikz, wireviz. Supported output formats: svg, png, pdf, jpeg, base64.",
	}, DownloadDiagram)

	if err := server.Run(context.Background(), mcp.NewStdioTransport()); err != nil {
		log.Fatal(err)
	}
}
