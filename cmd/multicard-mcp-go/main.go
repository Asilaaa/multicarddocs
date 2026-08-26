// Command multicard-mcp-go starts the Multicard documentation MCP server.
package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"multicard-mcp-go/internal/config"
	"multicard-mcp-go/internal/docs"
	"multicard-mcp-go/internal/mcp"
	"multicard-mcp-go/internal/oauth"
	"multicard-mcp-go/internal/render"
	"multicard-mcp-go/internal/util"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	corp, err := docs.LoadCorpus(cfg.DocsDir)
	if err != nil {
		return fmt.Errorf("failed to load docs: %w", err)
	}

	if cfg.Search != "" {
		results := corp.Search(cfg.Search, util.Clamp(cfg.Limit, 1, 10))
		fmt.Println(render.SearchText(cfg.Search, results))
		return nil
	}
	if cfg.Ask != "" {
		results := corp.Search(cfg.Ask, util.Clamp(cfg.Limit, 1, 8))
		answer, _ := render.Answer(cfg.Ask, results)
		fmt.Println(answer)
		return nil
	}
	if cfg.GetDoc != "" {
		path := strings.TrimPrefix(strings.TrimSpace(cfg.GetDoc), "/")
		doc, ok := corp.ByPath(path)
		if !ok {
			return fmt.Errorf("document not found: %s", path)
		}
		fmt.Println(doc.Content)
		return nil
	}

	server := mcp.NewServer(corp)
	if cfg.HTTP {
		auth := oauth.NewServer(oauth.Options{
			PublicBaseURL:      cfg.PublicBaseURL,
			DemoUsername:       cfg.OAuthDemoUser,
			DemoPassword:       cfg.OAuthDemoPassword,
			GoogleClientID:     cfg.GoogleClientID,
			GoogleClientSecret: cfg.GoogleClientSecret,
		})
		fmt.Fprintf(os.Stderr, "multicard-mcp-go: loaded %d docs from %s\n", corp.TotalDocs(), cfg.DocsDir)
		signInMode := "demo login"
		if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
			signInMode = "Google sign-in"
		}
		fmt.Fprintf(os.Stderr, "multicard-mcp-go: serving HTTP JSON-RPC on %s (OAuth 2.1 protected, %s)\n", cfg.ListenAddr, signInMode)
		if err := server.ServeHTTP(cfg.ListenAddr, auth); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "multicard-mcp-go: loaded %d docs from %s\n", corp.TotalDocs(), cfg.DocsDir)
	fmt.Fprintln(os.Stderr, "multicard-mcp-go: waiting for MCP client on stdio")
	if err := server.ServeStdio(os.Stdin, os.Stdout); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
