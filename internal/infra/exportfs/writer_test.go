package exportfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootedWriterResolvePathPolicy(t *testing.T) {
	root := t.TempDir()
	tests := []struct{ name, root, filename, outputPath, wantBase, wantErr string }{
		{"happy filename", root, "inv.pdf", "", "inv.pdf", ""},
		{"happy output path", root, "", "nested/inv.pdf", filepath.Join("nested", "inv.pdf"), ""},
		{"absolute rejected", root, "", "/etc/passwd.pdf", "", "output path must be relative"},
		{"traversal rejected", root, "", "../../etc/x.pdf", "", "escapes export root"},
		{"filename separator rejected", root, "a/b.pdf", "", "", "filename must not contain path separators"},
		{"empty root rejected", "", "inv.pdf", "", "", "BILLAR_EXPORT_DIR not configured"},
		{"default filename", root, "", "", "invoice.pdf", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (RootedWriter{Root: tc.root}).Resolve(tc.filename, tc.outputPath)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Resolve() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			want := filepath.Join(root, tc.wantBase)
			if got != want {
				t.Fatalf("Resolve() = %q, want %q", got, want)
			}
		})
	}
}

func TestDirectWriterResolveAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.pdf")
	got, err := (DirectWriter{}).Resolve("", path)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got != path {
		t.Fatalf("Resolve() = %q, want %q", got, path)
	}

	size, err := (DirectWriter{}).Write(path, []byte("%PDF-test"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if size != 9 {
		t.Fatalf("Write() size = %d, want 9", size)
	}
	if data, _ := os.ReadFile(path); string(data) != "%PDF-test" {
		t.Fatalf("written data = %q", string(data))
	}
}

func TestDirectWriterResolveNormalizesRelativePaths(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		outputPath string
	}{
		{name: "relative output path", outputPath: filepath.Join("out", "invoice.pdf")},
		{name: "relative filename fallback", filename: "invoice.pdf"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := (DirectWriter{}).Resolve(tc.filename, tc.outputPath)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if !filepath.IsAbs(got) {
				t.Fatalf("Resolve() = %q, want absolute path", got)
			}

			target := tc.outputPath
			if target == "" {
				target = tc.filename
			}
			want, err := filepath.Abs(target)
			if err != nil {
				t.Fatalf("filepath.Abs(%q) error = %v", target, err)
			}
			if got != want {
				t.Fatalf("Resolve() = %q, want %q", got, want)
			}
		})
	}
}

func TestDirectWriterWriteMissingParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "invoice.pdf")
	_, err := (DirectWriter{}).Write(path, []byte("%PDF-test"))
	if err == nil || !strings.Contains(err.Error(), "write file") {
		t.Fatalf("Write() error = %v, want wrapped write error", err)
	}
}
