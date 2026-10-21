package payroll

import "testing"

// --- ComputeLateDeduction: pure-function coverage of the deduction tiers ---

func TestComputeLateDeduction(t *testing.T) {
	const baseSalary = int64(5_200_000) // -> 100,000/day, 50,000 half-day

	tests := []struct {
		name        string
		lateMinutes int
		want        int64
	}{
		{"on time", 0, 0},
		{"negative (defensive)", -5, 0},
		{"tier1 lower bound", 1, LateTier1Amount},
		{"tier1 upper bound", 10, LateTier1Amount},
		{"tier2 lower bound", 11, LateTier2Amount},
		{"tier2 upper bound", 30, LateTier2Amount},
		{"tier3 lower bound", 31, baseSalary / WorkingDaysPerMonth / 2},
		{"tier3 very late", 240, baseSalary / WorkingDaysPerMonth / 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeLateDeduction(tt.lateMinutes, baseSalary)
			if got != tt.want {
				t.Errorf("ComputeLateDeduction(%d, %d) = %d, want %d", tt.lateMinutes, baseSalary, got, tt.want)
			}
		})
	}
}

func TestComputeAbsentDeduction(t *testing.T) {
	const baseSalary = int64(5_200_000) // / 26 -> 200,000/day
	if got := ComputeAbsentDeduction(0, baseSalary); got != 0 {
		t.Errorf("0 days = %d, want 0", got)
	}
	if got := ComputeAbsentDeduction(3, baseSalary); got != 600_000 {
		t.Errorf("3 days = %d, want 600000", got)
	}
}

func TestLateDeductionFor_PerDayRules(t *testing.T) {
	lo, hi := 1, 10
	rules := []DeductionRule{{RuleType: "LATE", LateMinMinutes: &lo, LateMaxMinutes: &hi, DeductionAmount: 25000, DeductionType: "FIXED"}}

	if got := lateDeductionFor(rules, 5, 5_200_000); got != 25000 {
		t.Errorf("5 min with rule = %d, want 25000", got)
	}
	// No rule covers 40 minutes: falls back to the built-in half-day tier.
	if got := lateDeductionFor(rules, 40, 5_200_000); got != 100_000 {
		t.Errorf("40 min fallback = %d, want 100000", got)
	}
	if got := lateDeductionFor(nil, 5, 5_200_000); got != LateTier1Amount {
		t.Errorf("5 min no rules = %d, want %d", got, LateTier1Amount)
	}
}
