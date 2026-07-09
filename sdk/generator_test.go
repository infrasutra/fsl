package sdk

import "testing"

func TestGeneratedSDK_GetFile(t *testing.T) {
	sdk := &GeneratedSDK{
		Language: "typescript",
		Files: map[string]string{
			"client.ts": "export const client = {}",
		},
		EntryPoint: "client.ts",
	}

	t.Run("existing file", func(t *testing.T) {
		content, ok := sdk.GetFile("client.ts")
		if !ok {
			t.Fatalf("expected client.ts to be found")
		}
		if content != "export const client = {}" {
			t.Fatalf("unexpected content: %q", content)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		content, ok := sdk.GetFile("missing.ts")
		if ok {
			t.Fatalf("expected missing.ts to not be found")
		}
		if content != "" {
			t.Fatalf("expected empty content, got %q", content)
		}
	})
}

func TestGeneratedSDK_FileList(t *testing.T) {
	t.Run("multiple files", func(t *testing.T) {
		sdk := &GeneratedSDK{
			Files: map[string]string{
				"a.ts": "a",
				"b.ts": "b",
			},
		}
		names := sdk.FileList()
		if len(names) != 2 {
			t.Fatalf("expected 2 files, got %d", len(names))
		}

		seen := map[string]bool{}
		for _, n := range names {
			seen[n] = true
		}
		if !seen["a.ts"] || !seen["b.ts"] {
			t.Fatalf("expected a.ts and b.ts in list, got %v", names)
		}
	})

	t.Run("empty sdk", func(t *testing.T) {
		sdk := &GeneratedSDK{Files: map[string]string{}}
		names := sdk.FileList()
		if len(names) != 0 {
			t.Fatalf("expected 0 files, got %d", len(names))
		}
	})
}
