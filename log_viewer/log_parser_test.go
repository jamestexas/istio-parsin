// log_viewer/log_parser_test.go
package main

import (
	"testing"
)

func TestParseRawLogs(t *testing.T) {
	singleLog := `{"level":"info","message":"Server started","timestamp":"2024-11-25T12:34:56Z"}`
	multipleLogs := []string{
		`{"level":"info","message":"Server started","timestamp":"2024-11-25T12:34:56Z"}`,
		`{"level":"error","message":"Connection failed","timestamp":"2024-11-25T12:35:00Z"}`,
	}
	jsonArrayLogs := `[{"level":"info","message":"Server started","timestamp":"2024-11-25T12:34:56Z"}, {"level":"error","message":"Connection failed","timestamp":"2024-11-25T12:35:00Z"}]`

	tests := []struct {
		name     string
		rawLogs  []string
		expected []ParsedLog
		wantErr  bool
	}{
		{
			name: "Single JSON log entry",
			rawLogs: []string{
				singleLog,
			},
			expected: []ParsedLog{
				{
					RawLog: singleLog,
					Fields: map[string]interface{}{
						"level":     "info",
						"message":   "Server started",
						"timestamp": "2024-11-25T12:34:56Z",
					},
					LineNumber: 1,
				},
			},
			wantErr: false,
		},
		{
			name:    "Multiple JSON log entries",
			rawLogs: multipleLogs,
			expected: []ParsedLog{
				{
					RawLog: multipleLogs[0],
					Fields: map[string]interface{}{
						"level":     "info",
						"message":   "Server started",
						"timestamp": "2024-11-25T12:34:56Z",
					},
					LineNumber: 1,
				},
				{
					RawLog: multipleLogs[1],
					Fields: map[string]interface{}{
						"level":     "error",
						"message":   "Connection failed",
						"timestamp": "2024-11-25T12:35:00Z",
					},
					LineNumber: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "JSON array of log entries",
			rawLogs: []string{
				jsonArrayLogs,
			},
			expected: []ParsedLog{
				{
					RawLog: `{"level":"info","message":"Server started","timestamp":"2024-11-25T12:34:56Z"}`,
					Fields: map[string]interface{}{
						"level":     "info",
						"message":   "Server started",
						"timestamp": "2024-11-25T12:34:56Z",
					},
					LineNumber: 1,
				},
				{
					RawLog: `{"level":"error","message":"Connection failed","timestamp":"2024-11-25T12:35:00Z"}`,
					Fields: map[string]interface{}{
						"level":     "error",
						"message":   "Connection failed",
						"timestamp": "2024-11-25T12:35:00Z",
					},
					LineNumber: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid JSON log entry",
			rawLogs: []string{
				singleLog,
				`invalid json`,
			},
			expected: []ParsedLog{
				{
					RawLog: singleLog,
					Fields: map[string]interface{}{
						"level":     "info",
						"message":   "Server started",
						"timestamp": "2024-11-25T12:34:56Z",
					},
					LineNumber: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "Empty input",
			rawLogs: []string{
				``,
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRawLogs(tt.rawLogs)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRawLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equal(got, tt.expected) {
				t.Errorf("parseRawLogs() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func equal(a, b []ParsedLog) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].RawLog != b[i].RawLog || a[i].LineNumber != b[i].LineNumber {
			return false
		}
		if !equalFields(a[i].Fields, b[i].Fields) {
			return false
		}
	}
	return true
}

func equalFields(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
