package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{name: "default text", input: "", want: FormatText},
		{name: "trimmed json", input: " json ", want: FormatJSON},
		{name: "upper toon", input: "TOON", want: FormatTOON},
		{name: "rejects unknown", input: "yaml", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ParseFormat() error = nil, want non-nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFormat() error = %v", err)
			}

			if got != tc.want {
				t.Fatalf("ParseFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriteOutput(t *testing.T) {
	t.Parallel()

	type sample struct {
		Name   string `json:"name" toon:"name"`
		Status string `json:"status" toon:"status"`
	}

	tests := []struct {
		name    string
		format  Format
		result  OutputResult
		want    string
		wantErr string
	}{
		{
			name:   "writes text",
			format: FormatText,
			result: OutputResult{
				Payload: sample{Name: "billar", Status: "ok"},
				TextWriter: func(w io.Writer) error {
					_, err := w.Write([]byte("Billar Health\n"))
					return err
				},
			},
			want: "Billar Health\n",
		},
		{
			name:   "writes json",
			format: FormatJSON,
			result: OutputResult{Payload: sample{Name: "billar", Status: "ok"}},
			want:   "{\"name\":\"billar\",\"status\":\"ok\"}\n",
		},
		{
			name:   "writes toon",
			format: FormatTOON,
			result: OutputResult{Payload: sample{Name: "billar", Status: "ok"}},
			want:   "name: billar\nstatus: ok\n",
		},
		{
			name:   "falls back to generic text for structs",
			format: FormatText,
			result: OutputResult{Payload: sample{Name: "billar", Status: "ok"}},
			want:   "name: billar\nstatus: ok\n",
		},
		{
			name:   "falls back to generic text for simple scalar payloads",
			format: FormatText,
			result: OutputResult{Payload: "ok"},
			want:   "ok\n",
		},
		{
			name:    "rejects unsupported generic text payloads",
			format:  FormatText,
			result:  OutputResult{Payload: []string{"billar", "ok"}},
			wantErr: "text output fallback does not support []string",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			err := WriteOutput(&out, tc.format, tc.result)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("WriteOutput() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("WriteOutput() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("WriteOutput() error = %v", err)
			}

			if out.String() != tc.want {
				t.Fatalf("WriteOutput() output = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestNewFormatterRejectsUnknownFormat(t *testing.T) {
	t.Parallel()

	_, err := NewFormatter(Format("yaml"))
	if err == nil {
		t.Fatal("NewFormatter() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Fatalf("NewFormatter() error = %q, want unsupported output format", err.Error())
	}
}

func TestWriteOutputUsesCanonicalPayloadForStructuredFormats(t *testing.T) {
	t.Parallel()

	type canonical struct {
		Status string `json:"status" toon:"status"`
	}

	result := OutputResult{
		Payload: canonical{Status: "ok"},
		TextWriter: func(io.Writer) error {
			return errors.New("text writer should not be used")
		},
	}

	for _, format := range []Format{FormatJSON, FormatTOON} {
		format := format

		t.Run(string(format), func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			if err := WriteOutput(&out, format, result); err != nil {
				t.Fatalf("WriteOutput() error = %v", err)
			}

			if !strings.Contains(out.String(), "ok") {
				t.Fatalf("WriteOutput() output = %q, want canonical payload contents", out.String())
			}
		})
	}
}
