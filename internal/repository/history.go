package repository

import (
	"encoding/json"
	"martis/internal/domain"
	"os"
	"path/filepath"
)

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "martis")
}

func LegacyConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil { return "." }
	return filepath.Join(home, ".config", "martis")
}

// WriteJSON replaces a file atomically; saved requests may contain credentials.
func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".martis-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func LoadHistory(path string) ([]domain.HistoryEntry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if path == filepath.Join(ConfigDir(), "history.json") {
			legacy, legacyErr := os.ReadFile(filepath.Join(LegacyConfigDir(), "history.json"))
			if legacyErr == nil { var entries []domain.HistoryEntry; if decodeErr:=json.Unmarshal(legacy,&entries);decodeErr!=nil{return nil,decodeErr};if saveErr:=WriteJSON(path,entries);saveErr!=nil{return nil,saveErr};return entries,nil }
			if !os.IsNotExist(legacyErr) { return nil, legacyErr }
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []domain.HistoryEntry
	err = json.Unmarshal(data, &entries)
	return entries, err
}

func AppendHistory(path string, entry domain.HistoryEntry) error {
	entries, err := LoadHistory(path)
	if err != nil {
		return err
	}
	entries = append([]domain.HistoryEntry{entry}, entries...)
	if len(entries) > 100 {
		entries = entries[:100]
	}
	return WriteJSON(path, entries)
}
