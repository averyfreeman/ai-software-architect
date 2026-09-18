// Package cli provides CLI command implementations for adr-rg.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format for commands.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// ParseOutputFormat parses a normal command's output format.
func ParseOutputFormat(s string) (OutputFormat, error) {
	return parseOutputFormat(s, false)
}

// ParseIndexOutputFormat parses an index command output format, including YAML.
func ParseIndexOutputFormat(s string) (OutputFormat, error) {
	return parseOutputFormat(s, true)
}

func parseOutputFormat(s string, allowYAML bool) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		if allowYAML {
			return FormatYAML, nil
		}
		return "", fmt.Errorf("invalid format %q: normal commands must use text or json", s)
	default:
		return "", fmt.Errorf("invalid format %q: must be text or json", s)
	}
}

// Output handles writing output in different formats.
type Output struct {
	Format OutputFormat
	Writer io.Writer
}

// NewOutput creates a new Output with the given format.
func NewOutput(format OutputFormat) *Output {
	return &Output{
		Format: format,
		Writer: os.Stdout,
	}
}

// Print outputs a message (text format only).
func (o *Output) Print(format string, args ...interface{}) {
	if o.Format == FormatText {
		_, _ = fmt.Fprintf(o.Writer, format, args...)
	}
}

// Println outputs a line (text format only).
func (o *Output) Println(format string, args ...interface{}) {
	if o.Format == FormatText {
		_, _ = fmt.Fprintf(o.Writer, format+"\n", args...)
	}
}

// PrintStructured outputs data in the configured structured format (JSON or YAML).
func (o *Output) PrintStructured(data interface{}) error {
	switch o.Format {
	case FormatJSON:
		return o.PrintJSON(data)
	case FormatYAML:
		return o.PrintYAML(data)
	default:
		return o.PrintJSON(data)
	}
}

// PrintJSON outputs data as JSON.
func (o *Output) PrintJSON(data interface{}) error {
	encoder := json.NewEncoder(o.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// PrintYAML outputs data as YAML.
func (o *Output) PrintYAML(data interface{}) error {
	encoded, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = o.Writer.Write(encoded)
	return err
}

// Error outputs an error message to stderr.
func (o *Output) Error(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

// Warn outputs a warning message to stderr.
func (o *Output) Warn(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// Success outputs a success message.
func (o *Output) Success(format string, args ...interface{}) {
	o.Println("✓ "+format, args...)
}

// Info outputs an informational message.
func (o *Output) Info(format string, args ...interface{}) {
	o.Println(format, args...)
}

// IsStructuredFormat returns true if the format is a structured data format.
func (o *Output) IsStructuredFormat() bool {
	return o.Format == FormatJSON || o.Format == FormatYAML
}
