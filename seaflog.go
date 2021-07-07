// Package seaflog provides tools to process SeaFlow V1 instrument log files.
package seaflog

import (
	"bufio"
	_ "embed" // for event definition JSON
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ctberthiaume/tsdata"
)

var Version string = "v0.1.3"

// Log is seaflog's logger
var Log *log.Logger

// Quiet turns off logging
func Quiet(on bool) {
	if on {
		Log.SetOutput(io.Discard)
	} else {
		Log.SetOutput(os.Stderr)
	}
}

// init reads
func init() {
	// event definitions from JSON
	result := struct {
		Events []EventDef
	}{}
	if err := json.Unmarshal([]byte(eventDefsJSON), &result); err != nil {
		panic(err)
	}
	EventDefs = make(map[string]EventDef)
	for _, edef := range result.Events {
		EventDefs[edef.Name] = edef
	}

	// Configure logger
	Log = log.New(
		os.Stderr,
		"-------------------------------------------------------------------------------\n",
		0,
	)
}

//go:embed event_definitions.json
var eventDefsJSON string

// EventDefs hold event defintions keyed by name.
var EventDefs map[string]EventDef

// EventDef defines a log file event
type EventDef struct {
	Name       string
	Type       string
	EventForms []EventForm `json:"forms"`
}

// EventForm defines a form of an event with a unique line prefix.
type EventForm struct {
	StartsWith  string `json:"startswith"`
	ValueAction string `json:"value_action"`
	Examples    []EventExample
}

// EventExample contains example input and parsed data for an Event.
type EventExample struct {
	Text   string
	Parsed Event
}

// Event is parsed log file event
type Event struct {
	Name       string
	Type       string
	Line       string
	Value      interface{}
	Time       time.Time
	LineNumber int `json:"line_number"`
	Error      error
}

// EventScanner provides an interface for reading through a SeaFlow v1 instrument log file.
type EventScanner struct {
	scanner *bufio.Scanner
	t       time.Time // time for last seen timestamp line
	i       int       // current line number, starting at 1
	event   Event
	error   error
	done    bool
}

func NewEventScanner(r io.Reader) *EventScanner {
	return &EventScanner{scanner: bufio.NewScanner(r)}
}

// Scan advances to the next event, which will then be available through the
// Event method. Returns false when the end of the input has been reached or
// after encountering an unrevorable error. This error which will be available
// with the Err method.
func (es *EventScanner) Scan() bool {
	if es.done {
		return false
	}

	for es.scanner.Scan() {
		es.i++
		line := es.scanner.Text()
		tnew, err := parseTimestamp(line)
		if err == nil {
			// New timestamp line
			es.t = tnew
		} else {
			// Event data line
			if line == "" || line == "Fault:" {
				// A lot of these, just skip
				continue
			}
			event, err := CreateEvent(line, es.t, es.i)
			if err != nil {
				es.error = err
				return false
			}
			es.event = event
			return true
		}
	}
	es.done = true

	if err := es.scanner.Err(); err != nil {
		es.error = err
	}
	return false
}

func (es *EventScanner) Event() Event {
	return es.event
}

// Err returns any unrecoverable error encountered during event scanning.
func (es *EventScanner) Err() error {
	return es.error
}

// CreateEvent creates an event
func CreateEvent(line string, t time.Time, lineNumber int) (event Event, err error) {
	event = Event{Time: t, Line: line, LineNumber: lineNumber}

	// Handle case where this event occurs before any timestamp
	if event.Time.IsZero() {
		event.Error = fmt.Errorf("event with no time set")
		return event, nil
	}

	// Parse the line
	for _, edef := range EventDefs {
		for _, eform := range edef.EventForms {
			if strings.HasPrefix(line, eform.StartsWith) {
				event.Name = edef.Name
				event.Type = edef.Type
				switch valueAction := eform.ValueAction; valueAction {
				case "as_float":
					parts := strings.SplitN(line, ":", 2)
					if len(parts) < 2 {
						event.Error = fmt.Errorf("missing expected separator ':'")
					} else {
						if f, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err != nil {
							event.Error = err
						} else {
							event.Value = f
						}
					}
				case "as_text":
					parts := strings.SplitN(line, ":", 2)
					if len(parts) < 2 {
						event.Error = fmt.Errorf("missing expected separator ':'")
					} else {
						event.Value = strings.TrimSpace(parts[1])
					}
				case "as_true":
					event.Value = true
				case "as_false":
					event.Value = false
				case "as_identity":
					event.Value = line
				default:
					// Should never happen
					return event, fmt.Errorf("invalid ValueAction in event defintiion: %v", valueAction)
				}
				return event, nil
			}
		}
	}

	// No prefix matched, mark as unhandled
	event.Error = fmt.Errorf("unrecognized event")
	event.Name = "unhandled"
	event.Type = "text"
	event.Value = event.Line

	return event, nil
}

// Match log file timestamp, e.g. "2015-03-14T00-26-52+00-00"
var timeExpr = regexp.MustCompile(
	`^(?P<date>\d{4}-\d{2}-\d{2})T(?P<h>\d{2})-(?P<m>\d{2})-(?P<s>\d{2})(?P<tzh>[+-]\d{2})-(?P<tzm>\d{2})$`,
)

// parseTimestamp converts a SeaFlow timestamp to a time.Time struct
func parseTimestamp(text string) (t time.Time, err error) {
	tstamp := timeExpr.ReplaceAllString(text, "${date}T${h}:${m}:${s}${tzh}:${tzm}")
	t, err = time.Parse(time.RFC3339, tstamp)
	if err != nil || tstamp == text {
		// Check tstamp == text in case we hit a data line that happens to be an
		// RFC3339 timestamp
		return time.Time{}, err
	}
	return t, nil
}

// TimeFilter returns true if an Event lies inclusively within the bounds of the
// times earliest and latest, and false otherwise. If earliest or latest are
// zero times they will be ignored.
func TimeFilter(event Event, earliest, latest time.Time) bool {
	afterEarliest := (earliest.IsZero() || (event.Time.After(earliest) || event.Time.Equal(earliest)))
	beforeLatest := (latest.IsZero() || (event.Time.Before(latest) || event.Time.Equal(latest)))
	return afterEarliest && beforeLatest
}

// UnhandledToNote converts an unhandled event to a note event
func UnhandledToNote(unhandled Event) Event {
	return Event{
		Name:       "note",
		Type:       "text",
		Value:      unhandled.Line,
		Line:       unhandled.Line,
		LineNumber: unhandled.LineNumber,
		Time:       unhandled.Time,
	}
}

// TsdataWriter provides tools to write SeaFlow log files in TSDATA file format
type TsdataWriter struct {
	tsdata tsdata.Tsdata
	coli   map[string]int // column index by column name
}

// NewTsdataWriter creates a new TsdataWriter struct
func NewTsdataWriter(fileType string, project string, description string) TsdataWriter {
	t := TsdataWriter{
		tsdata: tsdata.Tsdata{
			FileType:        fileType,
			Project:         project,
			FileDescription: description,
		},
	}
	// Get event names in unique, sorted order
	keys := make([]string, len(EventDefs))
	i := 0
	for name := range EventDefs {
		keys[i] = name
		i++
	}
	sort.Strings(keys)
	// Prepend "time"
	columns := make([]string, len(keys)+1)
	columns[0] = "time"
	for i, k := range keys {
		columns[i+1] = k
	}
	// Populate header fields
	t.tsdata.Headers = columns
	t.tsdata.Types = make([]string, len(columns))
	t.tsdata.Comments = make([]string, len(columns))
	t.tsdata.Units = make([]string, len(columns))

	t.coli = make(map[string]int) // column indexes by name
	for i, column := range columns {
		if i == 0 {
			t.tsdata.Comments[i] = "ISO8601 timestamp"
			t.tsdata.Types[i] = "time"
			t.tsdata.Units[i] = tsdata.NA
			t.coli["time"] = i
		} else {
			t.tsdata.Comments[i] = tsdata.NA
			edef, ok := EventDefs[column]
			if !ok {
				panic(fmt.Errorf("Event definition for %v not found", column))
			}
			t.tsdata.Types[i] = edef.Type
			t.tsdata.Units[i] = tsdata.NA
			t.coli[column] = i
		}
	}

	// Final check that everything looks good
	err := t.tsdata.ValidateMetadata()
	if err != nil {
		panic(err)
	}

	return t
}

// HeaderText returns a TSDATA header string
func (t TsdataWriter) HeaderText() string {
	return t.tsdata.Header()
}

// EventText returns a TSDATA event line string for one Event
func (t TsdataWriter) EventText(event Event) (string, error) {
	if event.Error != nil {
		return "", nil
	}

	outs := make([]string, len(t.tsdata.Headers))
	outs[0] = event.Time.Format("2006-01-02T15:04:05-07:00") // RFC3339 with numeric time zone
	for i := 1; i < len(outs); i++ {
		outs[i] = tsdata.NA
	}

	if i, ok := t.coli[event.Name]; ok {
		if t.tsdata.Types[i] == "boolean" {
			boolVal, ok := event.Value.(bool)
			if !ok {
				return "", fmt.Errorf("bad boolean value for column %q %q, line %d", event.Name, i, event.LineNumber)
			}
			if boolVal {
				outs[i] = "TRUE"
			} else {
				outs[i] = "FALSE"
			}
		} else if t.tsdata.Types[i] == "text" {
			// Replace tsdata.Delim with spaces
			outs[i] = strings.ReplaceAll(fmt.Sprintf("%v", event.Value), tsdata.Delim, " ")
		} else {
			outs[i] = fmt.Sprintf("%v", event.Value)
		}
	} else {
		return "", fmt.Errorf("TSDATA column index for event named '%s' not found", event.Name)
	}

	return strings.Join(outs, tsdata.Delim), nil
}
