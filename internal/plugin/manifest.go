package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveManifest saves a plugin manifest to file
func SaveManifest(pluginDir string, manifest Manifest) error {
	manifestPath := filepath.Join(pluginDir, "plugin.json")

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// ValidateManifest validates a plugin manifest
func ValidateManifest(manifest Manifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("manifest missing name")
	}
	if manifest.Version == "" {
		return fmt.Errorf("manifest missing version")
	}
	if manifest.EntryPoint == "" {
		return fmt.Errorf("manifest missing entry_point")
	}
	if manifest.Description == "" {
		return fmt.Errorf("manifest missing description")
	}
	if manifest.Author == "" {
		return fmt.Errorf("manifest missing author")
	}

	return nil
}

// LoadManifestFromFile loads a plugin manifest from file
func LoadManifestFromFile(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}
