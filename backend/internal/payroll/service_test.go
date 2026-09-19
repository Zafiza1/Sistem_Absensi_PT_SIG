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
