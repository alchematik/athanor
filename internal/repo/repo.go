package repo

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
	"strings"
)

type PluginType string

const (
	PluginTypeEmpty      = ""
	PluginTypeTranslator = "translator"
	PluginTypeProvider   = "provider"
)

func (p PluginType) String() string {
	return string(p)
}

type LockFile struct {
	Entries map[string]LockEntry
}

func (l *LockFile) Append(rootPath string, s Source, p Runtime) error {
	return nil
}

func (l *LockFile) Compare(rootPath string, s Source, p Runtime) (bool, error) {
	return false, nil
}

type LockEntry struct {
	ID       string
	Checksum string
}

// Runtime is runtime information about the requested plugin.
type Runtime struct {
	OS   string
	Arch string
}

type Source interface {
}

type Local struct {
	Path string
}

type GitHubRelease struct {
	RepoOwner string
	RepoName  string
	Name      string
}

func (g GitHubRelease) Download(ctx context.Context, rootDir string, t PluginType) (map[string]LockEntry, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", g.RepoOwner, g.RepoName, g.Name)

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

	downloadDir := filepath.Join(rootDir, t.String(), "github.com", g.RepoOwner, g.RepoName, g.Name)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
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

		downloadPath := filepath.Join(downloadDir, asset.Name)
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

	checksumFile, err := os.ReadFile(filepath.Join(downloadDir, "checksum.txt"))
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

	entries := map[string]LockEntry{}
	for _, asset := range releaseRes.Assets {
		if asset.Name == "checksum.txt" {
			continue
		}

		downloadPath := filepath.Join(downloadDir, asset.Name)
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

		tarFile, err := os.Create(filepath.Join(downloadDir, tarName))
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

			if hdr.Name != t.String() {
				continue
			}

			p := filepath.Join(downloadDir, name)
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
			id := filepath.Join(t.String(), "github.com", g.RepoOwner, g.RepoName, g.Name, name)
			entries[id] = LockEntry{
				ID:       id,
				Checksum: checksum,
			}

			if err := f.Close(); err != nil {
				return nil, err
			}
		}
	}

	return entries, nil
}

type gitHubReleaseResponse struct {
	Assets []gitHubReleaseAsset `json:"assets"`
}

type gitHubReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
