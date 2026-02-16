package cache

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/anchore/go-logger"
	"github.com/spf13/afero"
)

// NewFromDir creates a new cache manager which returns caches stored on disk, rooted at the given directory
func NewFromDir(log logger.Logger, dir string, ttl time.Duration) (Manager, error) {
	dir = filepath.Clean(dir)
	fsys, err := subFs(afero.NewOsFs(), dir)
	if err != nil {
		return nil, err
	}
	return &filesystemCache{
		log: log,
		dir: dir,
		fs:  fsys,
		ttl: ttl,
	}, nil
}

const filePermissions = 0700
const dirPermissions = os.ModeDir | filePermissions

type filesystemCache struct {
	log logger.Logger
	dir string
	fs  afero.Fs
	ttl time.Duration
}

func (d *filesystemCache) GetCache(name, version string) Cache {
	fsys, err := subFs(d.fs, name, version)
	if err != nil {
		warnLog(d.log, "error getting cache", "name", name, "version", version, "error", err)
		return &bypassedCache{}
	}
	return &filesystemCache{
		log: d.log,
		dir: filepath.Join(d.dir, name, version),
		fs:  fsys,
		ttl: d.ttl,
	}
}

func (d *filesystemCache) RootDirs() []string {
	if d.dir == "" {
		return nil
	}
	return []string{d.dir}
}

func (d *filesystemCache) Read(key string) (ReaderAtCloser, error) {
	path := makeDiskKey(key)
	f, err := d.fs.Open(path)
	if err != nil {
		traceLog(d.log, "no cache entry", "dir", d.dir, "key", key, "error", err)
		return nil, errNotFound
	} else if stat, err := f.Stat(); err != nil || stat == nil || time.Since(stat.ModTime()) > d.ttl {
		traceLog(d.log, "cache entry is too old", "dir", d.dir, "key", key)
		return nil, errExpired
	}
	traceLog(d.log, "using value from cache", "dir", d.dir, "key", key)
	return f, nil
}

func (d *filesystemCache) Write(key string, contents io.Reader) error {
	path := makeDiskKey(key)
	return afero.WriteReader(d.fs, path, contents)
}

// subFs returns a writable directory with the given name under the root cache directory returned from findRoot,
// the directory will be created if it does not exist
func subFs(fsys afero.Fs, subDirs ...string) (afero.Fs, error) {
	dir := filepath.Join(subDirs...)
	dir = filepath.Clean(dir)
	stat, err := fsys.Stat(dir)
	if errors.Is(err, afero.ErrFileNotFound) {
		err = fsys.MkdirAll(dir, dirPermissions)
		if err != nil {
			return nil, fmt.Errorf("unable to create directory at '%s': %v", dir, err)
		}
		stat, err = fsys.Stat(dir)
		if err != nil {
			return nil, err
		}
	}
	if err != nil || stat == nil || !stat.IsDir() {
		return nil, fmt.Errorf("unable to verify directory '%s': %v", dir, err)
	}
	fsys = afero.NewBasePathFs(fsys, dir)
	return fsys, err
}

var keyReplacer = regexp.MustCompile("[^-._/a-zA-Z0-9]")

// makeDiskKey makes a safe sub-path but not escape forward slashes, this allows for logical partitioning on disk
func makeDiskKey(key string) string {
	// encode single dot directory
	if key == "." {
		return "%2E"
	}
	// replace any disallowed chars with encoded form
	key = keyReplacer.ReplaceAllStringFunc(key, url.QueryEscape)
	// allow . in names but not ..
	key = strings.ReplaceAll(key, "..", "%2E%2E")
	return key
}

var errNotFound = fmt.Errorf("not found")
var errExpired = fmt.Errorf("expired")
