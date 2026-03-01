package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/anchore/grype/grype"
	"github.com/anchore/grype/grype/db/v6/distribution"
	"github.com/anchore/grype/grype/db/v6/installation"
	"github.com/anchore/grype/grype/matcher"
	"github.com/anchore/grype/grype/pkg"
	"github.com/anchore/syft/syft"
	"github.com/muandane/go-gitgres/pkg/clone"
)

const defaultGitURL = "https://github.com/nginx/nginx"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	conn := flag.String("conn", "", "Postgres connection string (e.g. dbname=gitgres_test)")
	repo := flag.String("repo", "", "Repository name in Postgres")
	url := flag.String("url", "", "Git URL to fetch (default: "+defaultGitURL+")")
	flag.Parse()

	if *conn == "" || *repo == "" {
		log.Error("missing required flags", "usage", "-conn <conninfo> -repo <reponame> [-url <git-url>]")
		os.Exit(1)
	}

	ctx := context.Background()

	if *url != "" {
		log.Info("fetching and pushing to Postgres", "url", *url, "repo", *repo)
		cleanup, err := clone.FetchAndPushToGitgres(ctx, *conn, *repo, *url)
		if err != nil {
			log.Error("fetch and push failed", "err", err)
			os.Exit(1)
		}
		defer cleanup()
		log.Info("pushed to Postgres")
	}

	log.Info("cloning from Postgres", "repo", *repo)
	dir, cleanup, err := clone.CloneFromGitgres(ctx, *conn, *repo)
	if err != nil {
		log.Error("clone from gitgres failed", "err", err)
		os.Exit(1)
	}
	defer cleanup()

	// Catalog with Syft (library)
	log.Info("cataloging with Syft", "dir", dir)
	source, err := syft.GetSource(ctx, "dir:"+dir, syft.DefaultGetSourceConfig())
	if err != nil {
		log.Error("syft get source failed", "err", err)
		os.Exit(1)
	}
	_, err = syft.CreateSBOM(ctx, source, syft.DefaultCreateSBOMConfig())
	if err != nil {
		log.Error("syft create SBOM failed", "err", err)
		os.Exit(1)
	}
	log.Info("SBOM created")

	// Load Grype vulnerability DB (uses temp dir; first run may download over network)
	dbRoot, err := os.MkdirTemp("", "grype-db-*")
	if err != nil {
		log.Error("create grype db dir failed", "err", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dbRoot)

	distCfg := distribution.DefaultConfig()
	installCfg := installation.Config{
		DBRootDir:        dbRoot,
		ValidateChecksum: false,
		ValidateAge:      false,
	}
	store, status, err := grype.LoadVulnerabilityDB(distCfg, installCfg, true)
	if err != nil {
		log.Error("load vulnerability DB failed", "err", err)
		os.Exit(1)
	}
	if c, ok := store.(io.Closer); ok {
		defer c.Close()
	}
	log.Info("vulnerability DB loaded", "path", status.Path)

	// Get packages from dir (Grype uses Syft internally for dir: input)
	packages, pkgContext, _, err := pkg.Provide("dir:"+dir, pkg.ProviderConfig{})
	if err != nil {
		log.Error("grype provide packages failed", "err", err)
		os.Exit(1)
	}

	// Match vulnerabilities
	matchers := matcher.NewDefaultMatchers(matcher.Config{})
	vm := grype.VulnerabilityMatcher{VulnerabilityProvider: store, Matchers: matchers}
	remainingMatches, _, err := vm.FindMatchesContext(ctx, packages, pkgContext)
	if err != nil {
		log.Error("find matches failed", "err", err)
		os.Exit(1)
	}

	// Print results (table-style)
	fmt.Fprintln(os.Stdout, "\nVulnerability matches:")
	fmt.Fprintf(os.Stdout, "%-20s %-12s %s\n", "ID", "SEVERITY", "PACKAGE")
	fmt.Fprintln(os.Stdout, "-------------------- ------------ --------")
	for _, m := range remainingMatches.Sorted() {
		sev := "unknown"
		if m.Vulnerability.Metadata != nil {
			sev = m.Vulnerability.Metadata.Severity
		}
		fmt.Fprintf(os.Stdout, "%-20s %-12s %s\n", m.Vulnerability.ID, sev, m.Package.Name)
	}
	log.Info("scan complete", "matches", remainingMatches.Count())
}
