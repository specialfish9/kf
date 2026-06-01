package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var logFMTRegexp = regexp.MustCompile(`(\w+)=([^\s]+)`)

func colorLine(line string) string {
	if s := colorLogfmt(line); s != "" {
		return s
	}
	if s := colorJSON(line); s != "" {
		return s
	}

	return line
}

// colorLogfmt takes a log line and returns a colored version if it's logfmt
func colorLogfmt(line string) string {
	// Regex to match key=value pairs
	matches := logFMTRegexp.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return ""
	}

	// Define styles
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	var coloredParts []string
	for _, m := range matches {
		key := keyStyle.Render(m[1])
		val := valStyle.Render(m[2])
		coloredParts = append(coloredParts, fmt.Sprintf("%s=%s", key, val))
	}

	return strings.Join(coloredParts, " ")
}

// colorJSON takes a log line and returns a colored version if it's JSON
func colorJSON(line string) string {
	var data any
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return ""
	}

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green

	return renderJSON(data, keyStyle, valStyle, 0)
}

func renderJSON(v interface{}, keyStyle, valStyle lipgloss.Style, indent int) string {
	indentStr := strings.Repeat("  ", indent)

	switch val := v.(type) {
	case map[string]interface{}:
		var parts []string
		for k, subVal := range val {
			coloredKey := keyStyle.Render(fmt.Sprintf("\"%s\"", k))
			parts = append(parts, fmt.Sprintf("%s%s: %s", indentStr, coloredKey, renderJSON(subVal, keyStyle, valStyle, indent+1)))
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + indentStr + "}"
	case []interface{}:
		var parts []string
		for _, subVal := range val {
			parts = append(parts, fmt.Sprintf("%s%s", indentStr, renderJSON(subVal, keyStyle, valStyle, indent+1)))
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
