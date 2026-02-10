package control

import (
	"path"
	"strings"

	"github.com/getplumber/plumber/collector"
	"github.com/getplumber/plumber/configuration"
	"github.com/sirupsen/logrus"
)

const ControlTypeGitlabPipelineOriginRequiredTemplatesVersion = "0.1.0"

//////////////////
// Control conf //
//////////////////

// GitlabPipelineRequiredTemplatesConf holds the configuration for required templates check
type GitlabPipelineRequiredTemplatesConf struct {
	// Enabled controls whether this check runs
	Enabled bool `json:"enabled"`
	// DNF (Disjunctive Normal Form) format:
	// Outer array = OR (at least one group must be satisfied)
	// Inner array = AND (all templates in group must be present)
	// Example: [["go", "helm"], ["go_helm_unified"]] means:
	//   "must have (go AND helm) OR (go_helm_unified)"
	RequiredGroups [][]string `json:"requiredGroups"`
}

// GetConf loads configuration from PlumberConfig
func (p *GitlabPipelineRequiredTemplatesConf) GetConf(plumberConfig *configuration.PlumberConfig) error {
	if plumberConfig == nil {
		p.Enabled = false
		return nil
	}

	config := plumberConfig.GetPipelineMustIncludeTemplateConfig()
	if config == nil {
		p.Enabled = false
		return nil
	}

	p.Enabled = config.IsEnabled()

	// Resolve required groups from either 'required' expression or legacy 'requiredGroups'
	groups, err := config.GetResolvedRequiredGroups()
	if err != nil {
		return err
	}
	p.RequiredGroups = groups

	l.WithFields(logrus.Fields{
		"enabled":        p.Enabled,
		"requiredGroups": p.RequiredGroups,
		"hasExpression":  config.Required != "",
	}).Debug("pipelineMustIncludeTemplate control configuration loaded from .plumber.yaml file")

	return nil
}

////////////////////////////
// Control data & metrics //
////////////////////////////

// TemplateGroupStatus tracks the status of a single requirement group (AND clause)
type TemplateGroupStatus struct {
	GroupIndex       int      `json:"groupIndex"`       // Which requirement group (0-based)
	RequiredOrigins  []string `json:"requiredOrigins"`  // Templates required in this group
	FoundOrigins     []string `json:"foundOrigins"`     // Templates found
	MissingOrigins   []string `json:"missingOrigins"`   // Templates missing from this group
	IsFullySatisfied bool     `json:"isFullySatisfied"` // All templates in group present
}

// GitlabPipelineRequiredTemplatesMetrics holds metrics about required templates
type GitlabPipelineRequiredTemplatesMetrics struct {
	TotalGroups       uint `json:"totalGroups"`       // Total number of requirement groups
	SatisfiedGroups   uint `json:"satisfiedGroups"`   // Number of fully satisfied groups
	AnySatisfiedGroup bool `json:"anySatisfiedGroup"` // True if at least one group satisfied
	CiInvalid         uint `json:"ciInvalid"`
	CiMissing         uint `json:"ciMissing"`
}

// GitlabPipelineRequiredTemplatesResult holds the result of the required templates control
type GitlabPipelineRequiredTemplatesResult struct {
	RequirementGroups []TemplateGroupStatus                  `json:"requirementGroups"`
	Issues            []RequiredTemplateIssue                `json:"issues"`
	Metrics           GitlabPipelineRequiredTemplatesMetrics `json:"metrics"`
	Compliance        float64                                `json:"compliance"`
	Version           string                                 `json:"version"`
	CiValid           bool                                   `json:"ciValid"`
	CiMissing         bool                                   `json:"ciMissing"`
	Skipped           bool                                   `json:"skipped"`
	Error             string                                 `json:"error,omitempty"`
}

////////////////////
// Control issues //
////////////////////

// RequiredTemplateIssue represents an issue with a missing required template
type RequiredTemplateIssue struct {
	TemplatePath string `json:"templatePath"`
	GroupIndex   int    `json:"groupIndex"`
}

///////////////////////
// Control functions //
///////////////////////

// pathsMatch checks if two paths match (direct or normalized)
func pathsMatch(path1, path2 string) bool {
	// Direct match
	if path1 == path2 {
		return true
	}
	// Normalized path match
	return path.Clean(path1) == path.Clean(path2)
}

// Run executes the required templates control
func (p *GitlabPipelineRequiredTemplatesConf) Run(pipelineOriginData *collector.GitlabPipelineOriginData) *GitlabPipelineRequiredTemplatesResult {
	l := l.WithFields(logrus.Fields{
		"control":        "GitlabPipelineRequiredTemplates",
		"controlVersion": ControlTypeGitlabPipelineOriginRequiredTemplatesVersion,
	})
	l.Info("Start required templates control")

	result := &GitlabPipelineRequiredTemplatesResult{
		RequirementGroups: []TemplateGroupStatus{},
		Issues:            []RequiredTemplateIssue{},
		Metrics:           GitlabPipelineRequiredTemplatesMetrics{},
		Compliance:        0.0,
		Version:           ControlTypeGitlabPipelineOriginRequiredTemplatesVersion,
		CiValid:           pipelineOriginData.CiValid,
		CiMissing:         pipelineOriginData.CiMissing,
		Skipped:           false,
	}

	if !p.Enabled {
		l.Info("Required templates control is disabled, skipping")
		result.Skipped = true
		result.Compliance = 100.0
		return result
	}

	// Initialize metrics
	metrics := GitlabPipelineRequiredTemplatesMetrics{}

	if !pipelineOriginData.CiValid {
		metrics.CiInvalid = 1
	}
	if pipelineOriginData.CiMissing {
		metrics.CiMissing = 1
	}

	// Build set of template paths found in the pipeline
	// For project includes, use PlumberOrigin.Path if available
	foundTemplates := make(map[string]bool)
	for _, origin := range pipelineOriginData.Origins {
		// Check if it's a template (project include with Plumber/R2 origin)
		if origin.FromPlumber && origin.PlumberOrigin.Path != "" {
			foundTemplates[origin.PlumberOrigin.Path] = true
			l.WithField("templatePath", origin.PlumberOrigin.Path).Debug("Found template in pipeline (from PlumberOrigin.Path)")
		}
		// Also check GitlabIncludeOrigin for project includes
		if origin.OriginType == "project" && origin.GitlabIncludeOrigin.Location != "" {
			// Try to extract a meaningful path from the include
			templatePath := origin.GitlabIncludeOrigin.Location
			// Remove file extension for matching
			templatePath = strings.TrimSuffix(templatePath, ".yml")
			templatePath = strings.TrimSuffix(templatePath, ".yaml")
			foundTemplates[templatePath] = true
			l.WithField("templatePath", templatePath).Debug("Found template in pipeline (from GitlabIncludeOrigin)")
		}
	}

	// Check each requirement group
	result.RequirementGroups = make([]TemplateGroupStatus, len(p.RequiredGroups))
	anySatisfied := false

	for i, group := range p.RequiredGroups {
		groupStatus := TemplateGroupStatus{
			GroupIndex:       i,
			RequiredOrigins:  group,
			FoundOrigins:     []string{},
			MissingOrigins:   []string{},
			IsFullySatisfied: true,
		}

		for _, requiredTemplate := range group {
			found := false

			// Check if the required template matches any found template
			for foundPath := range foundTemplates {
				if pathsMatch(foundPath, requiredTemplate) {
					found = true
					break
				}
				// Also try partial match (template name might be specified without full path)
				if strings.HasSuffix(foundPath, "/"+requiredTemplate) || strings.HasSuffix(foundPath, requiredTemplate) {
					found = true
					break
				}
			}

			if found {
				groupStatus.FoundOrigins = append(groupStatus.FoundOrigins, requiredTemplate)
			} else {
				groupStatus.MissingOrigins = append(groupStatus.MissingOrigins, requiredTemplate)
				groupStatus.IsFullySatisfied = false

				// Create issue for missing template
				result.Issues = append(result.Issues, RequiredTemplateIssue{
					TemplatePath: requiredTemplate,
					GroupIndex:   i,
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
	}).Info("Required templates control completed")

	return result
}
