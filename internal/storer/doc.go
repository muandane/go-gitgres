// Package storer provides a go-git storage.Storer implementation backed by PostgreSQL.
//
// Create a storer with NewPostgresStorer(ctx, pool, repoName). The repo is created
// if it does not exist. The returned *PostgresStorer implements storage.Storer and
// can be used with go-git's Repository (e.g. push, fetch, clone). Use RepoID to
// get the repository ID; use ListRefsRows for a single-query ref list when you do
// not need plumbing.Reference values.
package storer
