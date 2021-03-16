package seaflog_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/seaflow-uw/seaflog"
)

type eventTestData struct {
	name  string
	input string
	want  seaflog.Event
}

func TestKnownGoodEventParsing(t *testing.T) {

	tests := []eventTestData{}

	for name, edef := range seaflog.EventDefs {
		for _, eform := range edef.EventForms {
			i := 1
			for _, ex := range eform.Examples {
				tests = append(tests, eventTestData{
					name:  fmt.Sprintf("%s_%s_%d", name, eform.StartsWith, i),
					input: ex.Text,
					want:  ex.Parsed,
				})
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := seaflog.NewEventScanner(strings.NewReader(tt.input))
			for scanner.Scan() {
				got := scanner.Event()
				eventsEqual(got, tt.want, t)
			}
			if err := scanner.Err(); err != nil {
				t.Errorf("EventScanner error = %v; want nil", err)
			}
		})
	}
}

func TestUnknownEventParsing(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2015-03-14T00:26:52+00:00")
	tests := []eventTestData{}
	tests = append(tests, eventTestData{
		name:  "unknown event",
		input: "2015-03-14T00-26-52+00-00\nnot a real event data line\n",
		want: seaflog.Event{
			Name:       "unhandled",
			Type:       "text",
			Value:      "not a real event data line",
			Line:       "not a real event data line",
			LineNumber: 2,
			Time:       t0,
			Error:      fmt.Errorf("unrecognized event"),
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := seaflog.NewEventScanner(strings.NewReader(tt.input))
			for scanner.Scan() {
				got := scanner.Event()
				eventsEqual(got, tt.want, t)
			}
			if err := scanner.Err(); err != nil {
				t.Errorf("EventScanner error = %v; want nil", err)
			}
		})
	}
}

func TestBadFloatParsing(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2015-03-14T00:26:52+00:00")
	tests := []eventTestData{}
	tests = append(tests, eventTestData{
		name:  "bad float",
		input: "2015-03-14T00-26-52+00-00\nPMT1:1.a06\n",
		want: seaflog.Event{
			Name:       "PMT1",
			Type:       "float",
			Line:       "PMT1:1.a06",
			LineNumber: 2,
			Time:       t0,
			Error:      fmt.Errorf("placeholder error"),
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := seaflog.NewEventScanner(strings.NewReader(tt.input))
			for scanner.Scan() {
				got := scanner.Event()
				eventsEqual(got, tt.want, t)
			}
			if err := scanner.Err(); err != nil {
				t.Errorf("EventScanner error = %v; want nil", err)
			}
		})
	}
}

func TestUnhandledToNote(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2015-03-14T00:26:52+00:00")
	input := seaflog.Event{
		Name:       "unhandled",
		Type:       "text",
		Value:      "not a real event data line",
		Line:       "not a real event data line",
		LineNumber: 2,
		Time:       t0,
		Error:      fmt.Errorf("unrecognized event"),
	}
	want := seaflog.Event{
		Name:       "note",
		Type:       "text",
		Value:      "not a real event data line",
		Line:       "not a real event data line",
		LineNumber: 2,
		Time:       t0,
	}

	t.Run("unhandled to note", func(t *testing.T) {
		got := seaflog.UnhandledToNote(input)
		eventsEqual(got, want, t)
	})
}

func TestTimeFilter(t *testing.T) {
	stamps := []string{
		"2015-03-14T00:00:00+00:00",
		"2015-03-15T00:00:00+00:00",
		"2015-03-16T00:00:00+00:00",
		"2015-03-17T00:00:00+00:00",
	}
	events := make([]seaflog.Event, len(stamps))
	for i := range stamps {
		ti, _ := time.Parse(time.RFC3339, stamps[i])
		events[i] = seaflog.Event{Time: ti}
	}

	type timeFilterTestData struct {
		name     string
		events   []seaflog.Event
		earliest time.Time
		latest   time.Time
		want     []seaflog.Event
	}

	tests := []timeFilterTestData{
		{
			name:     "no filter",
			events:   events,
			earliest: time.Time{},
			latest:   time.Time{},
			want:     events,
		},
		{
			name:     "only earliest",
			events:   events,
			earliest: events[1].Time,
			latest:   time.Time{},
			want:     events[1:],
		},
		{
			name:     "only latest",
			events:   events,
			earliest: time.Time{},
			latest:   events[1].Time,
			want:     events[:2],
		},
		{
			name:     "both earliest and latest",
			events:   events,
			earliest: events[1].Time,
			latest:   events[3].Time,
			want:     events[1:4],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, got := []string{}, []string{}
			for _, e := range tt.events {
				if seaflog.TimeFilter(e, tt.earliest, tt.latest) {
					got = append(got, e.Time.Format(time.RFC3339))
				}
			}
			for _, w := range tt.want {
				want = append(want, w.Time.Format(time.RFC3339))
			}
			stringsEqual(got, want, t)
		})
	}
}

func eventsEqual(got, want seaflog.Event, t *testing.T) {
	if got.Name != want.Name {
		t.Errorf("Event.Name %v; want %v", got.Name, want.Name)
	}
	if got.Type != want.Type {
		t.Errorf("Event.Type %v; want %v", got.Type, want.Type)
	}
	if got.Line != want.Line {
		t.Errorf("Event.Line %v; want %v", got.Line, want.Line)
	}
	if got.Value != want.Value {
		t.Errorf("Event.Value %v; want %v", got.Value, want.Value)
	}
	if got.LineNumber != want.LineNumber {
		t.Errorf("Event.LineNumber %v; want %v", got.LineNumber, want.LineNumber)
	}
	if got.Error != nil && want.Error == nil {
		t.Errorf("Event.Error %v; want %v", got.Error, want.Error)
	}
	if got.Error == nil && want.Error != nil {
		// Don't care what error the test defines, just want an error
		t.Errorf("Event.Error %v; want an error", got.Error)
	}
	if !got.Time.Equal(want.Time) {
		t.Errorf("Event.Time %v; want %v", got.Time, want.Time)
	}
}

func stringsEqual(got, want []string, t *testing.T) {
	if len(got) != len(want) {
		t.Fatalf("len(got) %v: len(want) %v", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got[i] %v; want[i] %v", got[i], want[i])
		}
	}
}
