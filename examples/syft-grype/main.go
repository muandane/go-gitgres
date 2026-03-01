package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/muandane/go-gitgres/pkg/clone"
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
		cleanup, err := clone.FetchAndPushToGitgres(ctx, *conn, *repo, *url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch and push: %v\n", err)
			os.Exit(1)
		}
		defer cleanup()
	}

	dir, cleanup, err := clone.CloneFromGitgres(ctx, *conn, *repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clone from gitgres: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	syftCmd := exec.CommandContext(ctx, "syft", "dir:"+dir, "-o", "cyclonedx-json")
	syftOut, err := syftCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprint(os.Stderr, string(exitErr.Stderr))
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "syft: %v\n", err)
		os.Exit(1)
	}

	sbomFile := dir + "/sbom.cyclonedx.json"
	if err := os.WriteFile(sbomFile, syftOut, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write sbom: %v\n", err)
		os.Exit(1)
	}

	grypeCmd := exec.CommandContext(ctx, "grype", "sbom:"+sbomFile)
	grypeCmd.Stdout = os.Stdout
	grypeCmd.Stderr = os.Stderr
	if err := grypeCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "grype: %v\n", err)
		os.Exit(1)
	}
}
