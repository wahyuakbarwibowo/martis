package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"martis/internal/domain"
)

// CollectionRepository interface untuk abstraksi penyimpanan data
type CollectionRepository interface {
	Load() (*domain.Collection, error)
	Save(col *domain.Collection) error
}

type fileCollectionRepository struct {
	filePath string
}

// NewFileCollectionRepository membuat instance repository berbasis JSON file
func NewFileCollectionRepository(customPath ...string) CollectionRepository {
	path := ""
	if len(customPath) > 0 && customPath[0] != "" {
		path = customPath[0]
	} else {
		path = defaultStoragePath()
	}
	return &fileCollectionRepository{filePath: path}
}

func defaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "martis_collections.json"
	}
	configDir := filepath.Join(home, ".config", "martis")
	_ = os.MkdirAll(configDir, 0755)
	return filepath.Join(configDir, "collections.json")
}

// Load membaca data collection dari disk
func (r *fileCollectionRepository) Load() (*domain.Collection, error) {
	data, err := os.ReadFile(r.filePath)
	if err == nil {
		var col domain.Collection
		if json.Unmarshal(data, &col) == nil && len(col.Folders) > 0 {
			return &col, nil
		}
	}

	// Default template jika file belum ada
	defaultCol := &domain.Collection{
		Name: "Default Workspace",
		Folders: []domain.Folder{
			{
				ID:         "folder-auth",
				Name:       "Authentication & Users",
				IsExpanded: true,
				Items: []domain.CollectionItem{
					{
						ID:        "item-1",
						Name:      "Get All Users",
						Method:    "GET",
						URL:       "https://jsonplaceholder.typicode.com/users",
						HeaderKey: "Accept",
						HeaderVal: "application/json",
						BodyType:  "raw",
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
				Items: []domain.CollectionItem{
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
					},
				},
			},
		},
	}
	_ = r.Save(defaultCol)
	return defaultCol, nil
}

// Save menyimpan data collection ke disk
func (r *fileCollectionRepository) Save(col *domain.Collection) error {
	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}
	return os.WriteFile(r.filePath, data, 0644)
}
