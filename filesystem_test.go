package cache

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anchore/go-logger/adapter/discard"
	"github.com/stretchr/testify/require"
)

func Test_filesystemCache(t *testing.T) {
	dir := t.TempDir()
	man, err := NewFromDir(discard.New(), dir, 1*time.Minute)
	require.NoError(t, err)

	cacheName := "test"
	cacheVersion := "v1"
	c := man.GetCache(cacheName, cacheVersion)

	cacheKey := "test-key"
	contentsValue := "some contents to cache"

	err = c.Write(cacheKey, strings.NewReader(contentsValue))
	require.NoError(t, err)

	rdr, err := c.Read(cacheKey)
	require.NoError(t, err)
	defer func() { _ = rdr.Close() }()

	contents, err := io.ReadAll(rdr)
	require.NoError(t, err)
	require.Equal(t, contentsValue, string(contents))

	// check the contents were actually written to disk as expected
	contents, err = os.ReadFile(filepath.Join(dir, cacheName, cacheVersion, cacheKey))
	require.NoError(t, err)
	require.Equal(t, contentsValue, string(contents))

	_, err = c.Read("otherKey")
	require.ErrorContains(t, err, "not found")
}

func Test_makeDiskKey(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{
			in:       "",
			expected: "",
		},
		{
			in:       ".",
			expected: "%2E",
		},
		{
			in:       "..",
			expected: "%2E%2E",
		},
		{
			in:       "github.com",
			expected: "github.com",
		},
		{
			in:       "../github.com",
			expected: "%2E%2E/github.com",
		},
		{
			in:       "github.com/../..",
			expected: "github.com/%2E%2E/%2E%2E",
		},
		{
			in:       "github.com/%2E../..",
			expected: "github.com/%252E%2E%2E/%2E%2E",
		},
	}
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			got := makeDiskKey(test.in)
			// validate appropriate escaping
			require.Equal(t, test.expected, got)
			// also validate that unescaped string matches original
			unescaped, err := url.QueryUnescape(got)
			require.NoError(t, err)
			require.Equal(t, test.in, unescaped)
		})
	}
}

func Test_errors(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache")
	// make a non-writable directory
	require.NoError(t, os.MkdirAll(cacheDir, 0500|os.ModeDir))
	// attempt to make cache in non-writable directory
	cacheDir = filepath.Join(cacheDir, "dir")
	_, err := NewFromDir(discard.New(), cacheDir, time.Hour)
	require.ErrorContains(t, err, fmt.Sprintf("unable to create directory at '%s':", cacheDir))
}
