package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/exec"

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
	log.Info("cloned; running Syft", "dir", dir)

	syftCmd := exec.CommandContext(ctx, "syft", "dir:"+dir, "-o", "cyclonedx-json")
	syftOut, err := syftCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Stderr.Write(exitErr.Stderr)
			os.Exit(exitErr.ExitCode())
		}
		log.Error("syft failed", "err", err)
		os.Exit(1)
	}

	sbomFile := dir + "/sbom.cyclonedx.json"
	if err := os.WriteFile(sbomFile, syftOut, 0o644); err != nil {
		log.Error("write sbom failed", "err", err)
		os.Exit(1)
	}
	log.Info("SBOM written; running Grype")

	grypeCmd := exec.CommandContext(ctx, "grype", "sbom:"+sbomFile)
	grypeCmd.Stdout = os.Stdout
	grypeCmd.Stderr = os.Stderr
	if err := grypeCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Error("grype failed", "err", err)
		os.Exit(1)
	}
}
