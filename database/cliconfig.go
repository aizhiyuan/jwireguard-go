package database

import (
	"database/sql"
	"errors"
	"fmt"
	"jwireguard/global"
	"strings"
)

type CliConfig struct {
	CliID        sql.NullString `json:"cli_id"`
	SerID        sql.NullString `json:"ser_id"`
	CliSN        sql.NullString `json:"cli_sn"`
	CliMac       sql.NullString `json:"cli_mac"`
	CliName      sql.NullString `json:"cli_name"`
	SerName      sql.NullString `json:"ser_name"`
	CliAddress   sql.NullString `json:"cli_address"`
	CliMapping   sql.NullString `json:"cli_mapping"`
	CliStatus    sql.NullString `json:"cli_status"`
	Timestamp    sql.NullInt64  `json:"ts"`
	EditStatus   sql.NullInt32  `json:"edit_stauts"`
	OnlineStatus sql.NullString `json:"online_status"`
}

type ExportedCliConfig struct {
	CliID        string `json:"cli_id"`
	SerID        string `json:"ser_id"`
	CliSN        string `json:"cli_sn"`
	CliMac       string `json:"cli_mac"`
	CliName      string `json:"cli_name"`
	SerName      string `json:"ser_name"`
	CliAddress   string `json:"cli_address"`
	CliMapping   string `json:"cli_mapping"`
	CliStatus    string `json:"cli_status"`
	Timestamp    int64  `json:"ts"`
	EditStatus   int32  `json:"edit_stauts"`
	OnlineStatus string `json:"online_status"`
}

// ConvertToCliConfig converts ExportedCliConfig to CliConfig
func (exported *ExportedCliConfig) ConvertToCliConfig() CliConfig {
	return CliConfig{
		CliID:        sql.NullString{String: exported.CliID, Valid: exported.CliID != ""},
		SerID:        sql.NullString{String: exported.SerID, Valid: exported.SerID != ""},
		CliSN:        sql.NullString{String: exported.CliSN, Valid: exported.CliSN != ""},
		CliMac:       sql.NullString{String: exported.CliMac, Valid: exported.CliMac != ""},
		CliName:      sql.NullString{String: exported.CliName, Valid: exported.CliName != ""},
		SerName:      sql.NullString{String: exported.SerName, Valid: exported.SerName != ""},
		CliAddress:   sql.NullString{String: exported.CliAddress, Valid: exported.CliAddress != ""},
		CliMapping:   sql.NullString{String: exported.CliMapping, Valid: exported.CliMapping != ""},
		CliStatus:    sql.NullString{String: exported.CliStatus, Valid: exported.CliStatus != ""},
		Timestamp:    sql.NullInt64{Int64: exported.Timestamp, Valid: exported.Timestamp != -1},
		EditStatus:   sql.NullInt32{Int32: exported.EditStatus, Valid: exported.EditStatus != -1},
		OnlineStatus: sql.NullString{String: exported.OnlineStatus, Valid: exported.OnlineStatus != ""},
	}
}

// CreateCliConfig creates the cli_config table in MySQL
func (c *CliConfig) CreateCliConfig(db *sql.DB) {
	if !c.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS cli_config (
            cli_id VARCHAR(255) NOT NULL PRIMARY KEY,
            ser_id VARCHAR(255),
            cli_sn VARCHAR(255),
            cli_mac VARCHAR(255), 
            cli_name VARCHAR(255),
            ser_name VARCHAR(255),
            cli_address VARCHAR(255),
            cli_mapping VARCHAR(255),
            cli_status VARCHAR(255),
            ts BIGINT,
            edit_stauts INT,
            online_status VARCHAR(255)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			global.Log.Errorln("[CreateCliConfig] Error creating table:", err)
			return
		}
	} else {
		global.Log.Debugln("[CreateCliConfig] Table 'cli_config' already exists.")
	}
}

// ToExported converts CliConfig to ExportedCliConfig
func (c *CliConfig) ToExported() ExportedCliConfig {
	return ExportedCliConfig{
		CliID:        nullStringToString(c.CliID),
		SerID:        nullStringToString(c.SerID),
		CliSN:        nullStringToString(c.CliSN),
		CliMac:       nullStringToString(c.CliMac),
		CliName:      nullStringToString(c.CliName),
		SerName:      nullStringToString(c.SerName),
		CliAddress:   nullStringToString(c.CliAddress),
		CliMapping:   nullStringToString(c.CliMapping),
		CliStatus:    nullStringToString(c.CliStatus),
		Timestamp:    nullInt64ToInt64(c.Timestamp),
		EditStatus:   nullInt32ToInt32(c.EditStatus),
		OnlineStatus: nullStringToString(c.OnlineStatus),
	}
}

// InsertCliConfig inserts a new record into cli_config
func (c *CliConfig) InsertCliConfig(db *sql.DB) error {
	// 包含 cli_mac 字段
	stmt, err := db.Prepare("INSERT INTO cli_config (cli_id, ser_id, cli_sn, cli_mac, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts, online_status) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 添加 cli_mac.String 参数
	_, err = stmt.Exec(
		c.CliID.String,
		c.SerID.String,
		c.CliSN.String,
		c.CliMac.String,
		c.CliName.String,
		c.SerName.String,
		c.CliAddress.String,
		c.CliMapping.String,
		c.CliStatus.String,
		c.Timestamp.Int64,
		c.EditStatus.Int32,
		c.OnlineStatus.String)
	if err != nil {
		return err
	}

	return nil
}

// GetCliConfigByCliID retrieves a record by cli_id
func (c *CliConfig) GetCliConfigByCliID(db *sql.DB) error {
	query := "SELECT cli_id, ser_id, cli_sn, cli_mac, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts, online_status FROM cli_config WHERE cli_id = ?"
	row := db.QueryRow(query, c.CliID.String)

	// 添加 cli_mac 字段扫描
	err := row.Scan(
		&c.CliID,
		&c.SerID,
		&c.CliSN,
		&c.CliMac,
		&c.CliName,
		&c.SerName,
		&c.CliAddress,
		&c.CliMapping,
		&c.CliStatus,
		&c.Timestamp,
		&c.EditStatus,
		&c.OnlineStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("CliConfig with CliID %s not found", c.CliID.String)
		}
		return err
	}

	return nil
}

// GetCliConfigBySerID retrieves records by ser_id
func (c *CliConfig) GetCliConfigBySerID(db *sql.DB) ([]CliConfig, error) {
	query := "SELECT cli_id, ser_id, cli_sn, cli_mac, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts, online_status FROM cli_config WHERE ser_id = ? ORDER BY INET_ATON(cli_address)"
	rows, err := db.Query(query, c.SerID.String)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []CliConfig
	for rows.Next() {
		var config CliConfig
		// 添加 cli_mac 字段扫描
		err := rows.Scan(
			&config.CliID,
			&config.SerID,
			&config.CliSN,
			&config.CliMac,
			&config.CliName,
			&config.SerName,
			&config.CliAddress,
			&config.CliMapping,
			&config.CliStatus,
			&config.Timestamp,
			&config.EditStatus,
			&config.OnlineStatus,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// GetAllCliConfig retrieves all records from cli_config
func (c *CliConfig) GetAllCliConfig(db *sql.DB) ([]CliConfig, error) {
	query := "SELECT cli_id, ser_id, cli_sn, cli_mac, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts, online_status FROM cli_config"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []CliConfig
	for rows.Next() {
		var config CliConfig
		// 添加 cli_mac 字段扫描
		err := rows.Scan(
			&config.CliID,
			&config.SerID,
			&config.CliSN,
			&config.CliMac,
			&config.CliName,
			&config.SerName,
			&config.CliAddress,
			&config.CliMapping,
			&config.CliStatus,
			&config.Timestamp,
			&config.EditStatus,
			&config.OnlineStatus,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// UpdateCliConfig updates a record in cli_config
func (c *CliConfig) UpdateCliConfig(db *sql.DB) error {
	if c.CliID.String == "" {
		return errors.New("cli_id cannot be empty")
	}

	setClauses := []string{}
	args := []interface{}{}

	if c.SerID.String != "" {
		setClauses = append(setClauses, "ser_id = ?")
		args = append(args, c.SerID.String)
	}
	if c.CliSN.String != "" {
		setClauses = append(setClauses, "cli_sn = ?")
		args = append(args, c.CliSN.String)
	}
	if c.CliMac.String != "" {
		setClauses = append(setClauses, "cli_mac = ?")
		args = append(args, c.CliMac.String)
	}
	if c.CliName.String != "" {
		setClauses = append(setClauses, "cli_name = ?")
		args = append(args, c.CliName.String)
	}
	if c.SerName.String != "" {
		setClauses = append(setClauses, "ser_name = ?")
		args = append(args, c.SerName.String)
	}
	if c.CliAddress.String != "" {
		setClauses = append(setClauses, "cli_address = ?")
		args = append(args, c.CliAddress.String)
	}
	if c.CliMapping.String != "" {
		setClauses = append(setClauses, "cli_mapping = ?")
		args = append(args, c.CliMapping.String)
	}
	if c.CliStatus.String != "" {
		setClauses = append(setClauses, "cli_status = ?")
		args = append(args, c.CliStatus.String)
	}
	if c.Timestamp.Int64 != -1 {
		setClauses = append(setClauses, "ts = ?")
		args = append(args, c.Timestamp.Int64)
	}
	if c.EditStatus.Int32 != -1 {
		setClauses = append(setClauses, "edit_stauts = ?")
		args = append(args, c.EditStatus.Int32)
	}
	if c.OnlineStatus.String != "" {
		setClauses = append(setClauses, "online_status = ?")
		args = append(args, c.OnlineStatus.String)
	}

	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	query := fmt.Sprintf("UPDATE cli_config SET %s WHERE cli_id = ?", strings.Join(setClauses, ", "))
	args = append(args, c.CliID.String)

	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return err
	}

	return nil
}

// DeleteCliConfig deletes a record from cli_config
func (c *CliConfig) DeleteCliConfig(db *sql.DB) error {
	stmt, err := db.Prepare("DELETE FROM cli_config WHERE cli_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.CliID.String)
	if err != nil {
		return err
	}

	return nil
}

// TableExists checks if the table exists in MySQL
func (c *CliConfig) TableExists(db *sql.DB) bool {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'cli_config'"
	row := db.QueryRow(query)

	var name string
	err := row.Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		global.Log.Errorln("[TableExists] Error checking table existence:", err)
		return false
	}

	return true
}
