/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"jp.zpw.openfriend/internal/util"
)

const (
	DefaultManifestURL = "https://api.zpw.jp/version/"
	DefaultManifestKey = "OpenFriend"
	DefaultRepo        = "zerozshare/OpenFriendMC"
	DownloadURLTpl     = "https://github.com/{repo}/releases/download/v{version}/openfriend-{os}-{arch}{ext}"
)

type Checker struct {
	ManifestURL string
	ManifestKey string
	Repo        string
	Current     string
	Logger      *slog.Logger
}

type Outcome struct {
	Latest    string
	Available bool
}

func (c *Checker) Check(ctx context.Context) (*Outcome, error) {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	if c.ManifestURL == "" {
		c.ManifestURL = DefaultManifestURL
	}
	if c.ManifestKey == "" {
		c.ManifestKey = DefaultManifestKey
	}
	if c.Repo == "" {
		c.Repo = DefaultRepo
	}

	latest, err := c.fetchLatest(ctx)
	if err != nil {
		return nil, err
	}
	out := &Outcome{Latest: strings.TrimPrefix(latest, "v")}
	if out.Latest == "" {
		c.Logger.Debug("Manifest does not list openfriend")
		return out, nil
	}
	if !isNewer(out.Latest, c.Current) {
		c.Logger.Debug("Update check: already on latest", "current", c.Current, "latest", out.Latest)
		return out, nil
	}
	out.Available = true
	c.Logger.Info("Update available", "current", c.Current, "latest", out.Latest)
	return out, nil
}

func (c *Checker) Apply(ctx context.Context, version string) (string, error) {
	if c.Repo == "" {
		c.Repo = DefaultRepo
	}
	dlURL := expandTemplate(c.Repo, version)
	c.Logger.Info("Downloading new binary", "url", dlURL)

	tmpPath, err := download(ctx, dlURL)
	if err != nil {
		return "", err
	}

	selfPath, err := selfExecutablePath()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := replaceSelf(selfPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	c.Logger.Info("Binary replaced", "path", selfPath)
	return selfPath, nil
}

func (c *Checker) fetchLatest(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.ManifestURL, nil)
	resp, err := util.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("manifest status %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("manifest JSON: %w", err)
	}
	v, _ := m[c.ManifestKey].(string)
	return strings.TrimSpace(v), nil
}

func download(ctx context.Context, url string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := util.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download status %d url=%s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "openfriend-update-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	tmp.Close()
	return tmpPath, nil
}

func expandTemplate(repo, version string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	r := strings.NewReplacer(
		"{repo}", repo,
		"{version}", version,
		"{os}", runtime.GOOS,
		"{arch}", runtime.GOARCH,
		"{ext}", ext,
	)
	return r.Replace(DownloadURLTpl)
}

func selfExecutablePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate self: %w", err)
	}
	abs, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p, nil
	}
	return abs, nil
}

func isNewer(latest, current string) bool {
	if latest == "" {
		return false
	}
	if current == "" || current == "dev" {
		return true
	}
	a := parseVersion(latest)
	b := parseVersion(current)
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return len(a) > len(b)
}

func parseVersion(s string) []int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out = append(out, n)
	}
	return out
}

var ErrUnsupportedPlatform = errors.New("self-replace not supported on this platform")
