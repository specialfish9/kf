package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logFMTRegexp matches key=value pairs in logfmt format, handling quoted values
var logFMTRegexp = regexp.MustCompile(`(\w+)=("(?:[^"\\]|\\.)*"|[^\s]+)`)

// isLogFMTRegexp checks if a line is in logfmt format (key=value pairs)
var isLogFMTRegexp = regexp.MustCompile(`^(\w+=(?:[^\s"]+|"[^"]*"))(\s+\w+=(?:[^\s"]+|"[^"]*"))*$`)

func formatLine(line string, format bool) string {
	if s := colorLogfmt(line, format); s != "" {
		return s
	}
	if s := colorJSON(line, format); s != "" {
		return s
	}

	return line
}

// colorLogfmt takes a log line and returns a colored version if it's logfmt
func colorLogfmt(line string, format bool) string {
	if !isLogFMTRegexp.MatchString(line) {
		return ""
	}

	// Regex to match key=value pairs
	matches := logFMTRegexp.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return ""
	}

	// Define styles
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	maxKeyLen := 0
	for _, m := range matches {
		if len(m[1]) > maxKeyLen {
			maxKeyLen = len(m[1])
		}
	}

	var coloredParts []string
	for _, m := range matches {
		if format {
			// left-align key
			paddedKey := fmt.Sprintf("%-*s", maxKeyLen, m[1])
			key := keyStyle.Render(paddedKey)
			val := valStyle.Render(m[2])
			coloredParts = append(coloredParts, fmt.Sprintf("%s │ %s", key, val))
		} else {
			key := keyStyle.Render(m[1])
			val := valStyle.Render(m[2])
			coloredParts = append(coloredParts, fmt.Sprintf("%s=%s", key, val))
		}
	}

	if format {
		coloredParts = append(coloredParts, "---")
		return strings.Join(coloredParts, "\n")

	}
	return strings.Join(coloredParts, " ")

}

// colorJSON takes a log line and returns a colored version if it's JSON
func colorJSON(line string, format bool) string {
	var data any
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return ""
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	return renderJSON(data, keyStyle, valStyle, 0, format)
}

func renderJSON(v any, keyStyle, valStyle lipgloss.Style, indent int, format bool) string {
	var indentStr string
	if format {
		indentStr = strings.Repeat("  ", indent)
	}

	switch val := v.(type) {
	case map[string]interface{}:
		var parts []string
		for k, subVal := range val {
			coloredKey := keyStyle.Render(fmt.Sprintf("\"%s\"", k))
			parts = append(parts, fmt.Sprintf("%s%s: %s", indentStr, coloredKey, renderJSON(subVal, keyStyle, valStyle, indent+1, format)))
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + indentStr + "}"
	case []interface{}:
		var parts []string
		for _, subVal := range val {
			parts = append(parts, fmt.Sprintf("%s%s", indentStr, renderJSON(subVal, keyStyle, valStyle, indent+1, format)))
		}
		return "[\n" + strings.Join(parts, ",\n") + "\n" + indentStr + "]"
	case string:
		return valStyle.Render(fmt.Sprintf("\"%s\"", val))
	case float64, bool, nil:
		return valStyle.Render(fmt.Sprintf("%v", val))
	default:
		return valStyle.Render(fmt.Sprintf("%v", val))
	}
}
