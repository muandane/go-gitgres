package backend

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		url      string
		wantConn string
		wantRepo string
		wantErr  bool
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
		conn, repo, err := ParseURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseURL(%q) err = %v, wantErr %v", tt.url, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (conn != tt.wantConn || repo != tt.wantRepo) {
			t.Errorf("ParseURL(%q) = %q, %q; want %q, %q", tt.url, conn, repo, tt.wantConn, tt.wantRepo)
		}
	}
}
