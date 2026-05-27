package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Dir returns (and creates) the cache directory.
// Falls back to os.TempDir() if the user home directory cannot be determined.
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	dir := filepath.Join(home, ".git-explain", "cache")
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// Key returns a stable cache key for a given prompt + provider.
func Key(provider, prompt string) string {
	h := sha256.Sum256([]byte(provider + "::" + prompt))
	return fmt.Sprintf("%x", h)[:32]
}

// Get returns cached response if present and not expired.
func Get(key string, maxAge time.Duration) (string, bool) {
	path := filepath.Join(Dir(), key+".gz")
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if maxAge > 0 && time.Since(info.ModTime()) > maxAge {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", false
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// Set writes a response to the cache atomically (write to temp, then rename).
func Set(key, content string) error {
	dir := Dir()
	path := filepath.Join(dir, key+".gz")
	// Write to a temp file first so a crash doesn't corrupt the cache entry.
	tmp, err := os.CreateTemp(dir, "*.gz.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	gz := gzip.NewWriter(tmp)
	if _, err := io.Copy(gz, strings.NewReader(content)); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	committed = true
	return os.Rename(tmpName, path)
}

// Clear removes all cache entries.
func Clear() error {
	return os.RemoveAll(Dir())
}
