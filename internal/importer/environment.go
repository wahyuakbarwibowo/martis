package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"martis/internal/environment"
)

// PostmanEnvironment converts a Postman environment export into dotenv text.
// ok is false when the file is not a Postman environment.
func PostmanEnvironment(path string) (name, dotenv string, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, err
	}
	var raw struct {
		Name   string `json:"name"`
		Values []struct {
			Key     string `json:"key"`
			Value   any    `json:"value"`
			Enabled *bool  `json:"enabled"`
		} `json:"values"`
	}
	if json.Unmarshal(data, &raw) != nil || raw.Values == nil {
		return "", "", false, nil
	}
	var lines []string
	for _, v := range raw.Values {
		if v.Enabled != nil && !*v.Enabled {
			continue
		}
		value := fmt.Sprint(v.Value)
		if v.Value == nil {
			value = ""
		}
		if !environment.ValidName(v.Key) || strings.ContainsAny(value, "\r\n") {
			return "", "", true, fmt.Errorf("variabel %q tidak bisa disimpan ke .env", v.Key)
		}
		lines = append(lines, v.Key+"=\""+value+"\"")
	}
	sort.Strings(lines)
	return raw.Name, strings.Join(lines, "\n") + "\n", true, nil
}
