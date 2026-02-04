package control

import (
	"strings"

	"github.com/getplumber/plumber/collector"
	"github.com/getplumber/plumber/configuration"
	"github.com/sirupsen/logrus"
)

const ControlTypeGitlabPipelineOriginRequiredComponentsVersion = "0.1.0"

//////////////////
// Control conf //
//////////////////

// GitlabPipelineRequiredComponentsConf holds the configuration for required components check
type GitlabPipelineRequiredComponentsConf struct {
	// Enabled controls whether this check runs
	Enabled bool `json:"enabled"`
	// DNF (Disjunctive Normal Form) format:
	// Outer array = OR (at least one group must be satisfied)
	// Inner array = AND (all components in group must be present)
	// Example: [["comp-a", "comp-b"], ["comp-c"]] means:
	//   "must have (comp-a AND comp-b) OR (comp-c)"
	RequiredGroups [][]string `json:"requiredGroups"`
}

// GetConf loads configuration from PlumberConfig
func (p *GitlabPipelineRequiredComponentsConf) GetConf(plumberConfig *configuration.PlumberConfig) error {
	if plumberConfig == nil {
		p.Enabled = false
		return nil
	}

	config := plumberConfig.GetPipelineMustIncludeComponentConfig()
	if config == nil {
		p.Enabled = false
		return nil
	}

	p.Enabled = config.IsEnabled()
	p.RequiredGroups = config.RequiredGroups

	l.WithFields(logrus.Fields{
		"enabled":        p.Enabled,
		"requiredGroups": p.RequiredGroups,
	}).Debug("pipelineMustIncludeComponent control configuration loaded from .plumber.yaml file")

	return nil
}

////////////////////////////
// Control data & metrics //
////////////////////////////

// ComponentGroupStatus tracks the status of a single requirement group (AND clause)
type ComponentGroupStatus struct {
	GroupIndex       int      `json:"groupIndex"`       // Which requirement group (0-based)
	RequiredOrigins  []string `json:"requiredOrigins"`  // Components required in this group
	FoundOrigins     []string `json:"foundOrigins"`     // Components found
	MissingOrigins   []string `json:"missingOrigins"`   // Components missing from this group
	IsFullySatisfied bool     `json:"isFullySatisfied"` // All components in group present
}

// GitlabPipelineRequiredComponentsMetrics holds metrics about required components
type GitlabPipelineRequiredComponentsMetrics struct {
	TotalGroups       uint `json:"totalGroups"`       // Total number of requirement groups
	SatisfiedGroups   uint `json:"satisfiedGroups"`   // Number of fully satisfied groups
	AnySatisfiedGroup bool `json:"anySatisfiedGroup"` // True if at least one group satisfied
	CiInvalid         uint `json:"ciInvalid"`
	CiMissing         uint `json:"ciMissing"`
}

// GitlabPipelineRequiredComponentsResult holds the result of the required components control
type GitlabPipelineRequiredComponentsResult struct {
	RequirementGroups []ComponentGroupStatus                  `json:"requirementGroups"`
	Issues            []RequiredComponentIssue                `json:"issues"`
	Metrics           GitlabPipelineRequiredComponentsMetrics `json:"metrics"`
	Compliance        float64                                 `json:"compliance"`
	Version           string                                  `json:"version"`
	CiValid           bool                                    `json:"ciValid"`
	CiMissing         bool                                    `json:"ciMissing"`
	Skipped           bool                                    `json:"skipped"`
	Error             string                                  `json:"error,omitempty"`
}

////////////////////
// Control issues //
////////////////////

// RequiredComponentIssue represents an issue with a missing required component
type RequiredComponentIssue struct {
	ComponentPath string `json:"componentPath"`
	GroupIndex    int    `json:"groupIndex"`
}

///////////////////////
// Control functions //
///////////////////////

// parseComponentPath extracts the clean component path without version and instance URL
func parseComponentPath(componentLocation string) string {
	// Remove version (everything after @)
	if idx := strings.LastIndex(componentLocation, "@"); idx != -1 {
		componentLocation = componentLocation[:idx]
	}
	// Remove common instance prefixes
	componentLocation = strings.TrimPrefix(componentLocation, "$CI_SERVER_FQDN/")
	componentLocation = strings.TrimPrefix(componentLocation, "$CI_SERVER_HOST/")
	// Remove any gitlab.com or other instance URLs
	if idx := strings.Index(componentLocation, "/"); idx != -1 {
		parts := strings.SplitN(componentLocation, "/", 2)
		if len(parts) == 2 && strings.Contains(parts[0], ".") {
			// First part looks like a domain, remove it
			componentLocation = parts[1]
		}
	}
	return componentLocation
}

// Run executes the required components control
func (p *GitlabPipelineRequiredComponentsConf) Run(pipelineOriginData *collector.GitlabPipelineOriginData, gitlabURL string) *GitlabPipelineRequiredComponentsResult {
	l := l.WithFields(logrus.Fields{
		"control":        "GitlabPipelineRequiredComponents",
		"controlVersion": ControlTypeGitlabPipelineOriginRequiredComponentsVersion,
	})
	l.Info("Start required components control")

	result := &GitlabPipelineRequiredComponentsResult{
		RequirementGroups: []ComponentGroupStatus{},
		Issues:            []RequiredComponentIssue{},
		Metrics:           GitlabPipelineRequiredComponentsMetrics{},
		Compliance:        0.0,
		Version:           ControlTypeGitlabPipelineOriginRequiredComponentsVersion,
		CiValid:           pipelineOriginData.CiValid,
		CiMissing:         pipelineOriginData.CiMissing,
		Skipped:           false,
	}

	if !p.Enabled {
		l.Info("Required components control is disabled, skipping")
		result.Skipped = true
		result.Compliance = 100.0
		return result
	}

	// Initialize metrics
	metrics := GitlabPipelineRequiredComponentsMetrics{}

	if !pipelineOriginData.CiValid {
		metrics.CiInvalid = 1
	}
	if pipelineOriginData.CiMissing {
		metrics.CiMissing = 1
	}

	// Build set of component paths found in the pipeline
	foundComponents := make(map[string]bool)
	for _, origin := range pipelineOriginData.Origins {
		if origin.OriginType == "component" {
			cleanPath := parseComponentPath(origin.GitlabIncludeOrigin.Location)
			foundComponents[cleanPath] = true
			l.WithField("componentPath", cleanPath).Debug("Found component in pipeline")
		}
	}

	// Check each requirement group
	result.RequirementGroups = make([]ComponentGroupStatus, len(p.RequiredGroups))
	anySatisfied := false

	for i, group := range p.RequiredGroups {
		groupStatus := ComponentGroupStatus{
			GroupIndex:       i,
			RequiredOrigins:  group,
			FoundOrigins:     []string{},
			MissingOrigins:   []string{},
			IsFullySatisfied: true,
		}

		for _, requiredComponent := range group {
			cleanRequired := parseComponentPath(requiredComponent)

			if foundComponents[cleanRequired] {
				groupStatus.FoundOrigins = append(groupStatus.FoundOrigins, requiredComponent)
			} else {
				groupStatus.MissingOrigins = append(groupStatus.MissingOrigins, requiredComponent)
				groupStatus.IsFullySatisfied = false

				// Create issue for missing component
				result.Issues = append(result.Issues, RequiredComponentIssue{
					ComponentPath: requiredComponent,
					GroupIndex:    i,
				})
			}
		}

		if groupStatus.IsFullySatisfied {
			anySatisfied = true
			metrics.SatisfiedGroups++
		}

		result.RequirementGroups[i] = groupStatus
	}

	// Calculate metrics
	metrics.TotalGroups = uint(len(p.RequiredGroups))
	metrics.AnySatisfiedGroup = anySatisfied

	// Calculate compliance using DNF logic
	// 100% if at least one group is fully satisfied, 0% otherwise (for simplicity)
	// More nuanced: find the group with highest completion percentage
	if len(p.RequiredGroups) == 0 {
		result.Compliance = 100.0
	} else if anySatisfied {
		result.Compliance = 100.0
	} else {
		// Find the group with highest completion
		maxScore := 0.0
		for _, group := range result.RequirementGroups {
			if len(group.RequiredOrigins) > 0 {
				score := float64(len(group.FoundOrigins)) / float64(len(group.RequiredOrigins))
				if score > maxScore {
					maxScore = score
				}
			}
		}
		result.Compliance = maxScore * 100.0
	}

	result.Metrics = metrics

	l.WithFields(logrus.Fields{
		"totalGroups":     metrics.TotalGroups,
		"satisfiedGroups": metrics.SatisfiedGroups,
		"compliance":      result.Compliance,
		"issueCount":      len(result.Issues),
	}).Info("Required components control completed")

	return result
}
