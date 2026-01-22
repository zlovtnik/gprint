package config

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/godror/godror"
)

// OracleConfig holds Oracle database configuration
type OracleConfig struct {
	Host         string
	Port         string
	Service      string
	User         string
	Password     string
	MaxOpenConns int
	MaxIdleConns int
	// Wallet configuration for Oracle Cloud (ADB)
	WalletPath string
	TNSAlias   string
}

// escapeDSNValue escapes special characters in DSN values to prevent injection
func escapeDSNValue(s string) string {
	// Escape backslashes first, then double quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// DSN returns the Oracle connection string
func (c OracleConfig) DSN() string {
	user := escapeDSNValue(c.User)
	password := escapeDSNValue(c.Password)

	// If wallet is configured, use wallet-based connection
	if c.WalletPath != "" && c.TNSAlias != "" {
		tnsAlias := escapeDSNValue(c.TNSAlias)
		walletPath := escapeDSNValue(c.WalletPath)
		return fmt.Sprintf(`user="%s" password="%s" connectString="%s" configDir="%s" walletLocation="%s"`,
			user, password, tnsAlias, walletPath, walletPath)
	}
	// Standard connection string
	return fmt.Sprintf(`user="%s" password="%s" connectString="%s:%s/%s"`,
		user, password, c.Host, c.Port, c.Service)
}

// NewOracleDB creates a new Oracle database connection pool
func NewOracleDB(cfg OracleConfig) (*sql.DB, error) {
	db, err := sql.Open("godror", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
