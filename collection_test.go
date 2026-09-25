package main

import (
	"os"
	"testing"
)

func TestCollectionStorage(t *testing.T) {
	col := loadCollections()
	if len(col.Items) == 0 {
		t.Fatal("expected default collection to have items")
	}

	testItem := CollectionItem{
		ID:     "test-item",
		Name:   "Test Request",
		Method: "GET",
		URL:    "https://api.example.com",
	}

	col.Items = append(col.Items, testItem)
	err := saveCollections(col)
	if err != nil {
		t.Fatalf("failed to save collection: %v", err)
	}

	loaded := loadCollections()
	found := false
	for _, it := range loaded.Items {
		if it.ID == "test-item" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected test item to be present after save and reload")
	}

	// Cleanup test item
	var cleaned []CollectionItem
	for _, it := range loaded.Items {
		if it.ID != "test-item" {
			cleaned = append(cleaned, it)
		}
	}
	loaded.Items = cleaned
	_ = saveCollections(loaded)
}

func TestCollectionPath(t *testing.T) {
	path := getCollectionFilePath()
	if path == "" {
		t.Error("expected valid collection file path")
	}
	_ = os.Remove(path) // safe test
}
