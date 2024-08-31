package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
)

type Subnet struct {
	SerID   sql.NullString `json:"ser_id"`
	SerName sql.NullString `json:"ser_name"`
	SerNum  sql.NullInt32  `json:"ser_num"`
	CliNum  sql.NullInt32  `json:"cli_num"`
}

type ExportedSubnet struct {
	SerID   string `json:"ser_id"`
	SerName string `json:"ser_name"`
	SerNum  int32  `json:"ser_num"`
	CliNum  int32  `json:"cli_num"`
}

// ----------------------------------------------------------------------------------------------------------
// 创建子网表
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) CreateSubnet(db *sql.DB) {
	if !s.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS subnet (
            "ser_id" TEXT NOT NULL PRIMARY KEY,
            "ser_name" TEXT,
			"ser_num" INTEGER,
			"cli_num" INTEGER
        );`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Fatalln("[CreateSubnet] Error creating table:", err)
			return
		}
		// log.Println("[CreateSubnet] Table 'subnet' created successfully!")
	} else {
		// log.Println("[CreateSubnet] Table 'subnet' already exists.")
	}
}

// ToExported 负责将 CliConfig 转换为 ExportedCliConfig
func (s *Subnet) ToExported() ExportedSubnet {
	return ExportedSubnet{
		SerID:   nullStringToString(s.SerID),
		SerName: nullStringToString(s.SerName),
		SerNum:  nullInt32ToInt32(s.SerNum),
		CliNum:  nullInt32ToInt32(s.CliNum),
	}
}

// 将 ExportedCliConfig 转换为 CliConfig
func (exported *ExportedSubnet) ConvertToSubnet() Subnet {
	return Subnet{
		SerID:   sql.NullString{String: exported.SerID, Valid: exported.SerID != ""},
		SerName: sql.NullString{String: exported.SerName, Valid: exported.SerName != ""},
		SerNum:  sql.NullInt32{Int32: exported.SerNum, Valid: exported.SerNum != -1},
		CliNum:  sql.NullInt32{Int32: exported.CliNum, Valid: exported.CliNum != -1},
	}
}

// ----------------------------------------------------------------------------------------------------------
// 添加子网
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) InsertSubnet(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO subnet (ser_id, ser_name, ser_num, cli_num) VALUES(?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.SerID.String, s.SerName.String, s.SerNum.Int32, s.CliNum.Int32)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过 SerID 查询子网信息
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) GetSubnetBySerId(db *sql.DB) error {
	query := "SELECT ser_id, ser_name, ser_num, cli_num FROM subnet WHERE ser_id = ?"
	row := db.QueryRow(query, s.SerID.String)

	err := row.Scan(&s.SerID, &s.SerName, &s.SerNum, &s.CliNum)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("Subnet with SerID %s not found", s.SerID.String)
		}
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过多个 SerID 查询子网信息
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) GetSubnetBySerIDs(db *sql.DB, serids []string) ([]Subnet, error) {
	// 构建包含占位符的 SQL 语句
	placeholders := strings.Repeat("?,", len(serids))
	placeholders = placeholders[:len(placeholders)-1] // 去掉最后的逗号
	query := fmt.Sprintf("SELECT ser_id, ser_name, ser_num, cli_num FROM subnet WHERE ser_id IN (%s)", placeholders)

	// 将 serids 转换为 interface{} 类型的 slice 以便用于 Exec
	args := make([]interface{}, len(serids))
	for i, id := range serids {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subnets []Subnet
	for rows.Next() {
		var subnet Subnet
		err := rows.Scan(&subnet.SerID, &subnet.SerName, &subnet.SerNum, &subnet.CliNum)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, subnet)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subnets, nil
}

// ----------------------------------------------------------------------------------------------------------
// 获取 Subnet 表中的所有数据
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) GetAllSubnet(db *sql.DB) ([]Subnet, error) {
	query := "SELECT ser_id, ser_name, ser_num, cli_num FROM subnet"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subnets []Subnet
	for rows.Next() {
		var subnet Subnet
		err := rows.Scan(&subnet.SerID, &subnet.SerName, &subnet.SerNum, &subnet.CliNum)
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, subnet)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subnets, nil
}

// ----------------------------------------------------------------------------------------------------------
// 更新子网中的部分数据
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) UpdateSubnet(db *sql.DB) error {
	if s.SerID.String == "" {
		return errors.New("ser_id cannot be empty")
	}

	// 用于存储 SQL 语句片段和对应参数的切片
	setClauses := []string{}
	args := []interface{}{}

	// 动态添加不为空的字段
	if s.SerName.String != "" {
		setClauses = append(setClauses, "ser_name = ?")
		args = append(args, s.SerName.String)
	}
	if s.SerNum.Int32 != 0 {
		setClauses = append(setClauses, "ser_num = ?")
		args = append(args, s.SerNum.Int32)
	}
	if s.CliNum.Int32 != 0 {
		setClauses = append(setClauses, "cli_num = ?")
		args = append(args, s.CliNum.Int32)
	}

	// 如果没有任何字段需要更新
	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	// 构建最终的 SQL 语句
	query := fmt.Sprintf("UPDATE subnet SET %s WHERE ser_id = ?", strings.Join(setClauses, ", "))
	args = append(args, s.SerID.String)

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
// 删除子网中的数据
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) DeleteSubnet(db *sql.DB) error {
	stmt, err := db.Prepare("DELETE FROM subnet WHERE ser_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.SerID.String)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 检查表格是否存在
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) TableExists(db *sql.DB) bool {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='subnet';"
	var name string
	err := db.QueryRow(query).Scan(&name)
	return err == nil
}

// ----------------------------------------------------------------------------------------------------------
// 获取新的子网网段（SerNum），返回数据库中不存在的 SerNum 值
// ----------------------------------------------------------------------------------------------------------
func (s *Subnet) GetNewSubnetNumber(db *sql.DB) (int32, error) {
	// 查询现有的 SerNum 值
	query := "SELECT ser_num FROM subnet"
	rows, err := db.Query(query)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	// 使用 map 存储已存在的 SerNum
	existingSerNums := make(map[int32]bool)
	for rows.Next() {
		var serNum int32
		if err := rows.Scan(&serNum); err != nil {
			return -1, err
		}
		existingSerNums[serNum] = true
	}

	// 检查 0 ~ 254 的 SerNum 值，返回第一个不存在的值
	for i := int32(0); i <= 254; i++ {
		if _, exists := existingSerNums[i]; !exists {
			return i, nil
		}
	}

	// 如果所有 SerNum 都已被使用，返回错误
	return -1, errors.New("no available subnet numbers")
}
