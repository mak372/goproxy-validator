package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_project/config"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init() error {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open DB: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}

	return createTables()
}

func createTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS contracts (
			key        TEXT PRIMARY KEY,
			endpoint   TEXT NOT NULL,
			method     TEXT NOT NULL,
			target     TEXT NOT NULL,
			request    JSONB NOT NULL,
			response   JSONB NOT NULL,
			created_at TEXT NOT NULL DEFAULT ''
		);

		ALTER TABLE contracts ADD COLUMN IF NOT EXISTS created_at TEXT NOT NULL DEFAULT '';
		ALTER TABLE contracts ADD COLUMN IF NOT EXISTS protocol TEXT NOT NULL DEFAULT 'http';
		ALTER TABLE contracts ADD COLUMN IF NOT EXISTS proto_source TEXT NOT NULL DEFAULT '';
		ALTER TABLE contracts ADD COLUMN IF NOT EXISTS grpc_service TEXT NOT NULL DEFAULT '';
		ALTER TABLE contracts ADD COLUMN IF NOT EXISTS grpc_method TEXT NOT NULL DEFAULT '';

		CREATE TABLE IF NOT EXISTS violations (
			id        SERIAL PRIMARY KEY,
			timestamp TEXT NOT NULL,
			endpoint  TEXT NOT NULL,
			method    TEXT NOT NULL,
			direction TEXT NOT NULL,
			violations JSONB NOT NULL
		);
	`)
	return err
}

// Contracts

func SaveContract(c *config.Contract) error {
	req, _ := json.Marshal(c.Request)
	res, _ := json.Marshal(c.Response)
	key := c.Method + " " + c.Endpoint
	c.CreatedAt = time.Now().Format(time.RFC3339)
	if c.Protocol == "" {
		c.Protocol = "http"
	}
	_, err := DB.Exec(`
		INSERT INTO contracts (key, endpoint, method, target, request, response, created_at, protocol, proto_source, grpc_service, grpc_method)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (key) DO UPDATE
		SET endpoint=$2, method=$3, target=$4, request=$5, response=$6, created_at=$7, protocol=$8, proto_source=$9, grpc_service=$10, grpc_method=$11
	`, key, c.Endpoint, c.Method, c.Target, req, res, c.CreatedAt, c.Protocol, c.ProtoSource, c.GRPCService, c.GRPCMethod)
	return err
}

func LoadAllContracts() (map[string]*config.Contract, error) {
	rows, err := DB.Query(`SELECT key, endpoint, method, target, request, response, created_at, protocol, proto_source, grpc_service, grpc_method FROM contracts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contracts := make(map[string]*config.Contract)
	for rows.Next() {
		var key string
		var c config.Contract
		var req, res []byte
		if err := rows.Scan(&key, &c.Endpoint, &c.Method, &c.Target, &req, &res, &c.CreatedAt, &c.Protocol, &c.ProtoSource, &c.GRPCService, &c.GRPCMethod); err != nil {
			return nil, err
		}
		json.Unmarshal(req, &c.Request)
		json.Unmarshal(res, &c.Response)
		contracts[key] = &c
	}
	return contracts, nil
}

func DeleteContract(key string) error {
	_, err := DB.Exec(`DELETE FROM contracts WHERE key = $1`, key)
	return err
}

// Violations

type ViolationRecord struct {
	Timestamp  string        `json:"timestamp"`
	Endpoint   string        `json:"endpoint"`
	Method     string        `json:"method"`
	Direction  string        `json:"direction"`
	Violations []interface{} `json:"violations"`
}

func SaveViolation(endpoint, method, direction string, violations interface{}) error {
	v, _ := json.Marshal(violations)
	_, err := DB.Exec(`
		INSERT INTO violations (timestamp, endpoint, method, direction, violations)
		VALUES ($1, $2, $3, $4, $5)
	`, time.Now().Format(time.RFC3339), endpoint, method, direction, v)
	return err
}

func LoadAllViolations() ([]ViolationRecord, error) {
	rows, err := DB.Query(`SELECT timestamp, endpoint, method, direction, violations FROM violations ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ViolationRecord
	for rows.Next() {
		var r ViolationRecord
		var v []byte
		if err := rows.Scan(&r.Timestamp, &r.Endpoint, &r.Method, &r.Direction, &v); err != nil {
			return nil, err
		}
		json.Unmarshal(v, &r.Violations)
		records = append(records, r)
	}
	if records == nil {
		records = []ViolationRecord{}
	}
	return records, nil
}
