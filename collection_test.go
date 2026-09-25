package main

import (
	"os"
	"testing"
)

func TestCollectionStorage(t *testing.T) {
	col := loadCollections()
	if len(col.Folders) == 0 {
		t.Fatal("expected default collection to have folders")
	}

	testItem := CollectionItem{
		ID:     "test-item",
		Name:   "Test Request",
		Method: "GET",
		URL:    "https://api.example.com",
	}

	col.Folders[0].Items = append(col.Folders[0].Items, testItem)
	err := saveCollections(col)
	if err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}

	loaded := loadCollections()
	found := false
	for _, f := range loaded.Folders {
		for _, it := range f.Items {
			if it.ID == "test-item" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("expected test item to be present after save and reload")
	}

	// Cleanup test item
	var cleaned []CollectionItem
	for _, it := range loaded.Folders[0].Items {
		if it.ID != "test-item" {
			cleaned = append(cleaned, it)
		}
	}
	loaded.Folders[0].Items = cleaned
	_ = saveCollections(loaded)
}

func TestCollectionPath(t *testing.T) {
	path := getCollectionFilePath()
	if path == "" {
		t.Error("expected valid collection file path")
	}
	_ = os.Remove(path)
}
