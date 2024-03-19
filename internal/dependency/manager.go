package dependency

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type lockFileBin struct {
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}
type lockFileEntry struct {
	ID   string        `json:"id"`
	Bins []lockFileBin `json:"checksums"`
}
type lockFile struct {
	Entries []lockFileEntry `json:"deps"`
}

type ManagerParams struct {
	LockFilePath string
	Upgrade      bool
	FetchRemote  bool
}

func NewManager(params ManagerParams) (*Manager, error) {
	var rawLockFile lockFile
	lockfileData, err := os.ReadFile(params.LockFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading lockfile: %s", err)
		}
	} else {
		if err := json.Unmarshal(lockfileData, &rawLockFile); err != nil {
			return nil, err
		}
	}

	lf := &LockFile{
		Bins: map[string]LockFileBinEntry{},
	}

	depLocks := map[string]*sync.Mutex{}
	for _, e := range rawLockFile.Entries {
		m := map[string]string{}
		for _, b := range e.Bins {
			m[b.Name] = b.Checksum
		}

		lf.Bins[e.ID] = LockFileBinEntry{
			Bins: m,
		}
		depLocks[e.ID] = &sync.Mutex{}
	}

	return &Manager{
		lockFilePath: params.LockFilePath,
		LockFile:     lf,
		Upgrade:      params.Upgrade,
		FetchRemote:  params.FetchRemote,
		Dir:          ".athanor",
		lock:         sync.Mutex{},
		depLocks:     depLocks,
	}, nil
}

type Manager struct {
	FetchRemote  bool
	Upgrade      bool
	LockFile     *LockFile
	Dir          string
	lockFilePath string

	lock     sync.Mutex
	depLocks map[string]*sync.Mutex
}

type BinDependency struct {
	Type   string
	Source any
	OS     string
	Arch   string
}

type SourceLocal struct {
	Path string
}

type SourceGitHubRelease struct {
	RepoOwner string
	RepoName  string
	Name      string
}

type LockFile struct {
	sync.RWMutex

	Bins map[string]LockFileBinEntry
}

type LockFileBinEntry struct {
	Bins map[string]string
}

func (m *Manager) FlushLockFile() error {
	if !m.Upgrade {
		return nil
	}

	result := lockFile{}
	for id, entry := range m.LockFile.Bins {
		var bins []lockFileBin
		for k, v := range entry.Bins {
			bins = append(bins, lockFileBin{
				Name:     k,
				Checksum: v,
			})
		}
		sort.Slice(bins, func(i, j int) bool {
			return bins[i].Name < bins[j].Name
		})
		result.Entries = append(result.Entries, lockFileEntry{
			ID:   id,
			Bins: bins,
		})
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		return result.Entries[i].ID < result.Entries[j].ID
	})

	resultData, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return os.WriteFile("athanor.lock.json", resultData, 0755)
}

func (m *Manager) IsBinDependencyInstalled(d BinDependency) (bool, error) {
	if _, isLocal := d.Source.(SourceLocal); isLocal {
		return true, nil
	}

	release, ok := d.Source.(SourceGitHubRelease)
	if !ok {
		return false, fmt.Errorf("unsupported source type: %T", d.Source)
	}

	id := filepath.Join(d.Type, "github.com", release.RepoOwner, release.RepoName, release.Name)
	m.LockFile.RLock()
	_, inLockFile := m.LockFile.Bins[id]
	m.LockFile.RUnlock()
	if !inLockFile {
		return false, nil
	}

	binName := fmt.Sprintf("%s_%s", d.OS, d.Arch)
	binDir := filepath.Join(m.Dir, d.Type, "github.com", release.RepoOwner, release.RepoName, release.Name)
	binPath := filepath.Join(binDir, binName)
	_, err := os.Stat(binPath)
	return err == nil, nil
}

func (m *Manager) FetchBinDependency(ctx context.Context, d BinDependency) (string, error) {
	if local, isLocal := d.Source.(SourceLocal); isLocal {
		return local.Path, nil
	}

	release, ok := d.Source.(SourceGitHubRelease)
	if !ok {
		return "", fmt.Errorf("unsupported source type: %T", d.Source)
	}

	id := filepath.Join(d.Type, "github.com", release.RepoOwner, release.RepoName, release.Name)

	m.lock.Lock()
	lock, ok := m.depLocks[id]
	if !ok {
		lock = &sync.Mutex{}
		m.depLocks[id] = lock
	}
	m.lock.Unlock()

	lock.Lock()
	defer lock.Unlock()

	binName := fmt.Sprintf("%s_%s", d.OS, d.Arch)
	binDir := filepath.Join(m.Dir, d.Type, "github.com", release.RepoOwner, release.RepoName, release.Name)
	binPath := filepath.Join(binDir, binName)

	var binExists bool
	f, err := os.Open(binPath)
	if err == nil {
		binExists = true
	}

	var lockChecksum string
	entry, ok := m.LockFile.Bins[id]
	if ok {
		lockChecksum = entry.Bins[binName]
	}

	var checksum string
	if binExists {
		hash := sha256.New()
		if _, err := io.Copy(hash, f); err != nil {
			return "", err
		}

		checksum = fmt.Sprintf("%x", hash.Sum(nil))

		if checksum == lockChecksum {
			return binPath, nil
		}
	}

	switch {
	case m.Upgrade:
		checksums, err := m.Download(ctx, binDir, release, d.Type)
		if err != nil {
			return "", err
		}

		m.LockFile.Bins[id] = LockFileBinEntry{
			Bins: checksums,
		}
	case m.FetchRemote:
		if lockChecksum == "" {
			return "", fmt.Errorf("checksum not found for %s", id)
		}

		checksums, err := m.Download(ctx, binDir, release, d.Type)
		if err != nil {
			return "", err
		}

		checksum, ok = checksums[binName]
		if !ok {
			return "", fmt.Errorf("%s not available in downloaded binaries", binName)
		}

		if lockChecksum != checksum {
			return "", fmt.Errorf("checksum mismatch for %s", id)
		}
	default:
		if !binExists {
			return "", fmt.Errorf("%s not installed", id)
		}

		if lockChecksum == "" {
			return "", fmt.Errorf("checksum not found for %s", id)
		}
	}

	return binPath, nil
}

type gitHubReleaseResponse struct {
	Assets []gitHubReleaseAsset `json:"assets"`
}

type gitHubReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (m *Manager) Download(ctx context.Context, dir string, s SourceGitHubRelease, t string) (map[string]string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", s.RepoOwner, s.RepoName, s.Name)

	releaseResRaw, err := http.Get(releaseURL)
	if err != nil {
		return nil, err
	}

	defer releaseResRaw.Body.Close()

	releaseResBody, err := io.ReadAll(releaseResRaw.Body)
	if err != nil {
		return nil, err
	}

	var releaseRes gitHubReleaseResponse
	if err := json.Unmarshal(releaseResBody, &releaseRes); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	client := &http.Client{}
	for _, asset := range releaseRes.Assets {
		req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/octet-stream")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer res.Body.Close()

		downloadPath := filepath.Join(dir, asset.Name)
		f, err := os.Create(downloadPath)
		if err != nil {
			return nil, err
		}

		if _, err := io.Copy(f, res.Body); err != nil {
			return nil, err
		}

		if err := f.Close(); err != nil {
			return nil, err
		}
	}

	checksumFile, err := os.ReadFile(filepath.Join(dir, "checksum.txt"))
	if err != nil {
		return nil, err
	}

	checksums := map[string]string{}
	for _, line := range strings.Split(string(checksumFile), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "  ")
		checksums[parts[1]] = parts[0]
	}

	entries := map[string]string{}
	for _, asset := range releaseRes.Assets {
		if asset.Name == "checksum.txt" {
			continue
		}

		downloadPath := filepath.Join(dir, asset.Name)
		expectedSum := checksums[asset.Name]

		assetFile, err := os.Open(downloadPath)
		if err != nil {
			return nil, err
		}

		hash := sha256.New()
		if _, err := io.Copy(hash, assetFile); err != nil {
			return nil, err
		}

		checksum := fmt.Sprintf("%x", hash.Sum(nil))
		if checksum != expectedSum {
			return nil, fmt.Errorf("checksum does not match: %v", asset)
		}

		if _, err := assetFile.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		gzipReader, err := gzip.NewReader(assetFile)
		if err != nil {
			return nil, err
		}

		ext := filepath.Ext(asset.Name)
		tarName := asset.Name[:len(asset.Name)-len(ext)]

		tarFile, err := os.Create(filepath.Join(dir, tarName))
		if err != nil {
			return nil, err
		}

		if _, err := io.Copy(tarFile, gzipReader); err != nil {
			return nil, err
		}

		if err := gzipReader.Close(); err != nil {
			return nil, err
		}

		if _, err := tarFile.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		ext = filepath.Ext(tarName)
		name := tarName[:len(tarName)-len(ext)]

		tarReader := tar.NewReader(tarFile)
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			// Binary must be named either "provider" or "translater", i.e. the type of plugin.
			if hdr.Name != t {
				continue
			}

			p := filepath.Join(dir, name)
			f, err := os.Create(p)
			if err != nil {
				return nil, err
			}

			if _, err := io.Copy(f, tarReader); err != nil {
				return nil, err
			}

			if err := f.Chmod(0755); err != nil {
				return nil, err
			}

			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}

			hash := sha256.New()
			if _, err := io.Copy(hash, f); err != nil {
				return nil, err
			}

			checksum := fmt.Sprintf("%x", hash.Sum(nil))
			entries[name] = checksum

			if err := f.Close(); err != nil {
				return nil, err
			}
		}
	}

	return entries, nil
}
