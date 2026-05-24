package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()

	a, err := Load(filepath.Join(dir, "missing.json"))
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if a.Services == nil {
		t.Fatal("Services is nil")
	}
	if len(a.Services) != 0 {
		t.Fatalf("len(Services) = %d, want 0", len(a.Services))
	}
}

func TestGetService(t *testing.T) {
	a := SdsService{}
	sample := New("api", IndependentType)
	if err := a.SetService(*sample); err != nil {
		t.Fatalf("SetService: %v", err)
	}

	found, err := a.GetService("api")
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if found.Name != "api" {
		t.Fatalf("Name = %q, want api", found.Name)
	}

	if _, err := a.GetService("missing"); err == nil {
		t.Fatal("GetService missing service returned nil error")
	}
}

func TestLoadAppliesServiceDefaults(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "app.json")
	data := []byte(`{
  "services": [
    {
      "type": "Independent",
      "name": "api",
      "start-command": "go run ./cmd/api",
      "handlers": []
    }
  ]
}
`)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	loaded, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Services[0].StopCommand != DefaultStopCommand {
		t.Fatalf("StopCommand = %q, want %q", loaded.Services[0].StopCommand, DefaultStopCommand)
	}
}

func TestSetService(t *testing.T) {
	a := SdsService{}
	first := New("api", IndependentType)
	second := New("proxy", ProxyType)

	if err := a.SetService(*first); err != nil {
		t.Fatalf("SetService first: %v", err)
	}
	if err := a.SetService(*second); err != nil {
		t.Fatalf("SetService second: %v", err)
	}
	if len(a.Services) != 2 {
		t.Fatalf("len(Services) = %d, want 2", len(a.Services))
	}

	updated := *first
	updated.StartCommand = "go run ./cmd/api"
	updated.StopCommand = "systemctl stop api"
	updated.StatusCommand = "systemctl status api"
	if err := a.SetService(updated); err != nil {
		t.Fatalf("SetService update: %v", err)
	}
	if len(a.Services) != 2 {
		t.Fatalf("len(Services) after update = %d, want 2", len(a.Services))
	}

	found, err := a.GetService("api")
	if err != nil {
		t.Fatalf("GetService updated: %v", err)
	}
	if found.StartCommand != "go run ./cmd/api" {
		t.Fatalf("StartCommand = %q, want go run ./cmd/api", found.StartCommand)
	}
	if found.StopCommand != "systemctl stop api" {
		t.Fatalf("StopCommand = %q, want systemctl stop api", found.StopCommand)
	}
	if found.StatusCommand != "systemctl status api" {
		t.Fatalf("StatusCommand = %q, want systemctl status api", found.StatusCommand)
	}
}

func TestRemoveService(t *testing.T) {
	a := SdsService{}
	first := New("api", IndependentType)
	second := New("proxy", ProxyType)
	if err := a.SetService(*first); err != nil {
		t.Fatalf("SetService first: %v", err)
	}
	if err := a.SetService(*second); err != nil {
		t.Fatalf("SetService second: %v", err)
	}

	if err := a.RemoveService(""); err == nil {
		t.Fatal("RemoveService with empty name returned nil error")
	}
	if err := a.RemoveService("missing"); err == nil {
		t.Fatal("RemoveService with missing service returned nil error")
	}

	if err := a.RemoveService("api"); err != nil {
		t.Fatalf("RemoveService: %v", err)
	}
	if len(a.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(a.Services))
	}
	if a.Services[0].Name != "proxy" {
		t.Fatalf("remaining service = %q, want proxy", a.Services[0].Name)
	}
}

func TestLoadSave(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "app.json")
	original, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	sample := New("api", IndependentType)
	sample.Handlers = []Handler{
		{
			Type:   ReplierType,
			Socket: Socket{Id: "api_1", Port: 4101},
		},
	}
	if err := original.SetService(*sample); err != nil {
		t.Fatalf("SetService: %v", err)
	}

	if err := original.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("os.ReadFile: %v", err)
	}
	if !jsonLooksIndented(data) {
		t.Fatalf("written JSON is not indented: %s", string(data))
	}

	loaded, err := Load(filePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(loaded.Services))
	}
	if loaded.Services[0].Name != "api" {
		t.Fatalf("Name = %q, want api", loaded.Services[0].Name)
	}
	if loaded.Services[0].Handlers[0].Socket.Port != 4101 {
		t.Fatalf("Port = %d, want 4101", loaded.Services[0].Handlers[0].Socket.Port)
	}
}

func TestSaveWithoutFilePath(t *testing.T) {
	if err := (SdsService{}).Save(); err == nil {
		t.Fatal("Save without file path returned nil error")
	}
}

func jsonLooksIndented(data []byte) bool {
	for _, b := range data {
		if b == '\n' {
			return true
		}
	}
	return false
}
