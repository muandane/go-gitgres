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
	log.Info("cloned; running Trivy", "dir", dir)

	cmd := exec.CommandContext(ctx, "trivy", "fs", "--scanners", "vuln", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Error("trivy failed", "err", err)
		os.Exit(1)
	}
}
