package cmd

import "testing"

func TestReleaseAssetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tag     string
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{
			name:   "linux amd64",
			tag:    "v2.260225.2",
			goos:   "linux",
			goarch: "amd64",
			want:   "workit_2.260225.2_linux_amd64.tar.gz",
		},
		{
			name:   "darwin arm64",
			tag:    "v2.260225.2",
			goos:   "darwin",
			goarch: "arm64",
			want:   "workit_2.260225.2_darwin_arm64.tar.gz",
		},
		{
			name:   "windows arm64",
			tag:    "v2.260225.2",
			goos:   "windows",
			goarch: "arm64",
			want:   "workit_2.260225.2_windows_arm64.zip",
		},
		{
			name:    "unsupported architecture",
			tag:     "v2.260225.2",
			goos:    "linux",
			goarch:  "386",
			wantErr: true,
		},
		{
			name:    "unsupported OS",
			tag:     "v2.260225.2",
			goos:    "plan9",
			goarch:  "amd64",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := releaseAssetName(tc.tag, tc.goos, tc.goarch)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got asset %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("releaseAssetName error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("asset mismatch: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestFindAssetByName(t *testing.T) {
	t.Parallel()

	release := githubRelease{
		TagName: "v2.260225.2",
		Assets: []githubAsset{
			{Name: "checksums.txt", BrowserDownloadURL: "https://example/checksums"},
			{Name: "workit_2.260225.2_linux_amd64.tar.gz", BrowserDownloadURL: "https://example/linux"},
		},
	}

	asset, ok := findAssetByName(release, "workit_2.260225.2_linux_amd64.tar.gz")
	if !ok {
		t.Fatalf("expected asset to be found")
	}
	if asset.BrowserDownloadURL != "https://example/linux" {
		t.Fatalf("unexpected download URL %q", asset.BrowserDownloadURL)
	}

	if _, ok := findAssetByName(release, "missing.tar.gz"); ok {
		t.Fatalf("expected missing asset lookup to fail")
	}
}
