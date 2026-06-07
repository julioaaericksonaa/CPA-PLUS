package store

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/plusmanager/model"
	_ "modernc.org/sqlite"
)

type ModelPrice = model.ModelPrice

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	dbDir := filepath.Dir(dbPath)
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS model_prices (
		model TEXT PRIMARY KEY,
		input_per_mtok REAL NOT NULL,
		output_per_mtok REAL NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) ListModelPrices() ([]ModelPrice, error) {
	rows, err := s.db.Query(`SELECT model, input_per_mtok, output_per_mtok FROM model_prices ORDER BY model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := []ModelPrice{}
	for rows.Next() {
		var price ModelPrice
		if err := rows.Scan(&price.Model, &price.InputPerMTok, &price.OutputPerMTok); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prices, nil
}

func (s *Store) ReplaceModelPrices(prices []ModelPrice) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM model_prices`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO model_prices (model, input_per_mtok, output_per_mtok) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, price := range prices {
		if _, err := stmt.Exec(price.Model, price.InputPerMTok, price.OutputPerMTok); err != nil {
			return err
		}
	}
	return tx.Commit()
}
