package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func compileForDiff(t *testing.T, source string) *CompiledSchema {
	t.Helper()
	result := ParseWithDiagnostics(source)
	if !result.Valid {
		t.Fatalf("expected valid schema, got: %v", result.Diagnostics)
	}
	compiled, err := Compile(result.Schema, "Post", "post", false)
	if err != nil {
		t.Fatalf("Compile() error: %v", err)
	}
	return compiled
}

func TestSchemaDiff_ToJSON(t *testing.T) {
	from := compileForDiff(t, `type Post { title: String }`)
	to := compileForDiff(t, `type Post { title: String! }`)

	diff := DiffSchemas(from, to)
	data, err := diff.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var roundTripped SchemaDiff
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("failed to unmarshal ToJSON output: %v", err)
	}
	if len(roundTripped.Changes) != len(diff.Changes) {
		t.Errorf("expected %d changes after round trip, got %d", len(diff.Changes), len(roundTripped.Changes))
	}
}

func TestSchemaDiff_Summary(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		cs := compileForDiff(t, `type Post { title: String! }`)
		diff := DiffSchemas(cs, cs)
		if got := diff.Summary(); got != "No changes detected" {
			t.Errorf("got %q, want %q", got, "No changes detected")
		}
	})

	t.Run("mixed changes summarized", func(t *testing.T) {
		from := compileForDiff(t, `type Post { title: String body: Text }`)
		to := compileForDiff(t, `type Post { title: String! sub: String }`)

		diff := DiffSchemas(from, to)
		summary := diff.Summary()
		if !strings.Contains(summary, "change(s)") {
			t.Errorf("expected summary to mention change count, got: %q", summary)
		}
	})
}

func TestToFloat64Value(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  float64
		ok    bool
	}{
		{"float64", float64(3.5), 3.5, true},
		{"int64", int64(4), 4, true},
		{"int", 5, 5, true},
		{"string unsupported", "nope", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toFloat64Value(tc.input)
			if ok != tc.ok || got != tc.want {
				t.Errorf("toFloat64Value(%v) = (%v, %v), want (%v, %v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestToInt64Value_AllBranches(t *testing.T) {
	cases := []struct {
		name  string
		input any
		want  int64
		ok    bool
	}{
		{"int64", int64(7), 7, true},
		{"int", 8, 8, true},
		{"float64", float64(9.9), 9, true},
		{"unsupported", true, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toInt64Value(tc.input)
			if ok != tc.ok || got != tc.want {
				t.Errorf("toInt64Value(%v) = (%v, %v), want (%v, %v)", tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestIsBreakingDecoratorChange_MoreBranches(t *testing.T) {
	cases := []struct {
		name     string
		decName  string
		from, to any
		want     bool
	}{
		{"maxLength decreased is breaking", DecMaxLength, int64(200), int64(100), true},
		{"maxLength increased is safe", DecMaxLength, int64(100), int64(200), false},
		{"minLength increased is breaking", DecMinLength, int64(1), int64(5), true},
		{"max decreased is breaking", DecMax, float64(100), float64(50), true},
		{"min increased is breaking", DecMin, float64(0), float64(10), true},
		{"maxItems decreased is breaking", DecMaxItems, int64(10), int64(5), true},
		{"minItems increased is breaking", DecMinItems, int64(1), int64(3), true},
		{"unknown decorator is never breaking", "customDecorator", 1, 2, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isBreakingDecoratorChange(tc.decName, tc.from, tc.to)
			if got != tc.want {
				t.Errorf("isBreakingDecoratorChange(%s, %v, %v) = %v, want %v", tc.decName, tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestIsBreakingDecoratorRemoval_AllBranches(t *testing.T) {
	if isBreakingDecoratorRemoval(DecRequired) {
		t.Error("removing @required should not be flagged as breaking")
	}
	if isBreakingDecoratorRemoval(DecUnique) {
		t.Error("removing @unique should not be flagged as breaking")
	}
	if isBreakingDecoratorRemoval("somethingElse") {
		t.Error("removing an unrecognized decorator should not be flagged as breaking")
	}
}

func TestIsBreakingDecoratorAddition_AllBranches(t *testing.T) {
	trueCases := []string{DecRequired, DecMaxLength, DecMinLength, DecMin, DecMax, DecPattern, DecMinItems, DecMaxItems}
	for _, name := range trueCases {
		if !isBreakingDecoratorAddition(name, nil) {
			t.Errorf("expected adding @%s to be breaking", name)
		}
	}
	if isBreakingDecoratorAddition("unknownDecorator", nil) {
		t.Error("expected adding an unrecognized decorator to be non-breaking")
	}
}
