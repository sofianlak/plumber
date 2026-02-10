package configuration

import (
	"reflect"
	"testing"
)

func TestParseRequiredExpression(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected [][]string
		wantErr  bool
	}{
		// ── Empty / blank ──────────────────────────────────────────────
		{
			name:     "empty string",
			expr:     "",
			expected: [][]string{},
		},
		{
			name:     "whitespace only",
			expr:     "   \t  ",
			expected: [][]string{},
		},

		// ── Single identifier ──────────────────────────────────────────
		{
			name:     "single component",
			expr:     "components/sast/sast",
			expected: [][]string{{"components/sast/sast"}},
		},

		// ── AND ────────────────────────────────────────────────────────
		{
			name:     "two items AND",
			expr:     "components/sast/sast AND components/secret-detection/secret-detection",
			expected: [][]string{{"components/sast/sast", "components/secret-detection/secret-detection"}},
		},
		{
			name:     "three items AND",
			expr:     "a AND b AND c",
			expected: [][]string{{"a", "b", "c"}},
		},

		// ── OR ─────────────────────────────────────────────────────────
		{
			name:     "two items OR",
			expr:     "comp/a OR comp/b",
			expected: [][]string{{"comp/a"}, {"comp/b"}},
		},
		{
			name:     "three items OR",
			expr:     "a OR b OR c",
			expected: [][]string{{"a"}, {"b"}, {"c"}},
		},

		// ── AND has higher precedence than OR ──────────────────────────
		{
			name:     "AND then OR (precedence)",
			expr:     "a AND b OR c",
			expected: [][]string{{"a", "b"}, {"c"}},
		},
		{
			name:     "OR then AND (precedence)",
			expr:     "a OR b AND c",
			expected: [][]string{{"a"}, {"b", "c"}},
		},
		{
			name:     "mixed precedence complex",
			expr:     "a AND b OR c AND d",
			expected: [][]string{{"a", "b"}, {"c", "d"}},
		},

		// ── Parentheses override precedence ────────────────────────────
		{
			name:     "parens override: AND over OR",
			expr:     "a AND (b OR c)",
			expected: [][]string{{"a", "b"}, {"a", "c"}},
		},
		{
			name:     "parens group OR inside AND",
			expr:     "(a OR b) AND (c OR d)",
			expected: [][]string{{"a", "c"}, {"a", "d"}, {"b", "c"}, {"b", "d"}},
		},
		{
			name:     "parens with simple group",
			expr:     "(a AND b) OR c",
			expected: [][]string{{"a", "b"}, {"c"}},
		},
		{
			name:     "nested parens",
			expr:     "((a OR b)) AND c",
			expected: [][]string{{"a", "c"}, {"b", "c"}},
		},
		{
			name:     "deeply nested",
			expr:     "((a AND b) OR (c AND d)) AND e",
			expected: [][]string{{"a", "b", "e"}, {"c", "d", "e"}},
		},

		// ── Real-world examples from .plumber.yaml ─────────────────────
		{
			name:     "real: sast + secret-detection OR full-security",
			expr:     "(components/sast/sast AND components/secret-detection/secret-detection) OR your-org/full-security-pipeline/full-security",
			expected: [][]string{{"components/sast/sast", "components/secret-detection/secret-detection"}, {"your-org/full-security-pipeline/full-security"}},
		},
		{
			name:     "real: three security components",
			expr:     "components/secret-detection/secret-detection AND components/sast/sast AND getplumber/plumber/plumber",
			expected: [][]string{{"components/secret-detection/secret-detection", "components/sast/sast", "getplumber/plumber/plumber"}},
		},
		{
			name:     "real: templates go + trivy OR full pipeline",
			expr:     "(templates/go/go AND templates/trivy/trivy) OR templates/full-go-pipeline",
			expected: [][]string{{"templates/go/go", "templates/trivy/trivy"}, {"templates/full-go-pipeline"}},
		},

		// ── Case-insensitive operators ──────────────────────────────────
		{
			name:     "lowercase operators",
			expr:     "a and b or c",
			expected: [][]string{{"a", "b"}, {"c"}},
		},
		{
			name:     "mixed case operators",
			expr:     "a And b Or c",
			expected: [][]string{{"a", "b"}, {"c"}},
		},

		// ── Parens touching identifiers ─────────────────────────────────
		{
			name:     "parens touching idents",
			expr:     "(a)AND(b)",
			expected: [][]string{{"a", "b"}},
		},
		{
			name:     "parens touching idents with OR",
			expr:     "(a OR b)AND(c)",
			expected: [][]string{{"a", "c"}, {"b", "c"}},
		},

		// ── Error cases ─────────────────────────────────────────────────
		{
			name:    "error: leading AND",
			expr:    "AND a",
			wantErr: true,
		},
		{
			name:    "error: trailing AND",
			expr:    "a AND",
			wantErr: true,
		},
		{
			name:    "error: leading OR",
			expr:    "OR a",
			wantErr: true,
		},
		{
			name:    "error: trailing OR",
			expr:    "a OR",
			wantErr: true,
		},
		{
			name:    "error: double AND",
			expr:    "a AND AND b",
			wantErr: true,
		},
		{
			name:    "error: unmatched open paren",
			expr:    "(a AND b",
			wantErr: true,
		},
		{
			name:    "error: unmatched close paren",
			expr:    "a AND b)",
			wantErr: true,
		},
		{
			name:    "error: empty parens",
			expr:    "()",
			wantErr: true,
		},
		{
			name:    "error: just AND",
			expr:    "AND",
			wantErr: true,
		},
		{
			name:    "error: just OR",
			expr:    "OR",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRequiredExpression(tt.expr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRequiredExpression(%q) expected error, got %v", tt.expr, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseRequiredExpression(%q) unexpected error: %v", tt.expr, err)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseRequiredExpression(%q)\n  got:  %v\n  want: %v", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestGroupsToExpression(t *testing.T) {
	tests := []struct {
		name     string
		groups   [][]string
		expected string
	}{
		{
			name:     "empty",
			groups:   [][]string{},
			expected: "",
		},
		{
			name:     "nil",
			groups:   nil,
			expected: "",
		},
		{
			name:     "single item single group",
			groups:   [][]string{{"a"}},
			expected: "a",
		},
		{
			name:     "multiple items single group",
			groups:   [][]string{{"a", "b", "c"}},
			expected: "a AND b AND c",
		},
		{
			name:     "single items multiple groups",
			groups:   [][]string{{"a"}, {"b"}, {"c"}},
			expected: "a OR b OR c",
		},
		{
			name:     "mixed groups",
			groups:   [][]string{{"a", "b"}, {"c"}},
			expected: "(a AND b) OR c",
		},
		{
			name:     "two multi-item groups",
			groups:   [][]string{{"a", "b"}, {"c", "d"}},
			expected: "(a AND b) OR (c AND d)",
		},
		{
			name:     "skip empty groups",
			groups:   [][]string{{}, {"a"}, {}},
			expected: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupsToExpression(tt.groups)
			if got != tt.expected {
				t.Errorf("GroupsToExpression(%v)\n  got:  %q\n  want: %q", tt.groups, got, tt.expected)
			}
		})
	}
}

// TestRoundTrip verifies that parsing an expression and converting back gives
// an equivalent (though not necessarily identical) expression.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"single", "a"},
		{"two AND", "a AND b"},
		{"two OR", "a OR b"},
		{"DNF", "(a AND b) OR c"},
		{"complex DNF", "(a AND b) OR (c AND d)"},
		{"real world", "(components/sast/sast AND components/secret-detection/secret-detection) OR your-org/full-security/full-security"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse expression to groups
			groups, err := ParseRequiredExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseRequiredExpression(%q) failed: %v", tt.expr, err)
			}

			// Convert groups back to expression
			regenerated := GroupsToExpression(groups)

			// Parse the regenerated expression
			groups2, err := ParseRequiredExpression(regenerated)
			if err != nil {
				t.Fatalf("ParseRequiredExpression(%q) [regenerated] failed: %v", regenerated, err)
			}

			// The DNF groups should be identical
			if !reflect.DeepEqual(groups, groups2) {
				t.Errorf("Round-trip failed for %q:\n  groups1: %v\n  regen:   %q\n  groups2: %v",
					tt.expr, groups, regenerated, groups2)
			}
		})
	}
}

func TestValidateExpression(t *testing.T) {
	valid := []string{
		"",
		"a",
		"a AND b",
		"a OR b",
		"(a AND b) OR c",
		"a AND (b OR c)",
	}
	for _, expr := range valid {
		if err := ValidateExpression(expr); err != nil {
			t.Errorf("ValidateExpression(%q) expected valid, got error: %v", expr, err)
		}
	}

	invalid := []string{
		"AND",
		"OR",
		"a AND",
		"a OR",
		"(a",
		"a)",
		"()",
		"a AND AND b",
	}
	for _, expr := range invalid {
		if err := ValidateExpression(expr); err == nil {
			t.Errorf("ValidateExpression(%q) expected error, got nil", expr)
		}
	}
}
