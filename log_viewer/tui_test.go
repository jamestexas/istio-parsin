// log_viewer/tui_test.go

package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdate(t *testing.T) {
	rawLogEntry := []ParsedLog{
		{RawLog: "log1"}, {RawLog: "log2"},
	}
	model := &Model{ // Use a pointer here
		logs:         rawLogEntry,
		filteredLogs: rawLogEntry,
	}

	// Test moving down
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	newModel := updatedModel.(Model) // Correctly cast to *Model
	if newModel.selectedLogIndex != 1 {
		t.Errorf("expected selectedLogIndex to be 1, got %d", newModel.selectedLogIndex)
	}

	// Test moving up
	updatedModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	newModel = updatedModel.(Model)
	if newModel.selectedLogIndex != 0 {
		t.Errorf("expected selectedLogIndex to be 0, got %d", newModel.selectedLogIndex)
	}

	// Test entering search mode
	updatedModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	newModel = updatedModel.(Model)
	if !newModel.searchMode {
		t.Errorf("expected searchMode to be true, got %v", newModel.searchMode)
	}

	// Test entering jump mode
	updatedModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	newModel = updatedModel.(Model)
	if !newModel.jumpMode {
		t.Errorf("expected jumpMode to be true, got %v", newModel.jumpMode)
	}

	// Test quitting the TUI
	_, cmd := newModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Errorf("expected command to quit the TUI, got nil")
	}
}

func TestView(t *testing.T) {
	model := &Model{ // Use a pointer here
		logs:         []ParsedLog{{RawLog: "log1"}, {RawLog: "log2"}},
		filteredLogs: []ParsedLog{{RawLog: "log1"}, {RawLog: "log2"}},
		width:        100,
		height:       40,
	}

	view := model.View()
	if view == "" {
		t.Errorf("expected non-empty view, got empty view")
	}

	// Check panel widths
	expectedLeftWidth := 48  // (100 - 4) / 2
	expectedRightWidth := 48 // 100 - 4 - 48
	if model.width != 100 || model.height != 40 {
		t.Errorf("expected width=100 and height=40, got width=%d and height=%d", model.width, model.height)
	}
	if expectedLeftWidth != 48 || expectedRightWidth != 48 {
		t.Errorf("expected left panel width=48 and right panel width=48, got left panel width=%d and right panel width=%d", expectedLeftWidth, expectedRightWidth)
	}
}

func TestWindowSize(t *testing.T) {
	model := Model{
		logs:         []ParsedLog{{RawLog: "log1"}, {RawLog: "log2"}},
		filteredLogs: []ParsedLog{{RawLog: "log1"}, {RawLog: "log2"}},
	}

	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := model.Update(msg)
	newModel := updatedModel.(Model)

	if newModel.width != 100 || newModel.height != 40 {
		t.Errorf("expected width=100 and height=40, got width=%d and height=%d", newModel.width, newModel.height)
	}
}

func TestFilterLogs(t *testing.T) {
	logs := []ParsedLog{
		{RawLog: `{"level":"info","message":"Server started"}`},
		{RawLog: `{"level":"error","message":"Connection failed"}`},
	}
	model := &Model{
		logs:         logs,
		filteredLogs: logs,
	}

	// Test filtering logs
	model.searchQuery = "error"
	model.filteredLogs = filterLogs(model.logs, model.searchQuery)
	if len(model.filteredLogs) != 1 {
		t.Errorf("expected 1 filtered log, got %d", len(model.filteredLogs))
	}
	if model.filteredLogs[0].RawLog != `{"level":"error","message":"Connection failed"}` {
		t.Errorf("expected filtered log to be 'Connection failed', got %s", model.filteredLogs[0].RawLog)
	}
}

func TestJumpToLine(t *testing.T) {
	logs := []ParsedLog{
		{RawLog: `{"level":"info","message":"Server started"}`},
		{RawLog: `{"level":"error","message":"Connection failed"}`},
	}
	model := &Model{
		logs:         logs,
		filteredLogs: logs,
		jumpMode:     true,
		searchQuery:  "2",
	}

	// Test jumping to a specific line
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel := updatedModel.(Model)
	if newModel.selectedLogIndex != 1 {
		t.Errorf("expected selectedLogIndex to be 1, got %d", newModel.selectedLogIndex)
	}
	if newModel.jumpMode {
		t.Errorf("expected jumpMode to be false, got %v", newModel.jumpMode)
	}
}

// Add this to tui_test.go

func TestRawLogDisplay(t *testing.T) {
	// Create a test log with known content
	testJSON := `{"level":"info","message":"test message"}`
	model := &Model{
		logs: []ParsedLog{{
			RawLog: testJSON,
			Fields: map[string]interface{}{
				"level":   "info",
				"message": "test message",
			},
			LineNumber: 1,
		}},
		width:  100,
		height: 40,
	}
	model.filteredLogs = model.logs

	// Get the view
	view := model.View()

	// Check that the raw log section exists
	if !strings.Contains(view, "Raw Log") {
		t.Error("Raw Log section header not found in view")
	}

	// Test with properly formatted JSON
	complexJSON := `{"level":"error","message":"Connection failed","timestamp":"2024-01-01T00:00:00Z"}`
	model.logs = []ParsedLog{{
		RawLog: complexJSON,
		Fields: map[string]interface{}{
			"level":     "error",
			"message":   "Connection failed",
			"timestamp": "2024-01-01T00:00:00Z",
		},
		LineNumber: 1,
	}}
	model.filteredLogs = model.logs

	view = model.View()

	// Check for expected JSON formatting
	expectedParts := []string{
		`"level": "error"`,
		`"message": "Connection failed"`,
		`"timestamp": "2024-01-01T00:00:00Z"`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(view, part) {
			t.Errorf("Missing expected JSON content: %s", part)
		}
	}
}

func TestViewSections(t *testing.T) {
	testLog := ParsedLog{
		RawLog: `{"level":"info","message":"test"}`,
		Fields: map[string]interface{}{
			"level":   "info",
			"message": "test",
		},
		LineNumber: 1,
	}

	model := &Model{
		logs:         []ParsedLog{testLog},
		filteredLogs: []ParsedLog{testLog},
		width:        100,
		height:       40,
	}

	view := model.View()

	// Check for all three sections
	sections := []string{
		"Log List",
		"Raw Log",
		"Parsed Log Details",
	}

	for _, section := range sections {
		if !strings.Contains(view, section) {
			t.Errorf("Missing section: %s", section)
		}
	}

	// Check section ordering
	logListPos := strings.Index(view, "Log List")
	rawLogPos := strings.Index(view, "Raw Log")
	detailsPos := strings.Index(view, "Parsed Log Details")

	if !(logListPos < rawLogPos && rawLogPos < detailsPos) {
		t.Error("Sections are not in the correct order")
	}
}
