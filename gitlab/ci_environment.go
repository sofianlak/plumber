package gitlab

import (
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// DetectMergeRequestIID checks if we are running inside a GitLab CI
// merge request pipeline and returns the merge request IID.
// Returns 0 if not in a CI merge request context.
//
// GitLab CI sets CI_MERGE_REQUEST_IID only for merge request pipelines
// (pipelines triggered by `rules: - if: $CI_MERGE_REQUEST_IID` or
// `only: merge_requests`).
func DetectMergeRequestIID() int {
	l := logrus.WithField("action", "DetectMergeRequestIID")

	// Must be running in CI
	if !IsRunningInCI() {
		l.Debug("Not running in CI")
		return 0
	}

	// Must have a merge request IID
	mrIIDStr := os.Getenv("CI_MERGE_REQUEST_IID")
	if mrIIDStr == "" {
		l.Debug("CI_MERGE_REQUEST_IID not set, not a merge request pipeline")
		return 0
	}

	mrIID, err := strconv.Atoi(strings.TrimSpace(mrIIDStr))
	if err != nil {
		l.WithError(err).WithField("CI_MERGE_REQUEST_IID", mrIIDStr).Warn("Invalid CI_MERGE_REQUEST_IID value")
		return 0
	}

	l.WithField("mergeRequestIID", mrIID).Info("Detected merge request CI environment")
	return mrIID
}

// IsRunningInCI checks if the code is running inside a GitLab CI environment
// by checking if the CI environment variable is set to "true"
func IsRunningInCI() bool {
	ciEnv := os.Getenv("CI")
	return strings.ToLower(ciEnv) == "true"
}

// IsOnDefaultBranchCI checks if the current CI pipeline is running on the
// project's default branch by comparing CI_COMMIT_BRANCH to CI_DEFAULT_BRANCH.
// Only call this when IsRunningInCI() returns true.
func IsOnDefaultBranchCI() bool {
	commitBranch := os.Getenv("CI_COMMIT_BRANCH")
	defaultBranch := os.Getenv("CI_DEFAULT_BRANCH")

	if commitBranch == "" || defaultBranch == "" {
		return false
	}

	return commitBranch == defaultBranch
}
