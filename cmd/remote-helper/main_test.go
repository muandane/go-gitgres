package main

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		url        string
		wantConn   string
		wantRepo   string
		wantErr    bool
	}{
		{"dbname=gitgres_test/myrepo", "dbname=gitgres_test", "myrepo", false},
		{"host=localhost dbname=foo/bar", "host=localhost dbname=foo", "bar", false},
		{"a/b/c", "a/b", "c", false},
		{"nopath", "", "", true},
		{"/onlyslash", "", "", true},
		{"trailing/", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		conn, repo, err := parseURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseURL(%q) err = %v, wantErr %v", tt.url, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (conn != tt.wantConn || repo != tt.wantRepo) {
			t.Errorf("parseURL(%q) = %q, %q; want %q, %q", tt.url, conn, repo, tt.wantConn, tt.wantRepo)
		}
	}
}

func TestParsePushLine(t *testing.T) {
	tests := []struct {
		line string
		want []pushSpec
	}{
		{"push refs/heads/main:refs/heads/main", []pushSpec{{src: "refs/heads/main", dst: "refs/heads/main"}}},
		{"push +refs/heads/main:refs/heads/main", []pushSpec{{src: "refs/heads/main", dst: "refs/heads/main"}}},
		{"push :refs/heads/delete", []pushSpec{{dst: "refs/heads/delete"}}},
		{"push branch", []pushSpec{{dst: "branch"}}},
		{"list", nil},
		{"", nil},
	}
	for _, tt := range tests {
		got := parsePushLine(tt.line)
		if len(got) != len(tt.want) {
			t.Errorf("parsePushLine(%q) len = %d, want %d", tt.line, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].src != tt.want[i].src || got[i].dst != tt.want[i].dst {
				t.Errorf("parsePushLine(%q)[%d] = %+v, want %+v", tt.line, i, got[i], tt.want[i])
			}
		}
	}
}
