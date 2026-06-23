package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mai-xiyu/Paste-Tool/internal/metadata"
)

const defaultTimeout = 15 * time.Second

type Client struct {
	Repository string
	BaseAPIURL string
	HTTPClient *http.Client
	UserAgent  string
}

type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

func NewClient(repository string) Client {
	if strings.TrimSpace(repository) == "" {
		repository = "Mai-xiyu/Paste-Tool"
	}
	return Client{
		Repository: repository,
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		UserAgent:  metadata.Name + "/" + metadata.Version,
	}
}

func (c Client) Latest(ctx context.Context) (Release, error) {
	baseURL := strings.TrimRight(c.BaseAPIURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	apiURL := baseURL + "/repos/" + strings.Trim(c.Repository, "/") + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent())

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Release{}, fmt.Errorf("fetch latest release: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("decode latest release: %w", err)
	}
	if release.TagName == "" {
		return Release{}, errors.New("latest release does not contain tag_name")
	}
	return release, nil
}

func (c Client) Download(ctx context.Context, asset Asset, outputDir string) (string, error) {
	if asset.DownloadURL == "" {
		return "", errors.New("asset download URL is empty")
	}
	if outputDir == "" {
		var err error
		outputDir, err = DefaultDownloadDir()
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	outputPath := filepath.Join(outputDir, filepath.Base(asset.Name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent())
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("download %q: %w", asset.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download %q: status %d", asset.Name, resp.StatusCode)
	}

	tmp := outputPath + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	_, copyErr := io.Copy(file, resp.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("write output file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("close output file: %w", closeErr)
	}
	if err := os.Rename(tmp, outputPath); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("replace output file: %w", err)
	}
	return outputPath, nil
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultTimeout}
}

func (c Client) userAgent() string {
	if strings.TrimSpace(c.UserAgent) != "" {
		return c.UserAgent
	}
	return metadata.Name + "/" + metadata.Version
}

func DefaultDownloadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, "Downloads"), nil
}

func SelectAsset(release Release, kind, goos, goarch string) (Asset, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		kind = "portable"
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	var candidates []Asset
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if kind == "installer" {
			if !strings.Contains(name, "installer") {
				continue
			}
		} else {
			if strings.Contains(name, "installer") {
				continue
			}
		}
		candidates = append(candidates, asset)
	}
	if len(candidates) == 0 {
		return Asset{}, fmt.Errorf("release %s has no %s asset", release.TagName, kind)
	}

	osNames := osAliases(goos)
	archNames := archAliases(goarch)
	for _, asset := range candidates {
		name := strings.ToLower(asset.Name)
		if containsAny(name, osNames) && containsAny(name, archNames) {
			return asset, nil
		}
	}
	for _, asset := range candidates {
		if strings.Contains(strings.ToLower(asset.Name), "latest") {
			return asset, nil
		}
	}
	return candidates[0], nil
}

func CompareVersions(current, latest string) int {
	c := parseVersion(current)
	l := parseVersion(latest)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return 1
		}
		if l[i] < c[i] {
			return -1
		}
	}
	return 0
}

func HasUpdate(current string, release Release) bool {
	return CompareVersions(current, release.TagName) > 0
}

var versionRe = regexp.MustCompile(`(?i)^v?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?`)

func parseVersion(value string) [3]int {
	var out [3]int
	matches := versionRe.FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) == 0 {
		return out
	}
	for i := 1; i <= 3; i++ {
		if matches[i] == "" {
			continue
		}
		n, _ := strconv.Atoi(matches[i])
		out[i-1] = n
	}
	return out
}

func osAliases(goos string) []string {
	switch goos {
	case "windows":
		return []string{"windows", "win"}
	case "darwin":
		return []string{"darwin", "macos", "mac"}
	case "linux":
		return []string{"linux"}
	default:
		return []string{goos}
	}
}

func archAliases(goarch string) []string {
	switch goarch {
	case "amd64":
		return []string{"amd64", "x64", "x86_64"}
	case "arm64":
		return []string{"arm64", "aarch64"}
	default:
		return []string{goarch}
	}
}

func containsAny(value string, options []string) bool {
	for _, option := range options {
		if option != "" && strings.Contains(value, option) {
			return true
		}
	}
	return false
}
