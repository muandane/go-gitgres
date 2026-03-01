package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"go-gitgres/examples/common"
)

const defaultGitURL = "https://github.com/nginx/nginx"

func main() {
	conn := flag.String("conn", "", "Postgres connection string (e.g. dbname=gitgres_test)")
	repo := flag.String("repo", "", "Repository name in Postgres")
	url := flag.String("url", "", "Git URL to fetch (default: "+defaultGitURL+")")
	flag.Parse()

	if *conn == "" || *repo == "" {
		fmt.Fprintf(os.Stderr, "usage: %s -conn <conninfo> -repo <reponame> [-url <git-url>]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -url: if set, fetch from URL and push to Postgres before clone+scan (e.g. %s)\n", defaultGitURL)
		os.Exit(1)
	}

	ctx := context.Background()

	if *url != "" {
		cleanup, err := common.FetchAndPushToGitgres(ctx, *conn, *repo, *url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch and push: %v\n", err)
			os.Exit(1)
		}
		defer cleanup()
	}

	dir, cleanup, err := common.CloneFromGitgres(ctx, *conn, *repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clone from gitgres: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, "trivy", "fs", "--scanners", "vuln", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "trivy: %v\n", err)
		os.Exit(1)
	}
}
