package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    int
	}{
		{"0.2.0", "v0.2.1", 1},
		{"0.2.1", "v0.2.0", -1},
		{"v0.2", "0.2.0", 0},
		{"0.3.0-dev", "v0.2.0", -1},
	}
	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.latest, func(t *testing.T) {
			if got := CompareVersions(tt.current, tt.latest); got != tt.want {
				t.Fatalf("CompareVersions() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSelectAsset(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: "paste_tool-installer-v1.0.0-windows-x64.exe"},
			{Name: "paste_tool-v1.0.0-linux-amd64.tar.gz"},
			{Name: "paste_tool-latest-windows-x64.exe"},
		},
	}
	asset, err := SelectAsset(release, "portable", "windows", "amd64")
	if err != nil {
		t.Fatalf("SelectAsset portable: %v", err)
	}
	if asset.Name != "paste_tool-latest-windows-x64.exe" {
		t.Fatalf("portable asset = %q", asset.Name)
	}
	asset, err = SelectAsset(release, "installer", "windows", "amd64")
	if err != nil {
		t.Fatalf("SelectAsset installer: %v", err)
	}
	if asset.Name != "paste_tool-installer-v1.0.0-windows-x64.exe" {
		t.Fatalf("installer asset = %q", asset.Name)
	}
}

func TestLatestParsesRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"tag_name":"v1.2.3","html_url":"https://example.test/release","assets":[{"name":"paste_tool-latest-windows-x64.exe","browser_download_url":"https://example.test/download"}]}`)
	}))
	defer server.Close()

	client := NewClient("owner/repo")
	client.BaseAPIURL = server.URL
	release, err := client.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if release.TagName != "v1.2.3" || len(release.Assets) != 1 {
		t.Fatalf("release = %#v", release)
	}
}
