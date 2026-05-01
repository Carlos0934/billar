package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Lister struct{}

func LookupByID(dir, id string) (Record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, fmt.Errorf("backup %s not found", id)
		}
		return Record{}, fmt.Errorf("read backup directory %q: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		_, fileID := parseRecordName(path)
		if fileID != id {
			continue
		}
		record, err := readRecord(path)
		if err != nil {
			return Record{}, err
		}
		if !record.Metadata {
			return Record{}, fmt.Errorf("sidecar_missing: backup %s is missing metadata sidecar %q", id, path+".json")
		}
		if record.ID != fileID {
			return Record{}, fmt.Errorf("id_basename_mismatch: sidecar id %q does not match file basename %q", record.ID, fileID)
		}
		return record, nil
	}

	return Record{}, fmt.Errorf("backup %s not found", id)
}

func (Lister) List(ctx context.Context, dir string) ([]Record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, fmt.Errorf("read backup directory %q: %w", dir, err)
	}

	records := make([]Record, 0)
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		record, err := readRecord(path)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	return records, nil
}

func readRecord(dbPath string) (Record, error) {
	sidecarPath := dbPath + ".json"
	data, err := os.ReadFile(sidecarPath)
	if err == nil {
		var record Record
		if err := json.Unmarshal(data, &record); err != nil {
			return Record{}, fmt.Errorf("parse backup metadata %q: %w", sidecarPath, err)
		}
		record.File = dbPath
		record.SidecarFile = sidecarPath
		record.Metadata = true
		return record, nil
	}
	if !os.IsNotExist(err) {
		return Record{}, fmt.Errorf("read backup metadata %q: %w", sidecarPath, err)
	}

	createdAt, id := parseRecordName(dbPath)
	info, statErr := os.Stat(dbPath)
	if statErr != nil {
		return Record{}, fmt.Errorf("stat backup %q: %w", dbPath, statErr)
	}
	return Record{
		ID:          id,
		File:        dbPath,
		SidecarFile: sidecarPath,
		CreatedAt:   createdAt,
		SizeBytes:   info.Size(),
		Metadata:    false,
	}, nil
}

func parseRecordName(path string) (time.Time, string) {
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		if ts, err := time.Parse("20060102T150405Z", parts[1]); err == nil {
			return ts.UTC(), id
		}
	}
	return time.Time{}, id
}
