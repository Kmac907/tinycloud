package tinyterraformcmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func TinyTerraformAzShimAssetPath(repoRoot string) string {
	return filepath.Join(repoRoot, "scripts", "tinyterraform-azshim.ps1")
}

func LoadTinyTerraformAzShimScript(repoRoot string) (string, error) {
	path := TinyTerraformAzShimAssetPath(repoRoot)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read tinyterraform az shim asset: %w", err)
	}
	return string(content), nil
}
