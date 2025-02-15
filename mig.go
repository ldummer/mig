package migpgx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

type MigrationRunner struct {
	pgxConn *pgx.Conn
	migDir  string
}

func NewMigrationRunner(conn *pgx.Conn, migDir string) *MigrationRunner {
	return &MigrationRunner{
		pgxConn: conn,
		migDir:  migDir,
	}
}

func (m *MigrationRunner) ApplyMigrations(ctx context.Context, migTable string ) error {
	if err := m.ensureMigrationsTable(ctx, migTable); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	appliedMigrations, err := m.getAppliedMigrations(ctx, migTable)
	if err != nil {
		return fmt.Errorf("failed to retrieve applied migrations: %w", err)
	}

	files, err := getMigrationFiles(m.migDir)
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	for _, file := range files {
		if _, alreadyApplied := appliedMigrations[file]; alreadyApplied {
			continue
		}

		sqlContent, err := os.ReadFile(filepath.Join(m.migDir, file))
		if err != nil {
			return fmt.Errorf("error reading migration file %s: %w", file, err)
		}

		if err := m.applyMigration(ctx, file, string(sqlContent), migTable); err != nil {
			return fmt.Errorf("error applying migration %s: %w", file, err)
		}
	}

	return nil
}

func (m *MigrationRunner) ensureMigrationsTable(ctx context.Context, migTable string) error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		filename TEXT UNIQUE NOT NULL,
		applied_at TIMESTAMP DEFAULT NOW()
	);`, migTable)

	return m.executeSQL(ctx, query)
}


func (m *MigrationRunner) getAppliedMigrations(ctx context.Context, migTable string) (map[string]struct{}, error) {
	query := "SELECT filename FROM " + migTable + ";"
	applied := make(map[string]struct{})

	rows, err := m.pgxConn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[filename] = struct{}{}
	}

	return applied, nil
}

func (m *MigrationRunner) applyMigration(ctx context.Context, filename, sqlContent, migTable string) error {
	tx, err := m.pgxConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(123456)"); err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	if _, err := tx.Exec(ctx, sqlContent); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", filename, err)
	}

	insertQuery := "INSERT INTO " + migTable + " (filename) VALUES ($1);"
	if _, err := tx.Exec(ctx, insertQuery, filename); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", filename, err)
	}

	return tx.Commit(ctx)
}

func getMigrationFiles(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading migration directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	sort.Strings(sqlFiles)
	return sqlFiles, nil
}

func (m *MigrationRunner) executeSQL(ctx context.Context, query string) error {
	if _, err := m.pgxConn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	
	return nil
}
