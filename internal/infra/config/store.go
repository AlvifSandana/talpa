package config

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
)

type Store struct{}

func NewStore() Store { return Store{} }

func (Store) LoadWhitelist(ctx context.Context) ([]string, error) {
	_ = ctx
	path, err := whitelistPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = expandHome(line)
		line = filepath.Clean(line)
		if abs, err := filepath.Abs(line); err == nil {
			line = abs
		}
		out = append(out, line)
	}
	return out, s.Err()
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func whitelistPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "talpa", "whitelist"), nil
}
