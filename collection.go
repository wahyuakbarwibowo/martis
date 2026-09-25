package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CollectionItem merepresentasikan sebuah endpoint request yang tersimpan
type CollectionItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	HeaderKey  string `json:"header_key,omitempty"`
	HeaderVal  string `json:"header_val,omitempty"`
	HeaderAuth string `json:"header_auth,omitempty"`
	BodyType   string `json:"body_type"` // "raw" atau "form"
	BodyRaw    string `json:"body_raw,omitempty"`
	FormKey    string `json:"form_key,omitempty"`
	FormPath   string `json:"form_path,omitempty"`
}

// Folder merepresentasikan sebuah grup request
type Folder struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	IsExpanded bool             `json:"is_expanded"`
	Items      []CollectionItem `json:"items"`
}

// Collection merepresentasikan kumpulan folder request
type Collection struct {
	Name    string   `json:"name"`
	Folders []Folder `json:"folders"`
}

// Dapatkan path default untuk file storage collection
func getCollectionFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "martis_collections.json"
	}
	configDir := filepath.Join(home, ".config", "martis")
	_ = os.MkdirAll(configDir, 0755)
	return filepath.Join(configDir, "collections.json")
}

// Load default collection dari disk atau buat default jika belum ada
func loadCollections() Collection {
	path := getCollectionFilePath()
	data, err := os.ReadFile(path)
	if err == nil {
		var col Collection
		if json.Unmarshal(data, &col) == nil && len(col.Folders) > 0 {
			return col
		}
	}

	// Default template collection dengan struktur folders
	defaultCol := Collection{
		Name: "Default Workspace",
		Folders: []Folder{
			{
				ID:         "folder-auth",
				Name:       "Authentication & Users",
				IsExpanded: true,
				Items: []CollectionItem{
					{
						ID:        "item-1",
						Name:      "Get All Users",
						Method:    "GET",
						URL:       "https://jsonplaceholder.typicode.com/users",
						HeaderKey: "Accept",
						HeaderVal: "application/json",
						BodyType:  "raw",
						BodyRaw:   "",
					},
					{
						ID:        "item-2",
						Name:      "Create New User",
						Method:    "POST",
						URL:       "https://reqres.in/api/users",
						HeaderKey: "Content-Type",
						HeaderVal: "application/json",
						BodyType:  "raw",
						BodyRaw:   "{\n  \"name\": \"Martis\",\n  \"job\": \"Ashura King\"\n}",
					},
				},
			},
			{
				ID:         "folder-echo",
				Name:       "Testing & Echo Service",
				IsExpanded: true,
				Items: []CollectionItem{
					{
						ID:        "item-3",
						Name:      "HTTPBin Anything (POST)",
						Method:    "POST",
						URL:       "https://httpbin.org/anything",
						HeaderKey: "Content-Type",
						HeaderVal: "application/json",
						BodyType:  "raw",
						BodyRaw:   "{\n  \"message\": \"Hello from Martis!\",\n  \"status\": \"fast\"\n}",
					},
					{
						ID:        "item-4",
						Name:      "HTTPBin IP Checker (GET)",
						Method:    "GET",
						URL:       "https://httpbin.org/ip",
						HeaderKey: "Accept",
						HeaderVal: "application/json",
						BodyType:  "raw",
						BodyRaw:   "",
					},
				},
			},
		},
	}
	_ = saveCollections(defaultCol)
	return defaultCol
}

// Simpan collection ke disk
func saveCollections(col Collection) error {
	path := getCollectionFilePath()
	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
