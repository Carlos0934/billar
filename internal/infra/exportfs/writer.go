package exportfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RootedWriter struct{ Root string }
type DirectWriter struct{}

func (w RootedWriter) Resolve(filename, outputPath string) (string, error) {
	root := strings.TrimSpace(w.Root)
	if root == "" {
		return "", errors.New("pdf export disabled: BILLAR_EXPORT_DIR not configured")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve export root: %w", err)
	}

	rel := strings.TrimSpace(outputPath)
	if rel == "" {
		name := strings.TrimSpace(filename)
		if name == "" {
			name = "invoice.pdf"
		}
		if name != filepath.Base(name) || strings.ContainsAny(name, `/\\`) {
			return "", errors.New("filename must not contain path separators")
		}
		rel = name
	} else if filepath.IsAbs(rel) {
		return "", errors.New("output path must be relative to export root")
	}

	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("output path escapes export root")
	}
	abs := filepath.Join(absRoot, clean)
	relToRoot, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return "", fmt.Errorf("validate export path: %w", err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relToRoot) {
		return "", errors.New("output path escapes export root")
	}
	return abs, nil
}

func (DirectWriter) Resolve(filename, outputPath string) (string, error) {
	target := strings.TrimSpace(outputPath)
	if target == "" {
		target = strings.TrimSpace(filename)
	}
	if target == "" {
		return "", errors.New("output path is required")
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	return absTarget, nil
}

func (RootedWriter) Write(absPath string, data []byte) (int64, error) {
	return writeFile(absPath, data)
}
func (DirectWriter) Write(absPath string, data []byte) (int64, error) {
	return writeFile(absPath, data)
}

func writeFile(path string, data []byte) (int64, error) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, fmt.Errorf("write file %s: %w", path, err)
	}
	return int64(len(data)), nil
}
