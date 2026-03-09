package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Preferences struct {
	DiffFormat   DiffFormat `json:"diff_format,omitempty"`
	SidebarWidth string     `json:"sidebar_width,omitempty"`
}

func preferencesPath() string {
	return filepath.Join(xdg.ConfigHome, "meatcheck", "preferences.json")
}

func preferredDiffFormat() DiffFormat {
	p := loadPreferences()
	if p.DiffFormat == DiffFormatSplit || p.DiffFormat == DiffFormatUnified {
		return p.DiffFormat
	}
	return DiffFormatUnified
}

func loadPreferences() Preferences {
	data, err := os.ReadFile(preferencesPath())
	if err != nil {
		return Preferences{}
	}
	var p Preferences
	_ = json.Unmarshal(data, &p)
	return p
}

func savePreference(fn func(*Preferences)) {
	p := loadPreferences()
	fn(&p)
	path := preferencesPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
