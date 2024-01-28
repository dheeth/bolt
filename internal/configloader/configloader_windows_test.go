// go:build windows

package configloader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenetriguba/bolt/internal/bolttest"
	"github.com/eugenetriguba/bolt/internal/configloader"
	"github.com/eugenetriguba/checkmate/assert"
)

func TestNewConfigWindowsSearchesToRootFilePath(t *testing.T) {
	expectedCfg := configloader.Config{
		Migrations: configloader.MigrationsConfig{DirectoryPath: "differentmigrationsdir"},
	}
	configPath := filepath.Join(os.Getenv("SystemDrive"), "bolt.toml")
	bolttest.CreateConfigFile(t, &expectedCfg, configPath)
	assert.Nil(t, err)

	homedir, err := os.UserHomeDir()
	assert.Nil(t, err)
	bolttest.ChangeCwd(t, homedir)

	cfg, err := configloader.NewConfig()
	assert.Nil(t, err)

	assert.DeepEqual(t, *cfg, expectedCfg)
}
