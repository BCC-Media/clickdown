package config

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Token        string
	DBPath       string
	ListenAddr   string
	SyncInterval time.Duration
}

func Load() (Config, error) {
	// Auto-load .env files (existing env always wins).
	loadDotEnv(".env")
	if home, err := os.UserHomeDir(); err == nil {
		loadDotEnv(filepath.Join(home, ".clickdown", ".env"))
	}

	c := Config{
		Token:        firstNonEmpty(os.Getenv("CLICKUP_API_TOKEN"), os.Getenv("CLICKUP_TOKEN")),
		DBPath:       envOr("CLICKDOWN_DB", defaultDBPath()),
		ListenAddr:   envOr("CLICKDOWN_ADDR", "127.0.0.1:7878"),
		SyncInterval: 10 * time.Minute,
	}
	if v := os.Getenv("CLICKDOWN_SYNC_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return c, err
		}
		c.SyncInterval = d
	}

	fs := flag.NewFlagSet("clickdown", flag.ContinueOnError)
	fs.StringVar(&c.Token, "token", c.Token, "ClickUp personal API token")
	fs.StringVar(&c.DBPath, "db", c.DBPath, "Path to SQLite database file")
	fs.StringVar(&c.ListenAddr, "addr", c.ListenAddr, "HTTP listen address")
	fs.DurationVar(&c.SyncInterval, "sync-interval", c.SyncInterval, "Auto-sync interval")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		return c, err
	}

	if c.SyncInterval < 30*time.Second {
		c.SyncInterval = 30 * time.Second
	}

	if err := os.MkdirAll(filepath.Dir(c.DBPath), 0o755); err != nil {
		return c, err
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "clickdown.db"
	}
	return filepath.Join(home, ".clickdown", "clickdown.db")
}

// loadDotEnv reads a KEY=VALUE file and sets any keys not already in the
// process environment. Silently ignores a missing file.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Allow "export KEY=VAL"
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		// Strip optional surrounding quotes.
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, present := os.LookupEnv(key); present {
			continue
		}
		_ = os.Setenv(key, val)
	}
}
