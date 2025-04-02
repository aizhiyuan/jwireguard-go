package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
)

type User struct {
	UserID     sql.NullString `json:"user_id"`
	SerID      sql.NullString `json:"ser_id"`
	ParentID   sql.NullString `json:"parent_id"`
	UserName   sql.NullString `json:"user_name"`
	UserPasswd sql.NullString `json:"user_passwd"`
	UserType   sql.NullInt64  `json:"user_type"`
	UserStatus sql.NullString `json:"user_status"`
	UserEmail  sql.NullString `json:"user_email"`
}

type ExportedUser struct {
	UserID     string `json:"user_id"`
	SerID      string `json:"ser_id"`
	ParentID   string `json:"parent_id"`
	UserName   string `json:"user_name"`
	UserPasswd string `json:"user_passwd"`
	UserType   int64  `json:"user_type"`
	UserStatus string `json:"user_status"`
	UserEmail  string `json:"user_email"`
}

// ----------------------------------------------------------------------------------------------------------
// 创建用户表
// ----------------------------------------------------------------------------------------------------------
func (u *User) CreateUser(db *sql.DB) {
	if !u.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS user (
            "user_id" TEXT NOT NULL PRIMARY KEY,
			"ser_id" TEXT,
            "parent_id" TEXT,
            "user_name" TEXT,
			"user_passwd" TEXT,
			"user_type" INTEGER,
			"user_status" TEXT,
			"user_email" TEXT
        );`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Println("[CreateUser] Error creating table:", err)
			return
		}
		// log.Println("[CreateUser] Table 'user' created successfully!")
	} else {
		// log.Println("[CreateUser] Table 'user' already exists.")
	}
}

// ToExported 负责将 CliConfig 转换为 ExportedCliConfig
func (u *User) ToExported() ExportedUser {
	return ExportedUser{
		UserID:     nullStringToString(u.UserID),
		SerID:      nullStringToString(u.SerID),
		ParentID:   nullStringToString(u.ParentID),
		UserName:   nullStringToString(u.UserName),
		UserPasswd: nullStringToString(u.UserPasswd),
		UserType:   nullInt64ToInt64(u.UserType),
		UserStatus: nullStringToString(u.UserStatus),
		UserEmail:  nullStringToString(u.UserEmail),
	}
}

// 将 ExportedCliConfig 转换为 CliConfig
func (exported *ExportedUser) ConvertToUser() User {
	return User{
		UserID:     sql.NullString{String: exported.UserID, Valid: exported.UserID != ""},
		SerID:      sql.NullString{String: exported.SerID, Valid: exported.SerID != ""},
		ParentID:   sql.NullString{String: exported.ParentID, Valid: exported.ParentID != ""},
		UserName:   sql.NullString{String: exported.UserName, Valid: exported.UserName != ""},
		UserPasswd: sql.NullString{String: exported.UserPasswd, Valid: exported.UserPasswd != ""},
		UserType:   sql.NullInt64{Int64: exported.UserType, Valid: exported.UserType != 0},
		UserStatus: sql.NullString{String: exported.UserStatus, Valid: exported.UserStatus != ""},
		UserEmail:  sql.NullString{String: exported.UserEmail, Valid: exported.UserEmail != ""},
	}
}

// ----------------------------------------------------------------------------------------------------------
// 添加用户表
// ----------------------------------------------------------------------------------------------------------
func (u *User) InsertUser(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO user (user_id, ser_id, parent_id, user_name, user_passwd, user_type, user_status, user_email) VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.UserID.String, u.SerID.String, u.ParentID.String, u.UserName.String, u.UserPasswd.String, u.UserType.Int64, u.UserStatus.String, u.UserEmail.String)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过 UserID 查询用户信息
// ----------------------------------------------------------------------------------------------------------
func (u *User) GetUserByID(db *sql.DB) error {
	query := "SELECT user_id, ser_id, parent_id, user_name, user_passwd, user_type, user_status, user_email FROM user WHERE user_id = ?"
	row := db.QueryRow(query, u.UserID.String)

	err := row.Scan(&u.UserID, &u.SerID, &u.ParentID, &u.UserName, &u.UserPasswd, &u.UserType, &u.UserStatus, &u.UserEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("User with UserID %s not found", u.UserID.String)
		}
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过 UserName 查询用户信息
// ----------------------------------------------------------------------------------------------------------
func (u *User) GetUserByName(db *sql.DB) error {
	query := "SELECT user_id, ser_id, parent_id, user_name, user_passwd, user_type, user_status, user_email FROM user WHERE user_name = ?"
	row := db.QueryRow(query, u.UserName.String)

	err := row.Scan(&u.UserID, &u.SerID, &u.ParentID, &u.UserName, &u.UserPasswd, &u.UserType, &u.UserStatus, &u.UserEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("User with UserName %s not found", u.UserName.String)
		}
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 通过多个 UserID 查询子网信息，去除重复的 ser_id
// ----------------------------------------------------------------------------------------------------------
func (u *User) GetSubnetIdsByUserIds(db *sql.DB, userIds []string) ([]string, error) {
	// 构建查询 user_subnet_map 表以获取子网ID的 SQL 语句
	placeholders := strings.Repeat("?,", len(userIds))
	placeholders = placeholders[:len(placeholders)-1] // 去掉最后的逗号
	query := fmt.Sprintf("SELECT DISTINCT ser_id FROM user WHERE user_id IN (%s)", placeholders)

	// 将 userIds 转换为 interface{} 类型的 slice 以便用于 Exec
	args := make([]interface{}, len(userIds))
	for i, id := range userIds {
		args[i] = id
	}
	// fmt.Println("query:", query)
	// fmt.Println("args:", args)
	// 执行查询获取子网ID
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	subnetIDSet := make(map[string]struct{}) // 使用 map 记录已扫描的 subnet_id
	for rows.Next() {
		var subnetID sql.NullString
		if err := rows.Scan(&subnetID); err != nil {
			return nil, err
		}

		// 如果字符串已经存在于 map 中，则跳过
		if _, exists := subnetIDSet[subnetID.String]; exists {
			continue
		}

		if subnetID.String == "" {
			continue
		}

		subnetIDSet[subnetID.String] = struct{}{} // 将 subnet_id 添加到 set 中
		result = append(result, subnetID.String)
		// fmt.Println("subnetID:", subnetID.String)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// 如果没有获取到任何子网ID，则返回空结果
	if len(subnetIDSet) == 0 {
		return []string{}, nil
	}

	return result, nil

}

// ----------------------------------------------------------------------------------------------------------
// 获取 User 表中的所有数据
// ----------------------------------------------------------------------------------------------------------
func (u *User) GetAllUsers(db *sql.DB) ([]User, error) {
	query := "SELECT user_id, ser_id, parent_id, user_name, user_passwd, user_type, user_status, user_email FROM user"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.UserID, &u.SerID, &user.ParentID, &user.UserName, &user.UserPasswd, &user.UserType, &user.UserStatus, &u.UserEmail)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// 检查是否有扫描错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// ----------------------------------------------------------------------------------------------------------
// 更新表中的部分数据
// ----------------------------------------------------------------------------------------------------------
func (u *User) UpdateUsers(db *sql.DB) error {
	if u.UserID.String == "" {
		return errors.New("user_id cannot be empty")
	}

	// 用于存储 SQL 语句片段和对应参数的切片
	setClauses := []string{}
	args := []interface{}{}

	// 动态添加不为空的字段
	if u.SerID.String != "" {
		setClauses = append(setClauses, "ser_id = ?")
		args = append(args, u.SerID.String)
	}
	if u.ParentID.String != "" {
		setClauses = append(setClauses, "parent_id = ?")
		args = append(args, u.ParentID.String)
	}
	if u.UserName.String != "" {
		setClauses = append(setClauses, "user_name = ?")
		args = append(args, u.UserName.String)
	}
	if u.UserPasswd.String != "" {
		setClauses = append(setClauses, "user_passwd = ?")
		args = append(args, u.UserPasswd.String)
	}
	if u.UserType.Int64 != 0 {
		setClauses = append(setClauses, "user_type = ?")
		args = append(args, u.UserType.Int64)
	}
	if u.UserStatus.String != "" {
		setClauses = append(setClauses, "user_status = ?")
		args = append(args, u.UserStatus.String)
	}
	if u.UserEmail.String != "" {
		setClauses = append(setClauses, "user_email = ?")
		args = append(args, u.UserEmail.String)
	}

	// 如果没有任何字段需要更新
	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	// 构建最终的 SQL 语句
	query := fmt.Sprintf("UPDATE user SET %s WHERE user_id = ?", strings.Join(setClauses, ", "))
	args = append(args, u.UserID.String)

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
// 删除表格中数据
// ----------------------------------------------------------------------------------------------------------
func (u *User) DeleteUsers(db *sql.DB) error {
	stmt, err := db.Prepare("DELETE FROM user WHERE user_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.UserID.String)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 检查表格是否存在
// ----------------------------------------------------------------------------------------------------------
func (u *User) TableExists(db *sql.DB) bool {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='user';"
	var name string
	err := db.QueryRow(query).Scan(&name)
	return err == nil
}

// ----------------------------------------------------------------------------------------------------------
// 查询符合条件的 user_id
// ----------------------------------------------------------------------------------------------------------
func (u *User) QueryUserIds(conn *sql.DB, targetUserId string) ([]string, error) {
	var userIds []string
	err := u.queryUserIdsRecursive(conn, targetUserId, &userIds)
	if err != nil {
		return nil, err
	}
	return userIds, nil
}

// ----------------------------------------------------------------------------------------------------------
// 递归查询符合条件的 user_id (私有函数)
// ----------------------------------------------------------------------------------------------------------
func (u *User) queryUserIdsRecursive(conn *sql.DB, currentUserId string, userIds *[]string) error {
	*userIds = append(*userIds, currentUserId)

	// 查询当前 user_id 的子节点
	rows, err := conn.Query("SELECT user_id FROM user WHERE parent_id = ?", currentUserId)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var childUserId string
		if err := rows.Scan(&childUserId); err != nil {
			return err
		}
		if err := u.queryUserIdsRecursive(conn, childUserId, userIds); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 登录验证
// ----------------------------------------------------------------------------------------------------------
func (u *User) CheckLogin(db *sql.DB) (bool, error) {
	var userID string
	// Prepare the SQL query to prevent SQL injection
	query := "SELECT user_id FROM user WHERE user_name = ? AND user_passwd = ?"
	err := db.QueryRow(query, u.UserName.String, u.UserPasswd.String).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No matching record found, return false indicating invalid credentials
			return false, nil
		}
		// Return any other errors that occurred
		return false, err
	}

	// Set the UserID in the User struct if login is successful
	u.UserID.String = userID

	// Return true indicating a successful login
	return true, nil
}
