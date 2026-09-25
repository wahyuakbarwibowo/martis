package repository

import (
	"os"
	"path/filepath"
	"testing"

	"martis/internal/domain"
)

func TestCollectionRepository(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_collections.json")

	repo := NewFileCollectionRepository(tempFile)

	col, err := repo.Load()
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if len(col.Folders) == 0 {
		t.Fatal("expected default folders to be created")
	}

	testItem := domain.CollectionItem{
		ID:     "unit-test-item",
		Name:   "Unit Test Endpoint",
		Method: "GET",
		URL:    "https://api.test.com",
	}

	col.Folders[0].Items = append(col.Folders[0].Items, testItem)
	if err := repo.Save(col); err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}

	// Reload from disk
	reloaded, err := repo.Load()
	if err != nil {
		t.Fatalf("failed to reload collection: %v", err)
	}

	found := false
	for _, it := range reloaded.Folders[0].Items {
		if it.ID == "unit-test-item" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected unit-test-item to be persisted and reloaded")
	}

	_ = os.Remove(tempFile)
}
