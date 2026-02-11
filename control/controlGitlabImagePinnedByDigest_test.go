package control

import (
	"strings"
	"testing"

	"github.com/getplumber/plumber/collector"
)

func TestIsImagePinnedByDigest(t *testing.T) {
	sha256Digest := strings.Repeat("a", 64)
	sha512Digest := strings.Repeat("b", 128)

	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{
			name:  "digest only",
			image: "docker.io/library/alpine@sha256:" + sha256Digest,
			want:  true,
		},
		{
			name:  "tag and digest",
			image: "docker.io/library/node:20@sha256:" + sha256Digest,
			want:  true,
		},
		{
			name:  "sha512 digest",
			image: "registry.example.com/team/app@sha512:" + sha512Digest,
			want:  true,
		},
		{
			name:  "tag only",
			image: "docker.io/library/alpine:3.19",
			want:  false,
		},
		{
			name:  "implicit latest",
			image: "docker.io/library/alpine",
			want:  false,
		},
		{
			name:  "digest variable",
			image: "docker.io/library/alpine@$DIGEST",
			want:  false,
		},
		{
			name:  "invalid digest format",
			image: "docker.io/library/alpine@sha256:not-a-hex-digest",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isImagePinnedByDigest(tt.image)
			if got != tt.want {
				t.Fatalf("isImagePinnedByDigest(%q) = %v, want %v", tt.image, got, tt.want)
			}
		})
	}
}

func TestGitlabImagePinnedByDigestRunEnabled(t *testing.T) {
	conf := &GitlabImagePinnedByDigestConf{
		Enabled: true,
	}

	sha256Digest := strings.Repeat("a", 64)

	data := &collector.GitlabPipelineImageData{
		CiValid:   true,
		CiMissing: false,
		Images: []collector.GitlabPipelineImageInfo{
			{
				Link: "docker.io/library/alpine@sha256:" + sha256Digest,
				Job:  "build",
			},
			{
				Link: "docker.io/library/node:20",
				Job:  "test",
			},
			{
				Link: "docker.io/library/golang",
				Job:  "lint",
			},
		},
	}

	result := conf.Run(data)

	if result.Skipped {
		t.Fatalf("expected control to run, but it was skipped")
	}
	if result.Compliance != 0 {
		t.Fatalf("expected compliance to be 0, got %v", result.Compliance)
	}
	if result.Metrics.Total != 3 {
		t.Fatalf("expected total metric to be 3, got %d", result.Metrics.Total)
	}
	if result.Metrics.PinnedByDigest != 1 {
		t.Fatalf("expected pinnedByDigest metric to be 1, got %d", result.Metrics.PinnedByDigest)
	}
	if result.Metrics.NotPinnedByDigest != 2 {
		t.Fatalf("expected notPinnedByDigest metric to be 2, got %d", result.Metrics.NotPinnedByDigest)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(result.Issues))
	}
}

func TestGitlabImagePinnedByDigestRunDisabled(t *testing.T) {
	conf := &GitlabImagePinnedByDigestConf{
		Enabled: false,
	}

	data := &collector.GitlabPipelineImageData{
		CiValid:   true,
		CiMissing: false,
		Images: []collector.GitlabPipelineImageInfo{
			{
				Link: "docker.io/library/node:20",
				Job:  "build",
			},
		},
	}

	result := conf.Run(data)

	if !result.Skipped {
		t.Fatalf("expected control to be skipped")
	}
	if result.Compliance != 100 {
		t.Fatalf("expected compliance to remain 100 when skipped, got %v", result.Compliance)
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues when skipped, got %d", len(result.Issues))
	}
}
