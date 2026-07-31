// Package config contains command-line and environment configuration helpers.
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the runtime configuration for the MCP server binary.
type Config struct {
	DocsDir    string
	Search     string
	Ask        string
	GetDoc     string
	HTTP       bool
	ListenAddr string
	Limit      int
}

// Load parses command-line flags and returns application configuration.
func Load(args []string) (Config, error) {
	fs := flag.NewFlagSet("multicard-mcp-go", flag.ContinueOnError)

	var cfg Config
	var docsDirFlag string
	fs.StringVar(&docsDirFlag, "docs-dir", "", "Path to the multicard-docs directory")
	fs.StringVar(&cfg.Search, "search", "", "Run a one-off local search instead of starting the MCP server")
	fs.StringVar(&cfg.Ask, "ask", "", "Ask a one-off question from the local docs instead of starting the MCP server")
	fs.StringVar(&cfg.GetDoc, "get-doc", "", "Print one full markdown doc by relative path instead of starting the MCP server")
	fs.BoolVar(&cfg.HTTP, "http", false, "Run as an HTTP JSON-RPC service instead of stdio MCP")
	fs.StringVar(&cfg.ListenAddr, "listen-addr", EnvOrDefault("LISTEN_ADDR", "127.0.0.1:8080"), "HTTP listen address for --http mode")
	fs.IntVar(&cfg.Limit, "limit", 5, "Result limit for --search or --ask")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	docsDir, err := ResolveDocsDir(docsDirFlag)
	if err != nil {
		return Config{}, err
	}
	cfg.DocsDir = docsDir
	return cfg, nil
}

// EnvOrDefault returns an environment variable value or a fallback when it is empty.
func EnvOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// ResolveDocsDir locates the bundled multicard-docs folder from CLI flag, env, cwd, or binary path.
func ResolveDocsDir(flagValue string) (string, error) {
	candidates := make([]string, 0, 6)
	if flagValue != "" {
		candidates = append(candidates, flagValue)
	}
	if env := os.Getenv("MULTICARD_DOCS_DIR"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, "multicard-docs")

	exe, _ := os.Executable()
	if exe != "" {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "multicard-docs"),
			filepath.Join(exeDir, "..", "multicard-docs"),
		)
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, "..", "multicard-docs"))
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		info, err := os.Stat(abs)
		if err == nil && info.IsDir() {
			return abs, nil
		}
	}

	return "", fmt.Errorf("could not find multicard-docs directory")
}
