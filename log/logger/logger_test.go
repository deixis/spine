package logger

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/deixis/spine/log"
	fjson "github.com/deixis/spine/log/formatter/json"
)

func TestFields(t *testing.T) {
	f := &fjson.Formatter{}
	p := newMockPrinter()

	logger := Build("test", log.LevelTrace, f, p)

	// ensure that fields provided as arguments are added to the log
	// line
	expectedFields := []log.Field{
		log.String("key", "value"),
	}
	logger.Trace("my.func", "something happened", expectedFields...)
	if n := p.NumLines(); n != 1 {
		t.Fatalf("expected printer to have output %d lines, got %d",
			1, n)
	}
	checkFields(t, expectedFields, p)

	// ensure that fields added to the logger using `With` are added
	// to the log line
	addedFields := []log.Field{
		log.String("new", "field"),
	}
	logger2 := logger.With(addedFields...)
	logger2.Trace("my.func", "something happened", expectedFields...)
	if n := p.NumLines(); n != 2 {
		t.Fatalf("expected printer to have output %d lines, got %d",
			2, n)
	}
	allFields := append(expectedFields, addedFields...)
	checkFields(t, allFields, p)
}

func checkFields(
	t *testing.T,
	expectedFields []log.Field,
	p *mockPrinter,
) {
	output, err := p.LastJSON()
	if err != nil {
		t.Fatal(err)
	}

	fields, ok := output["fields"]
	if !ok {
		t.Fatalf("expected \"fields\" to exist")
	}

	m, ok := fields.(map[string]interface{})
	if !ok {
		t.Fatal()
	}

	for _, field := range expectedFields {
		k, v := field.KV()
		gotV, ok := m[k]
		if !ok {
			t.Fatalf("expected key \"%s\" to exist", k)
		}
		if gotV != v {
			t.Fatalf("expected key \"%s\" to have value \"%s\", got \"%s\"", k, v, gotV)
		}
	}
}

type mockPrinter struct {
	mu    sync.RWMutex
	lines []string
}

func newMockPrinter() *mockPrinter {
	return &mockPrinter{
		lines: make([]string, 0),
	}
}

func (m *mockPrinter) Close() error {
	return nil
}

func (m *mockPrinter) Print(ctx *log.Context, s string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lines = append(m.lines, s)

	return nil
}

func (m *mockPrinter) NumLines() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.lines)
}

func (m *mockPrinter) Last() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.lines) == 0 {
		return "", false
	}

	return m.lines[len(m.lines)-1], true
}

func (m *mockPrinter) LastJSON() (map[string]interface{}, error) {
	line, ok := m.Last()
	if !ok {
		return nil, fmt.Errorf("no lines have been added")
	}

	var out map[string]interface{}
	err := json.Unmarshal([]byte(line), &out)

	return out, err
}
