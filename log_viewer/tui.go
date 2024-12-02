package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	highlightColor  = lipgloss.Color("39")  // Blue
	normalColor     = lipgloss.Color("252") // Light gray
	headerColor     = lipgloss.Color("105") // Light purple
	errorColor      = lipgloss.Color("196") // Red
	warnColor       = lipgloss.Color("214") // Orange
	infoColor       = lipgloss.Color("83")  // Green
	jsonKeyColor    = lipgloss.Color("105") // Purple for JSON keys
	jsonStringColor = lipgloss.Color("83")  // Green for strings
	jsonNumberColor = lipgloss.Color("214") // Orange for numbers
	jsonNullColor   = lipgloss.Color("245") // Gray for null values

	// Header style
	headerStyle = lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1).
			MarginBottom(1)

	// Log styles
	logStyle = lipgloss.NewStyle().
			Foreground(normalColor)

	selectedLogStyle = lipgloss.NewStyle().
				Foreground(highlightColor).
				Bold(true).
				Background(lipgloss.Color("236"))

	// Search overlay style
	searchStyle = lipgloss.NewStyle().
			Foreground(highlightColor).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			MarginTop(1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			Padding(1)

	// JSON highlighting styles
	jsonKeyStyle = lipgloss.NewStyle().
			Foreground(jsonKeyColor).
			Bold(true)

	jsonStringStyle = lipgloss.NewStyle().
			Foreground(jsonStringColor)

	jsonNumberStyle = lipgloss.NewStyle().
			Foreground(jsonNumberColor)

	jsonNullStyle = lipgloss.NewStyle().
			Foreground(jsonNullColor).
			Italic(true)
)

type Model struct {
	logs             []ParsedLog
	filteredLogs     []ParsedLog
	selectedLogIndex int
	searchMode       bool
	jumpMode         bool
	searchQuery      string
	width            int
	height           int
}

func filterLogs(logs []ParsedLog, query string) []ParsedLog {
	if query == "" {
		return logs
	}

	var filtered []ParsedLog
	lowerQuery := strings.ToLower(query)

	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.RawLog), lowerQuery) {
			filtered = append(filtered, log)
			continue
		}

		for key, value := range log.Fields {
			if strings.Contains(strings.ToLower(fmt.Sprint(key)), lowerQuery) ||
				strings.Contains(strings.ToLower(fmt.Sprint(value)), lowerQuery) {
				filtered = append(filtered, log)
				break
			}
		}
	}

	return filtered
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.selectedLogIndex > 0 {
				m.selectedLogIndex--
			}
		case "down", "j":
			if m.selectedLogIndex < len(m.filteredLogs)-1 {
				m.selectedLogIndex++
			}
		case "/":
			m.jumpMode = true
			m.searchMode = false
			m.searchQuery = ""
		case "s":
			m.searchMode = true
			m.jumpMode = false
			m.searchQuery = ""
		case "esc":
			m.searchMode = false
			m.jumpMode = false
			m.searchQuery = ""
		case "enter":
			if m.jumpMode {
				if lineNum, err := strconv.Atoi(m.searchQuery); err == nil {
					// Convert from 1-based (user input) to 0-based (internal index)
					targetIdx := lineNum - 1
					if targetIdx >= 0 && targetIdx < len(m.logs) {
						m.selectedLogIndex = targetIdx
					}
				}
				m.jumpMode = false
				m.searchQuery = ""
			} else if m.searchMode {
				m.filteredLogs = filterLogs(m.logs, m.searchQuery)
				if len(m.filteredLogs) > 0 {
					m.selectedLogIndex = 0
				}
				m.searchMode = false
				m.searchQuery = ""
			}
		default:
			if m.searchMode || m.jumpMode {
				if msg.Type == tea.KeyBackspace && len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				} else if msg.Type == tea.KeyRunes {
					m.searchQuery += string(msg.Runes)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func renderRawLog(log ParsedLog, width, height int) string {
	rawStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(normalColor).
		Padding(0, 1).
		Width(width - 2).
		Height(height).
		BorderTop(false).
		BorderBottom(true)

	var builder strings.Builder
	builder.WriteString(headerStyle.Render("Raw Log") + "\n")

	// Parse the JSON first to ensure it's valid
	var parsedJSON interface{}
	if err := json.Unmarshal([]byte(log.RawLog), &parsedJSON); err == nil {
		// Re-marshal with indentation
		prettyJSON, err := json.MarshalIndent(parsedJSON, "", "  ")
		if err == nil {
			builder.WriteString(jsonStringStyle.Render(string(prettyJSON)))
		} else {
			builder.WriteString(jsonStringStyle.Render(log.RawLog))
		}
	} else {
		builder.WriteString(jsonStringStyle.Render(log.RawLog))
	}

	return rawStyle.Render(builder.String())
}

func (m Model) View() string {
	if len(m.filteredLogs) == 0 {
		return errorStyle.Render("No valid logs found. Press 'q' to quit.")
	}

	header := headerStyle.Render(fmt.Sprintf(
		"Log %d of %d | Press 's' to search, '/' to jump, 'q' to quit",
		m.selectedLogIndex+1,
		len(m.filteredLogs),
	))

	// Calculate heights - top section should be smaller since it's just a list
	mainHeight := m.height - 4 // Reserve space for header
	if mainHeight < 0 {
		mainHeight = 0
	}

	// Allocate heights: 20% list (it's compact), 10% raw log (single line), 70% details
	listHeight := (mainHeight * 20) / 100
	rawLogHeight := 3 // Fixed height for single line + borders
	detailHeight := mainHeight - listHeight - rawLogHeight

	// Ensure minimum heights
	if listHeight < 5 {
		listHeight = 5
	}
	if detailHeight < 10 {
		detailHeight = 10
	}

	logList := renderLogList(m.filteredLogs, m.selectedLogIndex, m.width, listHeight)
	rawLog := renderRawLog(m.filteredLogs[m.selectedLogIndex], m.width, m.height)
	detailView := renderDetailView(m.filteredLogs[m.selectedLogIndex], m.width, m.height)

	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		logList,
		rawLog,
		detailView,
	)

	if m.searchMode || m.jumpMode {
		mode := "Search"
		if m.jumpMode {
			mode = "Jump to line"
		}
		overlay := searchStyle.Render(fmt.Sprintf("%s: %s", mode, m.searchQuery))
		return lipgloss.JoinVertical(lipgloss.Left, header, mainContent, overlay)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, mainContent)
}

func renderLogList(logs []ParsedLog, selectedIdx, width, height int) string {
	if len(logs) == 0 {
		return ""
	}

	var builder strings.Builder
	listStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(normalColor).
		Width(width - 2).
		Height(height).
		BorderBottom(true)

	builder.WriteString(headerStyle.Render("Log List (use ↑↓ to navigate)") + "\n")

	// Calculate available lines for logs
	availableLines := height - 4 // Account for border, title, and padding
	if availableLines < 0 {
		availableLines = 0
	}

	// Calculate visible range
	startIdx := selectedIdx - (availableLines / 2)
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + availableLines
	if endIdx > len(logs) {
		endIdx = len(logs)
		startIdx = endIdx - availableLines
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Render logs
	for i := startIdx; i < endIdx && i < len(logs); i++ {
		log := logs[i]

		// Format line number and cursor
		cursor := "  "
		if i == selectedIdx {
			cursor = "▶ "
		}
		lineNum := fmt.Sprintf("%s%3d:", cursor, log.LineNumber)

		// Format preview with max width
		preview := formatLogPreview(log, width-len(lineNum)-6)
		line := fmt.Sprintf("%s %s", lineNum, preview)

		style := logStyle
		if i == selectedIdx {
			style = selectedLogStyle
		}

		if flags, ok := log.Fields["response_flags"].(string); ok {
			switch {
			case strings.Contains(flags, "UF"), strings.Contains(flags, "URX"):
				style = style.Copy().Foreground(errorColor)
			case strings.Contains(flags, "UH"), strings.Contains(flags, "UO"):
				style = style.Copy().Foreground(warnColor)
			}
		}

		builder.WriteString(style.Render(line) + "\n")
	}

	return listStyle.Render(builder.String())
}

func formatLogPreview(log ParsedLog, maxWidth int) string {
	var parts []string

	// Always try to get and format timestamp first
	if startTime, ok := log.Fields["start_time"].(string); ok && startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			parts = append(parts, t.Format("15:04:05"))
		}
	}

	// Add response code
	if code, ok := log.Fields["response_code"].(float64); ok {
		parts = append(parts, fmt.Sprintf("[%d]", int(code)))
	}

	// Add response flags
	if flags, ok := log.Fields["response_flags"].(string); ok && flags != "" {
		parts = append(parts, flags)
	}

	// Add method and path if available
	if method, ok := log.Fields["method"].(string); ok && method != "" && method != "null" {
		parts = append(parts, method)
	}
	if path, ok := log.Fields["path"].(string); ok && path != "" && path != "null" {
		parts = append(parts, path)
	}

	// Format the preview
	preview := strings.Join(parts, " ")
	if len(preview) == 0 {
		// If we couldn't create a formatted preview, truncate the raw log
		preview = truncate(log.RawLog, maxWidth)
	}

	return truncate(preview, maxWidth)
}

func getFieldSafely(fields map[string]interface{}, key string) string {
	if value, exists := fields[key]; exists {
		if value == nil {
			return "-"
		}
		str := fmt.Sprintf("%v", value)
		if str == "" || str == "null" {
			return "-"
		}
		return str
	}
	return "-"
}

func renderDetailView(log ParsedLog, width, height int) string {
	if width <= 0 {
		fmt.Print("Width is 0, bailing out of renderDetailView early")
		return ""
	}
	if height <= 0 {
		fmt.Print("Height is 0, bailing out of renderDetailView early")
		return ""
	}
	detailStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(normalColor).
		Padding(0, 1).
		Width(width - 2).
		Height(height). // Does not include border
		BorderTop(false)

	var builder strings.Builder
	builder.WriteString(headerStyle.Render("Parsed Log Details") + "\n\n")

	// Group and format fields with improved error handling
	groups := []struct {
		name   string
		fields []string
	}{
		{"Request Info", []string{
			"start_time", "method", "protocol", "authority", "path",
			"request_id", "user_agent", "client_ip", "x_forwarded_for",
		}},
		{"Response Info", []string{
			"response_code", "response_code_details", "response_flags",
			"duration", "bytes_sent", "bytes_received",
		}},
		{"Upstream Info", []string{
			"upstream_cluster", "upstream_host", "upstream_local_address",
			"upstream_service_time", "upstream_transport_failure_reason",
		}},
		{"Downstream Info", []string{
			"downstream_local_address", "downstream_remote_address",
			"requested_server_name", "route_name",
		}},
	}

	for _, group := range groups {
		builder.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			Render(group.name) + "\n")

		hasData := false
		for _, field := range group.fields {
			value := getFieldSafely(log.Fields, field)
			if value != "-" {
				hasData = true
			}
			fieldStr := jsonKeyStyle.Render(fmt.Sprintf("%-30s", field))
			valueStr := formatFieldValue(field, value)
			builder.WriteString(fmt.Sprintf("%s: %s\n", fieldStr, valueStr))
		}

		if !hasData {
			builder.WriteString(jsonNullStyle.Render("No data available\n"))
		}
		builder.WriteString("\n")
	}

	return detailStyle.Render(builder.String())
}

func formatFieldValue(field, value string) string {
	if value == "-" {
		return jsonNullStyle.Render("-")
	}

	valueStr := value
	var explanation string

	switch field {
	case "response_flags":
		explanation = explainResponseFlags(value)
	case "response_code":
		explanation = getResponseCodeExplanation(value)
	case "upstream_transport_failure_reason":
		explanation = getFailureExplanation(value)
	case "duration":
		if value == "0" {
			explanation = "request did not complete"
		}
	case "downstream_local_address", "downstream_remote_address", "upstream_host":
		explanation = formatAddress(value)
	}

	if explanation != "" {
		valueStr = fmt.Sprintf("%s %s",
			jsonStringStyle.Render(value),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("242")).
				Italic(true).
				Render(fmt.Sprintf("(%s)", explanation)))
	} else {
		valueStr = jsonStringStyle.Render(value)
	}

	return valueStr
}

func getResponseCodeExplanation(code string) string {
	switch code {
	case "200":
		return "OK"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "403":
		return "Forbidden"
	case "404":
		return "Not Found"
	case "500":
		return "Internal Server Error"
	case "502":
		return "Bad Gateway"
	case "503":
		return "Service Unavailable"
	case "504":
		return "Gateway Timeout"
	case "0":
		return "no response (connection failed)"
	}
	return ""
}

func getFailureExplanation(reason string) string {
	if strings.Contains(reason, "delayed_connect_error") {
		return "connection to upstream service failed"
	}
	return ""
}

func formatAddress(value string) string {
	parts := strings.Split(value, ":")
	if len(parts) == 2 {
		return fmt.Sprintf("IP: %s, Port: %s", parts[0], parts[1])
	}
	return ""
}

func explainResponseFlags(flags string) string {
	explanations := []string{}

	flagMap := map[string]string{
		"UH":   "upstream unhealthy",
		"UF":   "upstream connection failure",
		"UO":   "upstream overflow",
		"NR":   "no route configured",
		"URX":  "upstream request timeout",
		"DC":   "downstream connection termination",
		"LH":   "local service healthy",
		"UR":   "upstream retry",
		"UC":   "upstream connection termination",
		"DT":   "downstream request timeout",
		"LR":   "local service rejected",
		"RL":   "rate limited",
		"UAEX": "unauthorized external service",
		"RLSE": "rate limited service error",
		"IH":   "invalid HTTP response",
		"SI":   "stream idle timeout",
		"DPE":  "downstream protocol error",
		"UPE":  "upstream protocol error",
		"NC":   "no cluster found",
	}

	parts := strings.Split(flags, ",")
	for _, flag := range parts {
		if explanation, exists := flagMap[strings.TrimSpace(flag)]; exists {
			explanations = append(explanations, explanation)
		}
	}

	if len(explanations) > 0 {
		return strings.Join(explanations, ", ")
	}
	return ""
}

func truncate(input string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	inputRunes := []rune(input)
	if len(inputRunes) <= maxLen {
		return input
	}

	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}

	return string(inputRunes[:maxLen-3]) + "..."
}
