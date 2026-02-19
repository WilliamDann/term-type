package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Result struct {
	Date     time.Time `json:"date"`
	Mode     string    `json:"mode"`
	WPM      float64   `json:"wpm"`
	Accuracy float64   `json:"accuracy"`
	Correct  int       `json:"correct"`
	Wrong    int       `json:"wrong"`
}

func historyPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "term-type", "history.json")
}

func loadHistory() ([]Result, error) {
	path := historyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var results []Result
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func saveResult(r Result) error {
	results, _ := loadHistory()
	results = append(results, r)

	path := historyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
