package configuration

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// PlumberConfig represents the .plumber.yaml configuration file structure
type PlumberConfig struct {
	// Version of the config file format
	Version string `yaml:"version"`

	// Controls configuration
	Controls ControlsConfig `yaml:"controls"`
}

// ControlsConfig holds configuration for all controls
type ControlsConfig struct {
	// ContainerImageMustNotUseForbiddenTags control configuration
	ContainerImageMustNotUseForbiddenTags *ImageForbiddenTagsControlConfig `yaml:"containerImageMustNotUseForbiddenTags,omitempty"`

	// ContainerImagesMustBePinnedByDigest control configuration
	ContainerImagesMustBePinnedByDigest *ImagePinnedByDigestControlConfig `yaml:"containerImagesMustBePinnedByDigest,omitempty"`

	// ContainerImageMustComeFromAuthorizedSources control configuration
	ContainerImageMustComeFromAuthorizedSources *ImageAuthorizedSourcesControlConfig `yaml:"containerImageMustComeFromAuthorizedSources,omitempty"`

	// BranchMustBeProtected control configuration
	BranchMustBeProtected *BranchProtectionControlConfig `yaml:"branchMustBeProtected,omitempty"`

	// PipelineMustNotIncludeHardcodedJobs control configuration
	PipelineMustNotIncludeHardcodedJobs *HardcodedJobsControlConfig `yaml:"pipelineMustNotIncludeHardcodedJobs,omitempty"`

	// IncludesMustBeUpToDate control configuration
	IncludesMustBeUpToDate *IncludesUpToDateControlConfig `yaml:"includesMustBeUpToDate,omitempty"`

	// IncludesMustNotUseForbiddenVersions control configuration
	IncludesMustNotUseForbiddenVersions *IncludesForbiddenVersionsControlConfig `yaml:"includesMustNotUseForbiddenVersions,omitempty"`

	// PipelineMustIncludeComponent control configuration
	PipelineMustIncludeComponent *RequiredComponentsControlConfig `yaml:"pipelineMustIncludeComponent,omitempty"`

	// PipelineMustIncludeTemplate control configuration
	PipelineMustIncludeTemplate *RequiredTemplatesControlConfig `yaml:"pipelineMustIncludeTemplate,omitempty"`
}

// ImageForbiddenTagsControlConfig configuration for the forbidden image tags control
type ImageForbiddenTagsControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// Tags is a list of forbidden tags (e.g., latest, dev)
	Tags []string `yaml:"tags,omitempty"`
}

// ImagePinnedByDigestControlConfig configuration for the digest pinning control
type ImagePinnedByDigestControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`
}

// ImageAuthorizedSourcesControlConfig configuration for the authorized image sources control
type ImageAuthorizedSourcesControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// TrustedUrls is a list of trusted registry URLs/patterns (supports wildcards)
	TrustedUrls []string `yaml:"trustedUrls,omitempty"`

	// TrustDockerHubOfficialImages trusts official Docker Hub images (e.g., nginx, alpine)
	TrustDockerHubOfficialImages *bool `yaml:"trustDockerHubOfficialImages,omitempty"`
}

// BranchProtectionControlConfig configuration for the branch protection control
type BranchProtectionControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// NamePatterns is a list of branch name patterns that must be protected (supports wildcards)
	NamePatterns []string `yaml:"namePatterns,omitempty"`

	// DefaultMustBeProtected requires the default branch to be protected
	DefaultMustBeProtected *bool `yaml:"defaultMustBeProtected,omitempty"`

	// AllowForcePush when false, force push must be disabled on protected branches
	AllowForcePush *bool `yaml:"allowForcePush,omitempty"`

	// CodeOwnerApprovalRequired when true, code owner approval is required
	CodeOwnerApprovalRequired *bool `yaml:"codeOwnerApprovalRequired,omitempty"`

	// MinMergeAccessLevel minimum access level required to merge (0=No one, 30=Developer, 40=Maintainer)
	MinMergeAccessLevel *int `yaml:"minMergeAccessLevel,omitempty"`

	// MinPushAccessLevel minimum access level required to push (0=No one, 30=Developer, 40=Maintainer)
	MinPushAccessLevel *int `yaml:"minPushAccessLevel,omitempty"`
}

// HardcodedJobsControlConfig configuration for the hardcoded jobs control
type HardcodedJobsControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`
}

// IncludesUpToDateControlConfig configuration for the includes up-to-date control
type IncludesUpToDateControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`
}

// IncludesForbiddenVersionsControlConfig configuration for the forbidden versions control
type IncludesForbiddenVersionsControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// ForbiddenVersions is a list of version patterns considered forbidden (e.g., latest, main, HEAD)
	ForbiddenVersions []string `yaml:"forbiddenVersions,omitempty"`

	// DefaultBranchIsForbiddenVersion when true, adds the project's default branch to forbidden versions
	DefaultBranchIsForbiddenVersion *bool `yaml:"defaultBranchIsForbiddenVersion,omitempty"`
}

// RequiredComponentsControlConfig configuration for the required components control
type RequiredComponentsControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// Required is a human-readable boolean expression defining required components.
	// Supports AND, OR operators and parentheses for grouping.
	// AND has higher precedence than OR.
	//
	// Examples:
	//   "components/sast/sast AND components/secret-detection/secret-detection"
	//   "(components/sast/sast AND components/secret-detection/secret-detection) OR your-org/full-security/full-security"
	Required string `yaml:"required,omitempty"`

	// RequiredGroups uses DNF (Disjunctive Normal Form) format:
	// Outer array = OR (at least one group must be satisfied)
	// Inner array = AND (all components in group must be present)
	// Example: [["comp-a", "comp-b"], ["comp-c"]] means:
	//   "must have (comp-a AND comp-b) OR (comp-c)"
	//
	// Cannot be used together with 'required'.
	RequiredGroups [][]string `yaml:"requiredGroups,omitempty"`
}

// GetResolvedRequiredGroups returns the effective required groups by resolving
// either the 'required' expression or the 'requiredGroups' field.
// Returns an error if both are set or if the expression is invalid.
func (c *RequiredComponentsControlConfig) GetResolvedRequiredGroups() ([][]string, error) {
	if c == nil {
		return nil, nil
	}
	hasExpression := c.Required != ""
	hasGroups := len(c.RequiredGroups) > 0

	if hasExpression && hasGroups {
		return nil, fmt.Errorf("pipelineMustIncludeComponent: cannot use both 'required' and 'requiredGroups' — use only one")
	}
	if hasExpression {
		groups, err := ParseRequiredExpression(c.Required)
		if err != nil {
			return nil, fmt.Errorf("pipelineMustIncludeComponent: %w", err)
		}
		return groups, nil
	}
	return c.RequiredGroups, nil
}

// RequiredTemplatesControlConfig configuration for the required templates control
type RequiredTemplatesControlConfig struct {
	// Enabled controls whether this check runs
	Enabled *bool `yaml:"enabled,omitempty"`

	// Required is a human-readable boolean expression defining required templates.
	// Supports AND, OR operators and parentheses for grouping.
	// AND has higher precedence than OR.
	//
	// Examples:
	//   "templates/go/go AND templates/trivy/trivy"
	//   "(templates/go/go AND templates/trivy/trivy) OR templates/full-go-pipeline"
	Required string `yaml:"required,omitempty"`

	// RequiredGroups uses DNF (Disjunctive Normal Form) format:
	// Outer array = OR (at least one group must be satisfied)
	// Inner array = AND (all templates in group must be present)
	// Example: [["go", "helm"], ["go_helm_unified"]] means:
	//   "must have (go AND helm) OR (go_helm_unified)"
	//
	// Cannot be used together with 'required'.
	RequiredGroups [][]string `yaml:"requiredGroups,omitempty"`
}

// GetResolvedRequiredGroups returns the effective required groups by resolving
// either the 'required' expression or the 'requiredGroups' field.
// Returns an error if both are set or if the expression is invalid.
func (c *RequiredTemplatesControlConfig) GetResolvedRequiredGroups() ([][]string, error) {
	if c == nil {
		return nil, nil
	}
	hasExpression := c.Required != ""
	hasGroups := len(c.RequiredGroups) > 0

	if hasExpression && hasGroups {
		return nil, fmt.Errorf("pipelineMustIncludeTemplate: cannot use both 'required' and 'requiredGroups' — use only one")
	}
	if hasExpression {
		groups, err := ParseRequiredExpression(c.Required)
		if err != nil {
			return nil, fmt.Errorf("pipelineMustIncludeTemplate: %w", err)
		}
		return groups, nil
	}
	return c.RequiredGroups, nil
}

// LoadPlumberConfig loads configuration from a file path
// The config file path is required - returns error if empty or not found
func LoadPlumberConfig(configPath string) (*PlumberConfig, string, error) {
	l := logrus.WithField("action", "LoadPlumberConfig")

	if configPath == "" {
		return nil, "", fmt.Errorf("config file path is required")
	}

	l = l.WithField("configPath", configPath)
	l.Info("Loading configuration from file")

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, configPath, fmt.Errorf("config file not found: %s", configPath)
		}
		l.WithError(err).Error("Failed to read config file")
		return nil, configPath, err
	}

	// Parse YAML
	config := &PlumberConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		l.WithError(err).Error("Failed to parse config file")
		return nil, configPath, err
	}

	// Validate expression fields early to catch syntax errors at load time
	if err := config.validate(); err != nil {
		return nil, configPath, fmt.Errorf("configuration validation error: %w", err)
	}

	l.WithField("config", config).Debug("Configuration loaded successfully")
	return config, configPath, nil
}

// validate checks for configuration errors, including expression syntax
// and mutual exclusivity of 'required' vs 'requiredGroups'.
func (c *PlumberConfig) validate() error {
	if c == nil {
		return nil
	}

	// Validate pipelineMustIncludeComponent
	if comp := c.Controls.PipelineMustIncludeComponent; comp != nil {
		if _, err := comp.GetResolvedRequiredGroups(); err != nil {
			return err
		}
	}

	// Validate pipelineMustIncludeTemplate
	if tmpl := c.Controls.PipelineMustIncludeTemplate; tmpl != nil {
		if _, err := tmpl.GetResolvedRequiredGroups(); err != nil {
			return err
		}
	}

	return nil
}

// GetContainerImageMustNotUseForbiddenTagsConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetContainerImageMustNotUseForbiddenTagsConfig() *ImageForbiddenTagsControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.ContainerImageMustNotUseForbiddenTags
}

// GetContainerImagesMustBePinnedByDigestConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetContainerImagesMustBePinnedByDigestConfig() *ImagePinnedByDigestControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.ContainerImagesMustBePinnedByDigest
}

// GetContainerImageMustComeFromAuthorizedSourcesConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetContainerImageMustComeFromAuthorizedSourcesConfig() *ImageAuthorizedSourcesControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.ContainerImageMustComeFromAuthorizedSources
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *ImageForbiddenTagsControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *ImagePinnedByDigestControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *ImageAuthorizedSourcesControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetBranchMustBeProtectedConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetBranchMustBeProtectedConfig() *BranchProtectionControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.BranchMustBeProtected
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *BranchProtectionControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetPipelineMustNotIncludeHardcodedJobsConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetPipelineMustNotIncludeHardcodedJobsConfig() *HardcodedJobsControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.PipelineMustNotIncludeHardcodedJobs
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *HardcodedJobsControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetIncludesMustBeUpToDateConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetIncludesMustBeUpToDateConfig() *IncludesUpToDateControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.IncludesMustBeUpToDate
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *IncludesUpToDateControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetIncludesMustNotUseForbiddenVersionsConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetIncludesMustNotUseForbiddenVersionsConfig() *IncludesForbiddenVersionsControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.IncludesMustNotUseForbiddenVersions
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *IncludesForbiddenVersionsControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetPipelineMustIncludeComponentConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetPipelineMustIncludeComponentConfig() *RequiredComponentsControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.PipelineMustIncludeComponent
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *RequiredComponentsControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetPipelineMustIncludeTemplateConfig returns the control configuration
// Returns nil if not configured
func (c *PlumberConfig) GetPipelineMustIncludeTemplateConfig() *RequiredTemplatesControlConfig {
	if c == nil {
		return nil
	}
	return c.Controls.PipelineMustIncludeTemplate
}

// IsEnabled returns whether the control is enabled
// Returns false if not properly configured
func (c *RequiredTemplatesControlConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}
