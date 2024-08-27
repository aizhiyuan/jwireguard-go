package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
)

type CliConfig struct {
	CliID      string `json:"cli_id"`
	SerID      string `json:"ser_id"`
	CliSN      string `json:"cli_sn"`
	CliName    string `json:"cli_name"`
	SerName    string `json:"ser_name"`
	CliAddress string `json:"cli_address"`
	CliMapping string `json:"cli_mapping"`
	CliStatus  string `json:"cli_status"`
	Timestamp  int64  `json:"ts"`
	EditStatus string `json:"edit_stauts"`
}

// ----------------------------------------------------------------------------------------------------------
// 创建用户配置表
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) CreateCliConfig(db *sql.DB) {
	if !c.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS cli_config (
            "cli_id" TEXT NOT NULL PRIMARY KEY,
            "ser_id" TEXT NOT NULL,
			"cli_sn" TEXT NOT NULL,
			"cli_name" TEXT NOT NULL,
			"ser_name" TEXT NOT NULL,
            "cli_address" TEXT NOT NULL,
			"cli_mapping" TEXT NOT NULL,
			"cli_status" TEXT NOT NULL,
			"ts" TEXT NOT NULL,
			"edit_stauts" TEXT NOT NULL
        );`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Fatalln("[CreateCliConfig] Error creating table:", err)
			return
		}
		log.Println("[CreateCliConfig] Table 'cli_config' created successfully!")
	} else {
		log.Println("[CreateCliConfig] Table 'cli_config' already exists.")
	}
}

// ----------------------------------------------------------------------------------------------------------
// 添加用户配置
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) InsertCliConfig(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO cli_config (cli_id, ser_id, cli_sn, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.CliID, c.SerID, c.CliSN, c.CliName, c.SerName, c.CliAddress, c.CliMapping, c.CliStatus, c.Timestamp, c.EditStatus)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过 CliID 查询用户配置
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) GetCliConfigByCliID(db *sql.DB) error {
	query := "SELECT cli_id, ser_id, cli_sn, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts FROM cli_config WHERE cli_id = ?"
	row := db.QueryRow(query, c.CliID)

	err := row.Scan(&c.CliID, &c.SerID, &c.CliSN, &c.CliName, &c.SerName, &c.CliAddress, &c.CliMapping, &c.CliStatus, &c.Timestamp, &c.EditStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("CliConfig with CliID %s not found", c.CliID)
		}
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过 SerID 查询用户配置
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) GetCliConfigBySerID(db *sql.DB) ([]CliConfig, error) {
	query := "SELECT cli_id, ser_id, cli_sn, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts FROM cli_config WHERE ser_id = ?"
	rows, err := db.Query(query, c.SerID) // 使用 db.Query 代替 db.QueryRow
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []CliConfig
	for rows.Next() {
		var config CliConfig
		err := rows.Scan(
			&config.CliID,
			&config.SerID,
			&config.CliSN,
			&config.CliName,
			&config.SerName,
			&config.CliAddress,
			&config.CliMapping,
			&config.CliStatus,
			&config.Timestamp,
			&config.EditStatus,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// ----------------------------------------------------------------------------------------------------------
// 获取 CliConfig 表中的所有数据
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) GetAllCliConfig(db *sql.DB) ([]CliConfig, error) {
	query := "SELECT cli_id, ser_id, cli_sn, cli_name, ser_name, cli_address, cli_mapping, cli_status, ts, edit_stauts FROM cli_config"
	rows, err := db.Query(query) // 使用 db.Query 代替 db.QueryRow
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []CliConfig
	for rows.Next() {
		var config CliConfig
		err := rows.Scan(
			&config.CliID,
			&config.SerID,
			&config.CliSN,
			&config.CliName,
			&config.SerName,
			&config.CliAddress,
			&config.CliMapping,
			&config.CliStatus,
			&config.Timestamp,
			&config.EditStatus,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}

// ----------------------------------------------------------------------------------------------------------
// 更新用户配置中的部分数据
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) UpdateCliConfig(db *sql.DB) error {
	if c.CliID == "" {
		return errors.New("cli_id cannot be empty")
	}

	// 用于存储 SQL 语句片段和对应参数的切片
	setClauses := []string{}
	args := []interface{}{}

	// 动态添加不为空的字段
	if c.SerID != "" {
		setClauses = append(setClauses, "ser_id = ?")
		args = append(args, c.SerID)
	}
	if c.CliSN != "" {
		setClauses = append(setClauses, "cli_sn = ?")
		args = append(args, c.CliSN)
	}
	if c.CliName != "" {
		setClauses = append(setClauses, "cli_name = ?")
		args = append(args, c.CliName)
	}
	if c.SerName != "" {
		setClauses = append(setClauses, "ser_name = ?")
		args = append(args, c.SerName)
	}
	if c.CliAddress != "" {
		setClauses = append(setClauses, "cli_address = ?")
		args = append(args, c.CliAddress)
	}
	if c.CliMapping != "" {
		setClauses = append(setClauses, "cli_mapping = ?")
		args = append(args, c.CliMapping)
	}
	if c.CliStatus != "" {
		setClauses = append(setClauses, "cli_status = ?")
		args = append(args, c.CliStatus)
	}
	if c.Timestamp != 0 {
		setClauses = append(setClauses, "ts = ?")
		args = append(args, c.Timestamp)
	}
	if c.EditStatus != "" {
		setClauses = append(setClauses, "edit_stauts = ?")
		args = append(args, c.EditStatus)
	}

	// 如果没有任何字段需要更新
	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	// 构建最终的 SQL 语句
	query := fmt.Sprintf("UPDATE cli_config SET %s WHERE cli_id = ?", strings.Join(setClauses, ", "))
	args = append(args, c.CliID)

	// 准备并执行 SQL 语句
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

// ----------------------------------------------------------------------------------------------------------
// 删除用户配置中的数据
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) DeleteCliConfig(db *sql.DB) error {
	stmt, err := db.Prepare("DELETE FROM cli_config WHERE cli_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(c.CliID)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 检查表格是否存在
// ----------------------------------------------------------------------------------------------------------
func (c *CliConfig) TableExists(db *sql.DB) bool {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='cli_config'"
	row := db.QueryRow(query)

	var name string
	err := row.Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Fatalln("[TableExists] Error checking table existence:", err)
		return false
	}

	return err == nil
}
