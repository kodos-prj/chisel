package alpm

import (
	"testing"
)

// TestVersionComparison_Equal tests equal version strings.
func TestVersionComparison_Equal(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"1.0", "1.0"},
		{"2.5", "2.5"},
		{"1:2.0", "1:2.0"},
		{"1.0-1", "1.0-1"},
		{"1:2.0-3", "1:2.0-3"},
	}

	for _, tt := range tests {
		if result := VerCmp(tt.a, tt.b); result != 0 {
			t.Errorf("VerCmp(%q, %q) = %d, want 0", tt.a, tt.b, result)
		}
	}
}

// TestVersionComparison_LessThan tests less-than comparisons.
func TestVersionComparison_LessThan(t *testing.T) {
	tests := []struct {
		a, b string
		desc string
	}{
		{"1.0", "1.1", "patch version"},
		{"1.9", "2.0", "major version"},
		{"1.0-1", "1.0-2", "revision"},
		{"1.0", "1.0.1", "shorter < longer with extra segment"},
		{"1.0alpha", "1.0beta", "alpha < beta"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != -1 {
			t.Errorf("VerCmp(%q, %q) = %d (want -1) [%s]", tt.a, tt.b, result, tt.desc)
		}
	}
}

// TestVersionComparison_GreaterThan tests greater-than comparisons.
func TestVersionComparison_GreaterThan(t *testing.T) {
	tests := []struct {
		a, b string
		desc string
	}{
		{"1.1", "1.0", "patch version"},
		{"2.0", "1.9", "major version"},
		{"1.0-2", "1.0-1", "revision"},
		{"1.0.1", "1.0", "longer > shorter with extra segment"},
		{"1.0beta", "1.0alpha", "beta > alpha"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != 1 {
			t.Errorf("VerCmp(%q, %q) = %d (want 1) [%s]", tt.a, tt.b, result, tt.desc)
		}
	}
}

// TestVersionComparison_Epochs tests epoch handling (highest priority).
func TestVersionComparison_Epochs(t *testing.T) {
	tests := []struct {
		a, b   string
		expect int
		desc   string
	}{
		{"1:0.0", "2.0", 1, "epoch 1 > no epoch"},
		{"2:0.0", "1:9.9", 1, "epoch 2 > epoch 1"},
		{"1:1.0", "1:1.0", 0, "equal epochs and versions"},
		{"0:1.0", "1.0", 0, "explicit 0 epoch = implicit 0 epoch"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != tt.expect {
			t.Errorf("VerCmp(%q, %q) = %d (want %d) [%s]", tt.a, tt.b, result, tt.expect, tt.desc)
		}
	}
}

// TestVersionComparison_NumericPrefix tests numeric prefix comparisons.
func TestVersionComparison_NumericPrefix(t *testing.T) {
	tests := []struct {
		a, b string
		desc string
	}{
		{"1.0", "2.0", "different numeric prefixes"},
		{"10.0", "2.0", "numeric string comparison (10 > 2 numerically)"},
		{"1.0.0", "1.0", "different segment counts"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != 1 && result != -1 {
			t.Errorf("VerCmp(%q, %q) = %d (want -1 or 1) [%s]", tt.a, tt.b, result, tt.desc)
		}
	}
}

// TestVersionComparison_ComplexVersions tests realistic version strings from Arch.
func TestVersionComparison_ComplexVersions(t *testing.T) {
	tests := []struct {
		a, b   string
		expect int
		desc   string
	}{
		{"2.10.0-1", "2.9.0-1", 1, "2.10 > 2.9"},
		{"5.3.11-1", "5.3.11-2", -1, "revision 1 < 2"},
		{"1:5.0-1", "5.20-1", 1, "epoch 1 > no epoch"},
		{"4.3.0.p20201215-1", "4.3.0-1", 1, "pre-release tag"},
	}

	for _, tt := range tests {
		result := VerCmp(tt.a, tt.b)
		if result != tt.expect {
			t.Errorf("VerCmp(%q, %q) = %d (want %d) [%s]", tt.a, tt.b, result, tt.expect, tt.desc)
		}
	}
}

// TestVersionComparison_Symmetry tests that VerCmp is symmetric: if a < b then b > a.
func TestVersionComparison_Symmetry(t *testing.T) {
	pairs := [][2]string{
		{"1.0", "1.1"},
		{"1.0rc1", "1.0"},
		{"1:0.0", "2.0"},
		{"1.0-1", "1.0-2"},
	}

	for _, pair := range pairs {
		a, b := pair[0], pair[1]
		resultAB := VerCmp(a, b)
		resultBA := VerCmp(b, a)

		// resultAB should be -1 or 1, resultBA should be the opposite
		if resultAB == -1 && resultBA != 1 {
			t.Errorf("Asymmetry: VerCmp(%q, %q) = %d, VerCmp(%q, %q) = %d",
				a, b, resultAB, b, a, resultBA)
		}
		if resultAB == 1 && resultBA != -1 {
			t.Errorf("Asymmetry: VerCmp(%q, %q) = %d, VerCmp(%q, %q) = %d",
				a, b, resultAB, b, a, resultBA)
		}
	}
}

// TestParseDependency tests dependency parsing.
func TestParseDependency(t *testing.T) {
	tests := []struct {
		dep           string
		expectedName  string
		expectedType  ConstraintType
		expectedValue string
	}{
		{"pkg", "pkg", ConstraintNone, ""},
		{"pkg>=1.0", "pkg", ConstraintGreaterEqual, "1.0"},
		{"pkg<=2.0", "pkg", ConstraintLessEqual, "2.0"},
		{"pkg>1.5", "pkg", ConstraintGreater, "1.5"},
		{"pkg<3.0", "pkg", ConstraintLess, "3.0"},
		{"pkg=1.0", "pkg", ConstraintEqual, "1.0"},
	}

	for _, tt := range tests {
		name, constraint, err := ParseDependency(tt.dep)
		if err != nil {
			t.Errorf("ParseDependency(%q) returned error: %v", tt.dep, err)
			continue
		}
		if name != tt.expectedName {
			t.Errorf("ParseDependency(%q): name = %q, want %q", tt.dep, name, tt.expectedName)
		}
		if constraint.Type != tt.expectedType {
			t.Errorf("ParseDependency(%q): type = %v, want %v", tt.dep, constraint.Type, tt.expectedType)
		}
		if constraint.Value != tt.expectedValue {
			t.Errorf("ParseDependency(%q): value = %q, want %q", tt.dep, constraint.Value, tt.expectedValue)
		}
	}
}

// TestCheckVersionConstraint tests version constraint checking.
func TestCheckVersionConstraint(t *testing.T) {
	tests := []struct {
		version    string
		constraint Constraint
		expected   bool
		desc       string
	}{
		{"1.0", Constraint{ConstraintNone, ""}, true, "no constraint always true"},
		{"1.0", Constraint{ConstraintEqual, "1.0"}, true, "equal constraint"},
		{"1.1", Constraint{ConstraintEqual, "1.0"}, false, "unequal with equal constraint"},
		{"1.1", Constraint{ConstraintGreaterEqual, "1.0"}, true, ">= constraint match"},
		{"1.0", Constraint{ConstraintGreaterEqual, "1.0"}, true, ">= constraint equal"},
		{"0.9", Constraint{ConstraintGreaterEqual, "1.0"}, false, ">= constraint no match"},
		{"1.1", Constraint{ConstraintGreater, "1.0"}, true, "> constraint match"},
		{"1.0", Constraint{ConstraintGreater, "1.0"}, false, "> constraint equal"},
		{"0.9", Constraint{ConstraintLessEqual, "1.0"}, true, "<= constraint match"},
		{"1.0", Constraint{ConstraintLessEqual, "1.0"}, true, "<= constraint equal"},
		{"1.1", Constraint{ConstraintLessEqual, "1.0"}, false, "<= constraint no match"},
		{"0.9", Constraint{ConstraintLess, "1.0"}, true, "< constraint match"},
		{"1.0", Constraint{ConstraintLess, "1.0"}, false, "< constraint equal"},
	}

	for _, tt := range tests {
		result := CheckVersionConstraint(tt.version, tt.constraint)
		if result != tt.expected {
			t.Errorf("CheckVersionConstraint(%q, %v) = %v, want %v [%s]",
				tt.version, tt.constraint, result, tt.expected, tt.desc)
		}
	}
}
