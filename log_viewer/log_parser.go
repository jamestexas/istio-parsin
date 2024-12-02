// log_viewer/log_parser.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParsedLog represents a single log entry.
type ParsedLog struct {
	RawLog     string                 // Full JSON log string
	Fields     map[string]interface{} // Parsed fields
	LineNumber int                    // Original line number
}

// ParseLog parses a single log line into a ParsedLog struct.
func ParseLog(line string, lineNumber int) (ParsedLog, error) {
	var fields map[string]interface{}
	err := json.Unmarshal([]byte(line), &fields)
	if err != nil {
		return ParsedLog{}, fmt.Errorf("error parsing log line %d: %v", lineNumber, err)
	}
	return ParsedLog{
		RawLog:     line,
		Fields:     fields,
		LineNumber: lineNumber,
	}, nil
}

// parseRawLogs processes raw log lines into a slice of ParsedLog structs.
func parseRawLogs(rawLogs []string) ([]ParsedLog, error) {
	var parsedLogs []ParsedLog

	// Try to parse the entire input as a JSON array
	var logsArray []map[string]interface{}
	rawInput := strings.Join(rawLogs, "\n")
	if err := json.Unmarshal([]byte(rawInput), &logsArray); err == nil {
		for i, log := range logsArray {
			rawLog, err := json.Marshal(log)
			if err != nil {
				return nil, fmt.Errorf("error marshalling log entry %d: %v", i+1, err)
			}
			parsedLogs = append(parsedLogs, ParsedLog{
				RawLog:     string(rawLog),
				Fields:     log,
				LineNumber: i + 1,
			})
		}
		return parsedLogs, nil
	}

	// Fall back to parsing each line individually
	for i, line := range rawLogs {
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			parsedLog, err := ParseLog(line, i+1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing log line %d: %v\n", i+1, err)
				continue
			}
			parsedLogs = append(parsedLogs, parsedLog)
		} else {
			fmt.Fprintf(os.Stderr, "Skipping non-JSON log line %d: %s\n", i+1, line)
		}
	}

	if len(parsedLogs) == 0 {
		return nil, fmt.Errorf("no valid JSON logs found")
	}

	return parsedLogs, nil
}
