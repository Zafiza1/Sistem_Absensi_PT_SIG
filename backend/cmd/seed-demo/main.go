// Command seed-demo fills the database with a year of example attendance
// data — 24 employees across four "(Demo)" departments, with check-ins,
// lateness, izin, sakit, cuti and alpha from 1 January of the current year
// up to today — so the daily report, the monthly report, payroll and the
// year-end recap (Laporan → Tahunan) can be tried out before real data
// exists. It also creates and processes a payroll period for every month
// before the current one (all but the latest locked and marked paid), so
// the Payroll page has something to show. The data is generated from a fixed seed, so every run on the same
// day produces the same numbers.
//
// Every row it creates hangs off an employee whose NIK starts with "DEMO-"
// or, for payroll periods, carries the note demoPeriodNote; running it again
// first removes all of that, so it never touches real data. A month that
// already has a (non-demo) payroll period is left alone. It refuses to run with APP_ENV=production.
//
// Usage:
//
//	go run ./cmd/seed-demo           # (re)create the demo data
//	go run ./cmd/seed-demo -clean    # only remove it
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/suryaintigas/absensi-backend/internal/config"
	"github.com/suryaintigas/absensi-backend/internal/payroll"
	"github.com/suryaintigas/absensi-backend/internal/report"
)

const (
	numberPrefix = "DEMO-"
	deptSuffix   = " (Demo)"
	shiftName    = "Shift Demo 08:00-17:00"
	cutiQuota    = 12

	demoPeriodNote = "Data demo (seed-demo)"
)

var departments = []string{"Produksi", "Gudang", "Administrasi", "Keuangan"}

var names = []string{
	"Budi Santoso", "Siti Aminah", "Agus Wijaya", "Dewi Lestari", "Rudi Hartono", "Rina Marlina",
	"Joko Susilo", "Sri Wahyuni", "Andi Pratama", "Nur Hidayah", "Eko Prasetyo", "Lina Kurniawati",
	"Hendra Gunawan", "Yuni Astuti", "Fajar Nugroho", "Maya Sari", "Bambang Riyadi", "Wulan Purnama",
	"Dimas Saputra", "Ratna Dewi", "Imam Fauzi", "Fitri Handayani", "Teguh Santoso", "Indah Permata",
}

var (
	sakitReasons = []string{"Demam, kabar via WA", "Sakit perut", "Flu dan batuk", "Kontrol ke dokter", "Sakit gigi"}
	izinReasons  = []string{"Keperluan keluarga", "Mengurus dokumen", "Acara pernikahan saudara", "Anak sakit", "Ban motor bocor"}
	cutiReasons  = []string{"Cuti tahunan", "Pulang kampung", "Liburan keluarga"}
)

var jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}()

type leaveRow struct {
	employeeID uuid.UUID
	leaveType  string
	start, end time.Time
	reason     string
}

type attendanceRow struct {
	employeeID  uuid.UUID
	date        time.Time
	lateMinutes int
	checkedOut  bool
}

func main() {
	clean := flag.Bool("clean", false, "only remove the demo data, don't recreate it")
	flag.Parse()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	if cfg.IsProduction() {
		log.Fatal("seed-demo refuses to run with APP_ENV=production")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if err := removeDemo(ctx, tx); err != nil {
		log.Fatalf("failed to remove previous demo data: %v", err)
	}
	if !*clean {
		if err := seed(ctx, tx); err != nil {
			log.Fatalf("failed to seed demo data: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit: %v", err)
	}
	if *clean {
		log.Print("demo data removed")
		return
	}

	// Payroll runs through the real service, outside the transaction, so
	// the periods are computed exactly as the dashboard's "Proses" would.
	if err := seedPayroll(ctx, pool); err != nil {
		log.Fatalf("failed to seed demo payroll: %v", err)
	}
}

func seedPayroll(ctx context.Context, pool *pgxpool.Pool) error {
	var processor uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM users ORDER BY created_at LIMIT 1`).Scan(&processor)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Print("no dashboard user yet (run ./cmd/seed first) — skipping demo payroll periods")
		return nil
	}
	if err != nil {
		return err
	}

	svc := payroll.NewService(payroll.NewPostgresRepository(pool), report.NewService(report.NewPostgresRepository(pool)))
	now := time.Now().In(jakarta)
	created := 0
	for month := 1; month < int(now.Month()); month++ {
		period, err := svc.CreatePayrollPeriod(ctx, now.Year(), month, demoPeriodNote)
		if errors.Is(err, payroll.ErrPeriodExists) {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := svc.ProcessPayrollPeriod(ctx, period.ID, processor); err != nil {
			return err
		}
		// The latest month stays COMPLETED (still editable, unpaid); earlier
		// ones are paid on the 28th and locked, as a finished month would be.
		if month < int(now.Month())-1 {
			paidOn := time.Date(now.Year(), time.Month(month), 28, 0, 0, 0, 0, jakarta)
			if err := svc.MarkAsPaid(ctx, period.ID, paidOn, "TRANSFER", fmt.Sprintf("DEMO-%02d", month)); err != nil {
				return err
			}
			if err := svc.LockPayrollPeriod(ctx, period.ID); err != nil {
				return err
			}
		}
		created++
	}
	log.Printf("seeded %d demo payroll periods", created)
	return nil
}

func removeDemo(ctx context.Context, tx pgx.Tx) error {
	demoEmployees := `(SELECT id FROM employees WHERE employee_number LIKE '` + numberPrefix + `%')`
	stmts := []string{
		// Items cascade with their period.
		`DELETE FROM payroll_periods WHERE notes = '` + demoPeriodNote + `'`,
		`DELETE FROM payroll_items WHERE employee_id IN ` + demoEmployees,
		`DELETE FROM attendances WHERE employee_id IN ` + demoEmployees,
		`DELETE FROM attendance_logs WHERE employee_id IN ` + demoEmployees,
		// leaves and work_schedules cascade.
		`DELETE FROM employees WHERE employee_number LIKE '` + numberPrefix + `%'`,
		`DELETE FROM shifts WHERE name = '` + shiftName + `'`,
		`DELETE FROM departments WHERE name LIKE '%` + deptSuffix + `'`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(ctx, q); err != nil {
			return fmt.Errorf("%s: %w", q, err)
		}
	}
	return nil
}

func seed(ctx context.Context, tx pgx.Tx) error {
	rng := rand.New(rand.NewSource(2026))
	now := time.Now().In(jakarta)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	deptIDs := make([]uuid.UUID, len(departments))
	for i, d := range departments {
		if err := tx.QueryRow(ctx, `INSERT INTO departments (name) VALUES ($1) RETURNING id`, d+deptSuffix).Scan(&deptIDs[i]); err != nil {
			return err
		}
	}

	var shiftID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO shifts (name, start_time, end_time, late_tolerance_minutes, working_duration_minutes)
		VALUES ($1, '08:00', '17:00', 0, 480) RETURNING id`, shiftName).Scan(&shiftID); err != nil {
		return err
	}

	var (
		leaves      []leaveRow
		attendances []attendanceRow
	)

	for i, name := range names {
		var empID uuid.UUID
		salary := int64(4_500_000 + rng.Intn(8)*500_000)
		// employees.shift_id stays NULL: the Mon–Sat work_schedules rows
		// below make Sunday a day off without depending on Jam Kerja.
		if err := tx.QueryRow(ctx, `
			INSERT INTO employees (employee_number, name, department_id, base_salary)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			fmt.Sprintf("%s%03d", numberPrefix, i+1), name, deptIDs[i%len(deptIDs)], salary).Scan(&empID); err != nil {
			return err
		}
		for dow := 1; dow <= 6; dow++ {
			if _, err := tx.Exec(ctx, `INSERT INTO work_schedules (employee_id, shift_id, day_of_week) VALUES ($1, $2, $3)`,
				empID, shiftID, dow); err != nil {
				return err
			}
		}

		// Each employee gets their own habits: some are rarely sick, some
		// often late, so the recap has a realistic spread.
		sickness := 0.4 + rng.Float64()*1.6
		tardiness := 0.3 + rng.Float64()*2.0
		pSakit := 0.012 * sickness
		pIzin := pSakit + 0.010
		pCuti := pIzin + 0.008
		pAlpha := pCuti + 0.006*tardiness
		cutiUsed := 0

		for d := yearStart; d.Before(today); d = d.AddDate(0, 0, 1) {
			if d.Weekday() == time.Sunday {
				continue
			}
			switch r := rng.Float64(); {
			case r < pSakit:
				end := clampEnd(d, d.AddDate(0, 0, rng.Intn(3)), today)
				leaves = append(leaves, leaveRow{empID, "SAKIT", d, end, pick(rng, sakitReasons)})
				d = end
			case r < pIzin:
				leaves = append(leaves, leaveRow{empID, "IZIN", d, d, pick(rng, izinReasons)})
			case r < pCuti && cutiUsed <= cutiQuota-6:
				end := clampEnd(d, d.AddDate(0, 0, rng.Intn(3)), today)
				cutiUsed += int(end.Sub(d).Hours()/24) + 1
				leaves = append(leaves, leaveRow{empID, "CUTI", d, end, pick(rng, cutiReasons)})
				d = end
			case r < pAlpha:
				// Alpha: no check-in and nobody told HR.
			default:
				attendances = append(attendances, attendanceRow{empID, d, lateMinutes(rng, tardiness), true})
			}
		}

		// A few employees already have cuti booked for the end of the year.
		start := time.Date(now.Year(), 12, 22, 0, 0, 0, 0, time.UTC)
		if i%5 == 0 && cutiUsed+3 <= cutiQuota && start.After(today) {
			leaves = append(leaves, leaveRow{empID, "CUTI", start, start.AddDate(0, 0, 2), "Cuti akhir tahun"})
		}

		// Today: most have checked in (not out yet); one reported sick; the
		// rest haven't shown up — those are who HR labels in Laporan Harian.
		if today.Weekday() != time.Sunday {
			switch {
			case i == 3:
				leaves = append(leaves, leaveRow{empID, "SAKIT", today, today, "Demam, kabar via WA"})
			case i%6 != 5:
				attendances = append(attendances, attendanceRow{empID, today, lateMinutes(rng, tardiness), false})
			}
		}
	}

	for _, l := range leaves {
		days := int(l.end.Sub(l.start).Hours()/24) + 1
		if _, err := tx.Exec(ctx, `
			INSERT INTO leaves (employee_id, leave_type, start_date, end_date, days_count, reason)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			l.employeeID, l.leaveType, l.start, l.end, days, l.reason); err != nil {
			return err
		}
	}

	rows := make([][]any, 0, len(attendances))
	for _, a := range attendances {
		y, m, d := a.date.Date()
		checkIn := time.Date(y, m, d, 7, 40+rng.Intn(20), rng.Intn(60), 0, jakarta)
		status := "ON_TIME"
		if a.lateMinutes > 0 {
			checkIn = time.Date(y, m, d, 8, a.lateMinutes, rng.Intn(60), 0, jakarta)
			status = "LATE"
		}
		var (
			checkOut *time.Time
			duration *int
		)
		if a.checkedOut {
			out := time.Date(y, m, d, 17, rng.Intn(30), rng.Intn(60), 0, jakarta)
			mins := int(out.Sub(checkIn).Minutes())
			checkOut, duration, status = &out, &mins, "CHECKED_OUT"
		}
		rows = append(rows, []any{a.employeeID, shiftID, a.date, checkIn, checkOut, status, a.lateMinutes, duration})
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"attendances"},
		[]string{"employee_id", "shift_id", "attendance_date", "check_in_at", "check_out_at", "status", "late_minutes", "working_duration_minutes"},
		pgx.CopyFromRows(rows)); err != nil {
		return err
	}

	counts := map[string]int{}
	for _, l := range leaves {
		counts[l.leaveType]++
	}
	log.Printf("seeded %d demo employees, %d attendance rows, leaves: %d sakit, %d izin, %d cuti",
		len(names), len(attendances), counts["SAKIT"], counts["IZIN"], counts["CUTI"])
	return nil
}

// clampEnd keeps a multi-day leave that starts on start from reaching
// today or crossing into the next year.
func clampEnd(start, end, today time.Time) time.Time {
	if !end.Before(today) {
		end = today.AddDate(0, 0, -1)
	}
	if end.Year() != start.Year() || end.Before(start) {
		return start
	}
	return end
}

// lateMinutes returns 0 most days; when late, mostly a few minutes, rarely
// past half an hour — spread across the three payroll deduction tiers.
func lateMinutes(rng *rand.Rand, tardiness float64) int {
	if rng.Float64() >= 0.05*tardiness {
		return 0
	}
	switch r := rng.Float64(); {
	case r < 0.65:
		return 1 + rng.Intn(10)
	case r < 0.9:
		return 11 + rng.Intn(20)
	default:
		return 31 + rng.Intn(29)
	}
}

func pick(rng *rand.Rand, xs []string) string { return xs[rng.Intn(len(xs))] }
