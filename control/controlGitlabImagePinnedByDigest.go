package control

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/getplumber/plumber/collector"
	"github.com/getplumber/plumber/configuration"
	"github.com/sirupsen/logrus"
)

const ControlTypeGitlabImagePinnedByDigestVersion = "0.1.0"

var imageDigestPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(?:[+._-][A-Za-z][A-Za-z0-9]*)*:[0-9a-fA-F]{32,}$`)

// GitlabImagePinnedByDigestConf holds the configuration for digest pinning detection.
type GitlabImagePinnedByDigestConf struct {
	// Enabled controls whether this check runs.
	Enabled bool `json:"enabled"`
}

// GetConf loads configuration from PlumberConfig.
// Returns error if config is missing or incomplete.
func (p *GitlabImagePinnedByDigestConf) GetConf(plumberConfig *configuration.PlumberConfig) error {
	// Plumber config is required
	if plumberConfig == nil {
		return fmt.Errorf("Plumber config is required but not provided")
	}

	// Get control config from PlumberConfig
	imgConfig := plumberConfig.GetContainerImagesMustBePinnedByDigestConfig()
	if imgConfig == nil {
		return fmt.Errorf("containerImagesMustBePinnedByDigest control configuration is missing from .plumber.yaml config file")
	}

	// Check if enabled field is set
	if imgConfig.Enabled == nil {
		return fmt.Errorf("containerImagesMustBePinnedByDigest.enabled field is required in .plumber.yaml config file")
	}

	// Apply configuration
	p.Enabled = imgConfig.IsEnabled()

	l.WithField("enabled", p.Enabled).Debug("containerImagesMustBePinnedByDigest control configuration loaded from .plumber.yaml file")

	return nil
}

// GitlabImagePinnedByDigestMetrics holds metrics about digest pinning.
type GitlabImagePinnedByDigestMetrics struct {
	Total             uint `json:"total"`
	PinnedByDigest    uint `json:"pinnedByDigest"`
	NotPinnedByDigest uint `json:"notPinnedByDigest"`
	CiInvalid         uint `json:"ciInvalid"`
	CiMissing         uint `json:"ciMissing"`
}

// GitlabImagePinnedByDigestResult holds the result of the digest pinning control.
type GitlabImagePinnedByDigestResult struct {
	Issues     []GitlabPipelineImageIssueNotPinnedByDigest `json:"issues"`
	Metrics    GitlabImagePinnedByDigestMetrics            `json:"metrics"`
	Compliance float64                                     `json:"compliance"`
	Version    string                                      `json:"version"`
	CiValid    bool                                        `json:"ciValid"`
	CiMissing  bool                                        `json:"ciMissing"`
	Skipped    bool                                        `json:"skipped"`         // True if control was disabled
	Error      string                                      `json:"error,omitempty"` // Error message if data collection failed
}

////////////////////
// Control issues //
////////////////////

// GitlabPipelineImageIssueNotPinnedByDigest represents an image reference without an immutable digest.
type GitlabPipelineImageIssueNotPinnedByDigest struct {
	Link string `json:"link"`
	Job  string `json:"job"`
}

///////////////////////
// Control functions //
///////////////////////

func isImagePinnedByDigest(imageLink string) bool {
	link := strings.TrimSpace(imageLink)
	if link == "" {
		return false
	}

	lastAt := strings.LastIndex(link, "@")
	if lastAt <= 0 || lastAt >= len(link)-1 {
		return false
	}

	digest := link[lastAt+1:]
	return imageDigestPattern.MatchString(digest)
}

// Run executes the digest pinning control.
func (p *GitlabImagePinnedByDigestConf) Run(pipelineImageData *collector.GitlabPipelineImageData) *GitlabImagePinnedByDigestResult {
	l := l.WithFields(logrus.Fields{
		"control":        "GitlabImagePinnedByDigest",
		"controlVersion": ControlTypeGitlabImagePinnedByDigestVersion,
	})
	l.Info("Start image digest pinning control")

	result := &GitlabImagePinnedByDigestResult{
		Issues:     []GitlabPipelineImageIssueNotPinnedByDigest{},
		Metrics:    GitlabImagePinnedByDigestMetrics{},
		Compliance: 100.0,
		Version:    ControlTypeGitlabImagePinnedByDigestVersion,
		CiValid:    pipelineImageData.CiValid,
		CiMissing:  pipelineImageData.CiMissing,
		Skipped:    false,
	}

	// Check if control is enabled
	if !p.Enabled {
		l.Info("Image digest pinning control is disabled, skipping")
		result.Skipped = true
		return result
	}

	// If CI is invalid or missing, return early
	if !pipelineImageData.CiValid || pipelineImageData.CiMissing {
		result.Compliance = 0.0
		if !pipelineImageData.CiValid {
			result.Metrics.CiInvalid = 1
		}
		if pipelineImageData.CiMissing {
			result.Metrics.CiMissing = 1
		}
		return result
	}

	// Loop over all images and check digest pinning
	for _, image := range pipelineImageData.Images {
		if isImagePinnedByDigest(image.Link) {
			result.Metrics.PinnedByDigest++
			continue
		}

		issue := GitlabPipelineImageIssueNotPinnedByDigest{
			Link: image.Link,
			Job:  image.Job,
		}
		result.Issues = append(result.Issues, issue)
		result.Metrics.NotPinnedByDigest++
	}

	// Calculate compliance based on issues
	if len(result.Issues) > 0 {
		result.Compliance = 0.0
		l.WithField("issuesCount", len(result.Issues)).Debug("Found images not pinned by digest, setting compliance to 0")
	}

	// Set total metrics
	result.Metrics.Total = uint(len(pipelineImageData.Images))

	l.WithFields(logrus.Fields{
		"totalImages":         result.Metrics.Total,
		"pinnedByDigestCount": result.Metrics.PinnedByDigest,
		"notPinnedCount":      result.Metrics.NotPinnedByDigest,
		"compliance":          result.Compliance,
	}).Info("Image digest pinning control completed")

	return result
}
