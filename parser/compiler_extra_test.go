package parser

import "testing"

func compileSimplePost(t *testing.T) *CompiledSchema {
	t.Helper()
	result := ParseWithDiagnostics(`type Post { title: String! }`)
	if !result.Valid {
		t.Fatalf("expected valid schema, got: %v", result.Diagnostics)
	}
	compiled, err := Compile(result.Schema, "Post", "post", false)
	if err != nil {
		t.Fatalf("Compile() error: %v", err)
	}
	return compiled
}

func TestCompiledSchema_UpdateVersion(t *testing.T) {
	cs := compileSimplePost(t)
	originalVersion := cs.Version
	originalChecksum := cs.Checksum

	cs.UpdateVersion()

	if cs.Version != originalVersion+1 {
		t.Errorf("expected version %d, got %d", originalVersion+1, cs.Version)
	}
	if cs.Checksum != ComputeChecksum(cs) {
		t.Errorf("expected checksum to be recomputed after version bump")
	}
	_ = originalChecksum
}

func TestCompiledSchema_HasChanges(t *testing.T) {
	t.Run("nil other is always a change", func(t *testing.T) {
		cs := compileSimplePost(t)
		if !cs.HasChanges(nil) {
			t.Error("expected HasChanges(nil) to be true")
		}
	})

	t.Run("identical schemas have no changes", func(t *testing.T) {
		cs := compileSimplePost(t)
		other := compileSimplePost(t)
		if cs.HasChanges(other) {
			t.Error("expected identical schemas to report no changes")
		}
	})

	t.Run("different fields are detected as changes", func(t *testing.T) {
		cs := compileSimplePost(t)

		result := ParseWithDiagnostics(`type Post { title: String! body: RichText }`)
		if !result.Valid {
			t.Fatalf("expected valid schema, got: %v", result.Diagnostics)
		}
		other, err := Compile(result.Schema, "Post", "post", false)
		if err != nil {
			t.Fatalf("Compile() error: %v", err)
		}

		if !cs.HasChanges(other) {
			t.Error("expected differing schemas to report changes")
		}
	})
}
