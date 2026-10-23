package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type config struct {
	APIKey string `toml:"api_key"`
}

func configPath(dir string) string { return filepath.Join(dir, "config.toml") }

func loadKey(dir string) (string, error) {
	if dir == "" {
		return "", nil
	}
	var c config
	_, err := toml.DecodeFile(configPath(dir), &c)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", newError(exitError, "cannot read %s: %v", configPath(dir), err)
	}
	return c.APIKey, nil
}

// saveKey writes the config to a temp file in the same directory and renames
// it over config.toml, so that a failed write leaves the old file whole.
func saveKey(dir, key string) error {
	if dir == "" {
		return newError(exitError, "cannot find the user config directory (set PEPPOL_SH_API_KEY instead)")
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(config{APIKey: key}); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return newError(exitError, "cannot create %s: %v", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return newError(exitError, "cannot set the mode of %s: %v", dir, err)
	}

	// CreateTemp makes the file with mode 0600.
	tmp, err := os.CreateTemp(dir, "config-*.toml")
	if err != nil {
		return newError(exitError, "cannot write to %s: %v", dir, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return newError(exitError, "cannot write %s: %v", tmp.Name(), err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return newError(exitError, "cannot write %s: %v", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return newError(exitError, "cannot write %s: %v", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), configPath(dir)); err != nil {
		return newError(exitError, "cannot write %s: %v", configPath(dir), err)
	}
	return nil
}

// removeKey reports whether there was a config file to remove.
func removeKey(dir string) (bool, error) {
	if dir == "" {
		return false, nil
	}
	err := os.Remove(configPath(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, newError(exitError, "cannot remove %s: %v", configPath(dir), err)
	}
	return true, nil
}
