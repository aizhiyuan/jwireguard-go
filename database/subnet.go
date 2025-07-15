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

// CreateSubnet creates the subnet table in MySQL
func (s *Subnet) CreateSubnet(db *sql.DB) {
	if !s.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS subnet (
            ser_id VARCHAR(255) NOT NULL PRIMARY KEY,
            ser_name VARCHAR(255),
            ser_num INT,
            cli_num INT,
            INDEX idx_ser_num (ser_num)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Println("[CreateSubnet] Error creating table:", err)
			return
		}
	}
}

// ToExported converts Subnet to ExportedSubnet
func (s *Subnet) ToExported() ExportedSubnet {
	return ExportedSubnet{
		SerID:   nullStringToString(s.SerID),
		SerName: nullStringToString(s.SerName),
		SerNum:  nullInt32ToInt32(s.SerNum),
		CliNum:  nullInt32ToInt32(s.CliNum),
	}
}

// ConvertToSubnet converts ExportedSubnet to Subnet
func (exported *ExportedSubnet) ConvertToSubnet() Subnet {
	return Subnet{
		SerID:   sql.NullString{String: exported.SerID, Valid: exported.SerID != ""},
		SerName: sql.NullString{String: exported.SerName, Valid: exported.SerName != ""},
		SerNum:  sql.NullInt32{Int32: exported.SerNum, Valid: exported.SerNum != -1},
		CliNum:  sql.NullInt32{Int32: exported.CliNum, Valid: exported.CliNum != -1},
	}
}

// InsertSubnet inserts a new subnet record
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

// GetSubnetBySerId retrieves a subnet by ser_id
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

// GetSubnetBySerIDs retrieves multiple subnets by ser_ids
func (s *Subnet) GetSubnetBySerIDs(db *sql.DB, serids []string) ([]Subnet, error) {
	if len(serids) == 0 {
		return nil, errors.New("no ser_ids provided")
	}

	placeholders := strings.Repeat("?,", len(serids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf("SELECT ser_id, ser_name, ser_num, cli_num FROM subnet WHERE ser_id IN (%s) ORDER BY ser_name", placeholders)

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

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subnets, nil
}

// GetAllSubnet retrieves all subnet records
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

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subnets, nil
}

// UpdateSubnet updates a subnet record
func (s *Subnet) UpdateSubnet(db *sql.DB) error {
	if s.SerID.String == "" {
		return errors.New("ser_id cannot be empty")
	}

	setClauses := []string{}
	args := []interface{}{}

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

	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	query := fmt.Sprintf("UPDATE subnet SET %s WHERE ser_id = ?", strings.Join(setClauses, ", "))
	args = append(args, s.SerID.String)

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

// DeleteSubnet deletes a subnet record
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

// TableExists checks if the subnet table exists in MySQL
func (s *Subnet) TableExists(db *sql.DB) bool {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'subnet'"
	var name string
	err := db.QueryRow(query).Scan(&name)
	return err == nil
}

// GetNewSubnetNumber finds an available subnet number
func (s *Subnet) GetNewSubnetNumber(db *sql.DB) (int32, error) {
	// More efficient query to find gaps in the sequence
	query := `SELECT MIN(t1.ser_num) + 1 
              FROM subnet t1
              WHERE NOT EXISTS (
                  SELECT 1 FROM subnet t2 
                  WHERE t2.ser_num = t1.ser_num + 1
              ) AND t1.ser_num < 254`

	var availableNum sql.NullInt32
	err := db.QueryRow(query).Scan(&availableNum)
	if err != nil && err != sql.ErrNoRows {
		return -1, err
	}

	// If we found a gap, return it
	if availableNum.Valid && availableNum.Int32 <= 254 {
		return availableNum.Int32, nil
	}

	// If no gaps found, check if 0 is available
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM subnet WHERE ser_num = 0").Scan(&count)
	if err != nil {
		return -1, err
	}
	if count == 0 {
		return 0, nil
	}

	// Otherwise find the max number + 1
	var maxNum sql.NullInt32
	err = db.QueryRow("SELECT MAX(ser_num) FROM subnet").Scan(&maxNum)
	if err != nil {
		return -1, err
	}

	if maxNum.Valid && maxNum.Int32 < 254 {
		return maxNum.Int32 + 1, nil
	}

	return -1, errors.New("no available subnet numbers")
}
