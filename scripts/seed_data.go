//go:build ignore

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

func main() {
	// Load environment variables
	walletPath := os.Getenv("ORACLE_WALLET_PATH")
	tnsAlias := os.Getenv("ORACLE_TNS_ALIAS")
	user := os.Getenv("ORACLE_USER")
	password := os.Getenv("ORACLE_PASSWORD")

	if walletPath == "" || tnsAlias == "" || user == "" || password == "" {
		log.Fatal("ORACLE_WALLET_PATH, ORACLE_TNS_ALIAS, ORACLE_USER, and ORACLE_PASSWORD are required")
	}

	// Build connection string for Oracle Cloud with wallet
	// Only enable trace logging if ENABLE_ORACLE_TRACE is set
	traceParam := ""
	if os.Getenv("ENABLE_ORACLE_TRACE") == "true" {
		traceParam = "&TRACE FILE=trace.log"
	}
	dsn := fmt.Sprintf("oracle://%s:%s@%s?SSL=enable&SSL Verify=false&WALLET=%s%s",
		user, password, tnsAlias, walletPath, traceParam)

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Connected to Oracle database")

	// Default tenant_id - can be overridden via command line
	tenantID := "tenant_admin"
	if len(os.Args) > 1 {
		tenantID = os.Args[1]
	}
	fmt.Printf("Using tenant_id: %s\n", tenantID)

	// Check current counts
	printCounts(ctx, db, tenantID)

	// Insert seed data
	if err := insertCustomers(ctx, db, tenantID); err != nil {
		log.Fatalf("Failed to insert customers: %v", err)
	}

	if err := insertServices(ctx, db, tenantID); err != nil {
		log.Fatalf("Failed to insert services: %v", err)
	}

	if err := insertContracts(ctx, db, tenantID); err != nil {
		log.Fatalf("Failed to insert contracts: %v", err)
	}

	if err := insertPrintJobs(ctx, db, tenantID); err != nil {
		log.Fatalf("Failed to insert print jobs: %v", err)
	}

	fmt.Println("\n✓ Seed data inserted successfully!")
	printCounts(ctx, db, tenantID)
}

func printCounts(ctx context.Context, db *sql.DB, tenantID string) {
	tables := []string{"customers", "services", "contracts", "contract_print_jobs"}
	fmt.Println("\nRecord counts:")
	for _, table := range tables {
		var count int
		err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id = :1", table), tenantID).Scan(&count)
		if err != nil {
			fmt.Printf("  %s: error (%v)\n", table, err)
			continue
		}
		fmt.Printf("  %s: %d\n", table, count)
	}
}

func insertCustomers(ctx context.Context, db *sql.DB, tenantID string) error {
	customers := []struct {
		code, ctype, name, tradeName, taxID, email, phone string
	}{
		{"CUST001", "COMPANY", "Acme Corporation", "Acme Corp", "12345678901234", "contact@acme.com", "(11) 99999-0001"},
		{"CUST002", "COMPANY", "TechStart Ltda", "TechStart", "98765432109876", "info@techstart.com.br", "(11) 98888-0002"},
		{"CUST003", "INDIVIDUAL", "João Silva", "", "12345678901", "joao.silva@email.com", "(21) 97777-0003"},
		{"CUST004", "COMPANY", "Global Services SA", "GlobalServ", "11122233344455", "contato@globalserv.com.br", "(11) 96666-0004"},
		{"CUST005", "INDIVIDUAL", "Maria Santos", "", "98765432100", "maria.santos@gmail.com", "(31) 95555-0005"},
	}

	for _, c := range customers {
		var tradeName interface{}
		if c.tradeName != "" {
			tradeName = c.tradeName
		}

		_, err := db.ExecContext(ctx, `
			INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
			VALUES (:1, :2, :3, :4, :5, :6, :7, :8, 1, 'seed_script')`,
			tenantID, c.code, c.ctype, c.name, tradeName, c.taxID, c.email, c.phone)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "ORA-00001") {
				fmt.Printf("  Customer %s already exists, skipping\n", c.code)
				continue
			}
			return fmt.Errorf("insert customer %s: %w", c.code, err)
		}
		fmt.Printf("  Inserted customer: %s\n", c.name)
	}
	return nil
}

func insertServices(ctx context.Context, db *sql.DB, tenantID string) error {
	services := []struct {
		code, name, desc, category, priceUnit string
		price                                 float64
	}{
		{"SVC001", "Software Development", "Custom software development services", "Development", "HOUR", 250.00},
		{"SVC002", "Technical Support", "24/7 technical support and maintenance", "Support", "HOUR", 150.00},
		{"SVC003", "Cloud Hosting", "Managed cloud hosting services", "Infrastructure", "MONTH", 500.00},
		{"SVC004", "Consulting", "IT consulting and advisory services", "Consulting", "HOUR", 350.00},
		{"SVC005", "Training", "Technical training and workshops", "Training", "DAY", 2000.00},
		{"SVC006", "System Integration", "API and system integration services", "Development", "PROJECT", 15000.00},
	}

	for _, s := range services {
		_, err := db.ExecContext(ctx, `
			INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
			VALUES (:1, :2, :3, :4, :5, :6, 'BRL', :7, 1, 'seed_script')`,
			tenantID, s.code, s.name, s.desc, s.category, s.price, s.priceUnit)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "ORA-00001") {
				fmt.Printf("  Service %s already exists, skipping\n", s.code)
				continue
			}
			return fmt.Errorf("insert service %s: %w", s.code, err)
		}
		fmt.Printf("  Inserted service: %s\n", s.name)
	}
	return nil
}

func insertContracts(ctx context.Context, db *sql.DB, tenantID string) error {
	contracts := []struct {
		number, ctype, customerCode, status, billingCycle string
		startDate, endDate                                string
		value                                             float64
	}{
		{"CTR-2024-001", "SERVICE", "CUST001", "ACTIVE", "MONTHLY", "2024-01-15", "2025-01-14", 50000.00},
		{"CTR-2024-002", "RECURRING", "CUST002", "ACTIVE", "MONTHLY", "2024-03-01", "2025-02-28", 36000.00},
		{"CTR-2024-003", "PROJECT", "CUST003", "ACTIVE", "PROJECT", "2024-06-01", "2024-12-31", 25000.00},
		{"CTR-2024-004", "SERVICE", "CUST004", "DRAFT", "QUARTERLY", "2024-09-01", "", 75000.00},
		{"CTR-2024-005", "SERVICE", "CUST005", "PENDING", "MONTHLY", "2024-08-01", "2025-07-31", 12000.00},
	}

	for _, c := range contracts {
		// Get customer ID
		var customerID int64
		err := db.QueryRowContext(ctx,
			"SELECT id FROM customers WHERE tenant_id = :1 AND customer_code = :2",
			tenantID, c.customerCode).Scan(&customerID)
		if err != nil {
			return fmt.Errorf("find customer %s: %w", c.customerCode, err)
		}

		var endDate interface{}
		if c.endDate != "" {
			endDate = c.endDate
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, end_date, total_value, billing_cycle, status, created_by)
			VALUES (:1, :2, :3, :4, TO_DATE(:5, 'YYYY-MM-DD'), TO_DATE(:6, 'YYYY-MM-DD'), :7, :8, :9, 'seed_script')`,
			tenantID, c.number, c.ctype, customerID, c.startDate, endDate, c.value, c.billingCycle, c.status)
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "ORA-00001") {
				fmt.Printf("  Contract %s already exists, skipping\n", c.number)
				continue
			}
			return fmt.Errorf("insert contract %s: %w", c.number, err)
		}
		fmt.Printf("  Inserted contract: %s\n", c.number)
	}
	return nil
}

func insertPrintJobs(ctx context.Context, db *sql.DB, tenantID string) error {
	jobs := []struct {
		contractNumber, status, format string
	}{
		{"CTR-2024-001", "COMPLETED", "PDF"},
		{"CTR-2024-002", "PROCESSING", "PDF"},
		{"CTR-2024-003", "QUEUED", "DOCX"},
	}

	for _, j := range jobs {
		// Get contract ID
		var contractID int64
		err := db.QueryRowContext(ctx,
			"SELECT id FROM contracts WHERE tenant_id = :1 AND contract_number = :2",
			tenantID, j.contractNumber).Scan(&contractID)
		if err != nil {
			fmt.Printf("  Contract %s not found, skipping print job\n", j.contractNumber)
			continue
		}

		// Check if already exists
		var count int
		err = db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM contract_print_jobs WHERE tenant_id = :1 AND contract_id = :2",
			tenantID, contractID).Scan(&count)
		if err == nil && count > 0 {
			fmt.Printf("  Print job for %s already exists, skipping\n", j.contractNumber)
			continue
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO contract_print_jobs (tenant_id, contract_id, status, format, queued_at, requested_by)
			VALUES (:1, :2, :3, :4, CURRENT_TIMESTAMP, 'admin')`,
			tenantID, contractID, j.status, j.format)
		if err != nil {
			return fmt.Errorf("insert print job for %s: %w", j.contractNumber, err)
		}
		fmt.Printf("  Inserted print job for: %s\n", j.contractNumber)
	}
	return nil
}
