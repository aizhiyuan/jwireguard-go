package database

import (
	"database/sql"
	"errors"
	"fmt"
	"jwireguard/global"
	"strings"
)

type User struct {
	UserID         sql.NullString `json:"user_id"`
	SerID          sql.NullString `json:"ser_id"`
	ParentID       sql.NullString `json:"parent_id"`
	UserName       sql.NullString `json:"user_name"`
	UserPasswd     sql.NullString `json:"user_passwd"`
	UserType       sql.NullInt32  `json:"user_type"`
	UserStatus     sql.NullString `json:"user_status"`
	UserEmail      sql.NullString `json:"user_email"`
	UserMac        sql.NullString `json:"user_mac"` // 新增user_mac字段
	SessionID      sql.NullString `json:"session_id"`
	ExpirySeconds  sql.NullInt32  `json:"expiry_seconds"`
	ExpiresAt      sql.NullInt64  `json:"expires_at"`
	LoginErrTime   sql.NullInt32  `json:"login_err_time"`   // 登录错误次数
	LoginErrCount  sql.NullInt32  `json:"login_err_count"`  // 登录错误次数
	LimitLoginTime sql.NullInt32  `json:"limit_login_time"` // 限制登录时间
	LockUntil      sql.NullInt64  `json:"lock_until"`       // 锁定时间
	MailCode       sql.NullString `json:"mail_code"`        // 邮箱验证码
	MailTime       sql.NullInt64  `json:"mail_time"`        // 邮箱验证码发送时间
}

type ExportedUser struct {
	UserID         string `json:"user_id"`
	SerID          string `json:"ser_id"`
	ParentID       string `json:"parent_id"`
	UserName       string `json:"user_name"`
	UserPasswd     string `json:"user_passwd"`
	UserType       int32  `json:"user_type"`
	UserStatus     string `json:"user_status"`
	UserEmail      string `json:"user_email"`
	UserMac        string `json:"user_mac"` // 新增user_mac字段
	SessionID      string `json:"session_id"`
	ExpirySeconds  int32  `json:"expiry_seconds"`
	ExpiresAt      int64  `json:"expires_at"`
	LoginErrTime   int32  `json:"login_err_time"`
	LoginErrCount  int32  `json:"login_err_count"`
	LimitLoginTime int32  `json:"limit_login_time"`
	LockUntil      int64  `json:"lock_until"` // 锁定时间
	MailCode       string `json:"mail_code"`  // 邮箱验证码
	MailTime       int64  `json:"mail_time"`  // 邮箱验证码发送时间
}

// CreateUser creates the user table in MySQL
func (u *User) CreateUser(db *sql.DB) {
	if !u.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS user (
        user_id VARCHAR(255) NOT NULL PRIMARY KEY,
        ser_id VARCHAR(255),
        parent_id VARCHAR(255),
        user_name VARCHAR(255) NOT NULL,
        user_passwd VARCHAR(255) NOT NULL,
        user_type INT NOT NULL,
        user_status VARCHAR(255) NOT NULL,
        user_email VARCHAR(255),
        user_mac VARCHAR(255),
        session_id VARCHAR(255),
        expiry_seconds INT,
        expires_at BIGINT,
        login_err_time INT,
        login_err_count INT,
        limit_login_time INT,
		lock_until BIGINT,    
        mail_code VARCHAR(255), 
        mail_time BIGINT,        
        INDEX idx_user_name (user_name),
        INDEX idx_parent_id (parent_id),
        INDEX idx_session_id (session_id),
        INDEX idx_ser_id (ser_id),
        INDEX idx_user_mac (user_mac)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			global.Log.Errorln("[CreateUser] Error creating table:", err)
			return
		}
	}
}

// ToExported converts User to ExportedUser
func (u *User) ToExported() ExportedUser {
	return ExportedUser{
		UserID:         nullStringToString(u.UserID),
		SerID:          nullStringToString(u.SerID),
		ParentID:       nullStringToString(u.ParentID),
		UserName:       nullStringToString(u.UserName),
		UserPasswd:     nullStringToString(u.UserPasswd),
		UserType:       nullInt32ToInt32(u.UserType),
		UserStatus:     nullStringToString(u.UserStatus),
		UserEmail:      nullStringToString(u.UserEmail),
		UserMac:        nullStringToString(u.UserMac),
		SessionID:      nullStringToString(u.SessionID),
		ExpirySeconds:  nullInt32ToInt32(u.ExpirySeconds),
		ExpiresAt:      nullInt64ToInt64(u.ExpiresAt),
		LoginErrTime:   nullInt32ToInt32(u.LoginErrTime),
		LoginErrCount:  nullInt32ToInt32(u.LoginErrCount),
		LimitLoginTime: nullInt32ToInt32(u.LimitLoginTime),
		LockUntil:      nullInt64ToInt64(u.LockUntil),
		MailCode:       nullStringToString(u.MailCode),
		MailTime:       nullInt64ToInt64(u.MailTime),
	}
}

// ConvertToUser converts ExportedUser to User
func (exported *ExportedUser) ConvertToUser() User {
	return User{
		UserID:         sql.NullString{String: exported.UserID, Valid: exported.UserID != ""},
		SerID:          sql.NullString{String: exported.SerID, Valid: exported.SerID != ""},
		ParentID:       sql.NullString{String: exported.ParentID, Valid: exported.ParentID != ""},
		UserName:       sql.NullString{String: exported.UserName, Valid: exported.UserName != ""},
		UserPasswd:     sql.NullString{String: exported.UserPasswd, Valid: exported.UserPasswd != ""},
		UserType:       sql.NullInt32{Int32: exported.UserType, Valid: exported.UserType != 0},
		UserStatus:     sql.NullString{String: exported.UserStatus, Valid: exported.UserStatus != ""},
		UserEmail:      sql.NullString{String: exported.UserEmail, Valid: exported.UserEmail != ""},
		UserMac:        sql.NullString{String: exported.UserMac, Valid: exported.UserMac != ""},
		SessionID:      sql.NullString{String: exported.SessionID, Valid: exported.SessionID != ""},
		ExpirySeconds:  sql.NullInt32{Int32: exported.ExpirySeconds, Valid: exported.ExpirySeconds != 0},
		ExpiresAt:      sql.NullInt64{Int64: exported.ExpiresAt, Valid: exported.ExpiresAt != 0},
		LoginErrTime:   sql.NullInt32{Int32: exported.LoginErrTime, Valid: exported.LoginErrTime != 0},
		LoginErrCount:  sql.NullInt32{Int32: exported.LoginErrCount, Valid: exported.LoginErrCount != 0},
		LimitLoginTime: sql.NullInt32{Int32: exported.LimitLoginTime, Valid: exported.LimitLoginTime != 0},
		LockUntil:      sql.NullInt64{Int64: exported.LockUntil, Valid: exported.LockUntil != 0},
		MailCode:       sql.NullString{String: exported.MailCode, Valid: exported.MailCode != ""},
		MailTime:       sql.NullInt64{Int64: exported.MailTime, Valid: exported.MailTime != 0},
	}
}

// InsertUser inserts a new user record
func (u *User) InsertUser(db *sql.DB) error {
	query := `INSERT INTO user (
        user_id, ser_id, parent_id, user_name, user_passwd, user_type,
        user_status, user_email, user_mac, session_id, expiry_seconds, expires_at,
        login_err_time, login_err_count, limit_login_time, 
		lock_until, mail_code, mail_time
    ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		u.UserID.String,
		u.SerID.String,
		u.ParentID.String,
		u.UserName.String,
		u.UserPasswd.String,
		u.UserType.Int32,
		u.UserStatus.String,
		u.UserEmail.String,
		u.UserMac.String,
		u.SessionID.String,
		u.ExpirySeconds.Int32,
		u.ExpiresAt.Int64,
		u.LoginErrTime.Int32,
		u.LoginErrCount.Int32,
		u.LimitLoginTime.Int32,
		u.LockUntil.Int64,
		u.MailCode.String,
		u.MailTime.Int64,
	)
	if err != nil {
		return fmt.Errorf("insert user failed: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by user_id
func (u *User) GetUserByID(db *sql.DB) error {
	query := `SELECT 
        user_id, ser_id, parent_id, user_name, user_passwd,
        user_type, user_status, user_email, user_mac,
        session_id, expiry_seconds, expires_at,
        login_err_time, login_err_count, limit_login_time,
		lock_until, mail_code, mail_time
        FROM user WHERE user_id = ?`

	if err := db.QueryRow(query, u.UserID.String).Scan(
		&u.UserID,
		&u.SerID,
		&u.ParentID,
		&u.UserName,
		&u.UserPasswd,
		&u.UserType,
		&u.UserStatus,
		&u.UserEmail,
		&u.UserMac,
		&u.SessionID,
		&u.ExpirySeconds,
		&u.ExpiresAt,
		&u.LoginErrTime,
		&u.LoginErrCount,
		&u.LimitLoginTime,
		&u.LockUntil,
		&u.MailCode,
		&u.MailTime, // 新增锁定时间字段
	); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("get user by ID failed: %w", err)
	}
	return nil
}

// GetUserByName retrieves a user by user_name
func (u *User) GetUserByName(db *sql.DB) error {
	query := `SELECT 
        user_id, ser_id, parent_id, user_name, user_passwd,
        user_type, user_status, user_email, user_mac,
        session_id, expiry_seconds, expires_at,
        login_err_time, login_err_count, limit_login_time,
		lock_until, mail_code, mail_time
        FROM user WHERE user_name = ?`

	if err := db.QueryRow(query, u.UserName.String).Scan(
		&u.UserID,
		&u.SerID,
		&u.ParentID,
		&u.UserName,
		&u.UserPasswd,
		&u.UserType,
		&u.UserStatus,
		&u.UserEmail,
		&u.UserMac,
		&u.SessionID,
		&u.ExpirySeconds,
		&u.ExpiresAt,
		&u.LoginErrTime,
		&u.LoginErrCount,
		&u.LimitLoginTime,
		&u.LockUntil,
		&u.MailCode,
		&u.MailTime, // 新增锁定时间字段
	); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("get user by name failed: %w", err)
	}
	return nil
}

// GetUserBySessionID retrieves a user by session_id
func (u *User) GetUserBySessionID(db *sql.DB) error {
	query := `SELECT 
        user_id, ser_id, parent_id, user_name, user_passwd,
        user_type, user_status, user_email, user_mac,
        session_id, expiry_seconds, expires_at,
        login_err_time, login_err_count, limit_login_time,
		lock_until, mail_code, mail_time
        FROM user WHERE session_id = ?`

	if err := db.QueryRow(query, u.SessionID.String).Scan(
		&u.UserID,
		&u.SerID,
		&u.ParentID,
		&u.UserName,
		&u.UserPasswd,
		&u.UserType,
		&u.UserStatus,
		&u.UserEmail,
		&u.UserMac,
		&u.SessionID,
		&u.ExpirySeconds,
		&u.ExpiresAt,
		&u.LoginErrTime,
		&u.LoginErrCount,
		&u.LimitLoginTime,
		&u.LockUntil,
		&u.MailCode,
		&u.MailTime, // 新增锁定时间字段
	); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("get user by session id failed: %w", err)
	}
	return nil
}

// GetSubnetIdsByUserIds retrieves distinct ser_ids for given user_ids
func (u *User) GetSubnetIdsByUserIds(db *sql.DB, userIds []string) ([]string, error) {
	if len(userIds) == 0 {
		return nil, errors.New("no user_ids provided")
	}

	placeholders := strings.Repeat("?,", len(userIds))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf("SELECT DISTINCT ser_id FROM user WHERE user_id IN (%s)", placeholders)

	args := make([]interface{}, len(userIds))
	for i, id := range userIds {
		args[i] = id
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	subnetIDSet := make(map[string]struct{})
	for rows.Next() {
		var subnetID sql.NullString
		if err := rows.Scan(&subnetID); err != nil {
			return nil, err
		}

		if subnetID.Valid && subnetID.String != "" {
			if _, exists := subnetIDSet[subnetID.String]; !exists {
				subnetIDSet[subnetID.String] = struct{}{}
				result = append(result, subnetID.String)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAllUsers retrieves all users
func (u *User) GetAllUsers(db *sql.DB) ([]User, error) {
	// 更新查询语句包含user_mac字段
	query := `SELECT user_id, ser_id, parent_id, user_name, user_passwd, 
              user_type, user_status, user_email, user_mac, 
              session_id, expiry_seconds, expires_at, login_err_time, login_err_count, limit_login_time,
			  lock_until, mail_code, mail_time
              FROM user`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.UserID,
			&user.SerID,
			&user.ParentID,
			&user.UserName,
			&user.UserPasswd,
			&user.UserType,
			&user.UserStatus,
			&user.UserEmail,
			&user.UserMac, // 扫描新增字段
			&user.SessionID,
			&user.ExpirySeconds,
			&user.ExpiresAt,
			&user.LoginErrTime,
			&user.LoginErrCount,
			&user.LimitLoginTime,
			&user.LockUntil,
			&user.MailCode,
			&user.MailTime, // 扫描新增字段
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateUsers updates user information
func (u *User) UpdateUsers(db *sql.DB) error {
	if u.UserID.String == "" {
		return errors.New("user_id cannot be empty")
	}

	setClauses := []string{}
	args := []interface{}{}

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
	if u.UserType.Int32 != 0 {
		setClauses = append(setClauses, "user_type = ?")
		args = append(args, u.UserType.Int32)
	}
	if u.UserStatus.String != "" {
		setClauses = append(setClauses, "user_status = ?")
		args = append(args, u.UserStatus.String)
	}
	if u.UserEmail.String != "" {
		setClauses = append(setClauses, "user_email = ?")
		args = append(args, u.UserEmail.String)
	}
	if u.UserMac.String != "" {
		setClauses = append(setClauses, "user_mac = ?")
		args = append(args, u.UserMac.String)
	}
	if u.SessionID.String != "" {
		setClauses = append(setClauses, "session_id = ?")
		args = append(args, u.SessionID.String)
	}
	if u.ExpirySeconds.Int32 != 0 {
		setClauses = append(setClauses, "expiry_seconds = ?")
		args = append(args, u.ExpirySeconds.Int32)
	}
	if u.ExpiresAt.Int64 != 0 {
		setClauses = append(setClauses, "expires_at = ?")
		args = append(args, u.ExpiresAt.Int64)
	}
	if u.LoginErrTime.Int32 != 0 {
		setClauses = append(setClauses, "login_err_time = ?")
		args = append(args, u.LoginErrTime.Int32)
	}
	if u.LoginErrCount.Int32 != 0 {
		setClauses = append(setClauses, "login_err_count = ?")
		args = append(args, u.LoginErrCount.Int32)
	}
	if u.LimitLoginTime.Int32 != 0 {
		setClauses = append(setClauses, "limit_login_time = ?")
		args = append(args, u.LimitLoginTime.Int32)
	}
	if u.LockUntil.Int64 != 0 {
		setClauses = append(setClauses, "lock_until = ?")
		args = append(args, u.LockUntil.Int64)
	}
	if u.MailCode.String != "" {
		setClauses = append(setClauses, "mail_code = ?")
		args = append(args, u.MailCode.String)
	}
	if u.MailTime.Int64 != 0 {
		setClauses = append(setClauses, "mail_time = ?")
		args = append(args, u.MailTime.Int64)
	}

	// 如果没有任何字段需要更新，返回错误
	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	query := fmt.Sprintf("UPDATE user SET %s WHERE user_id = ?", strings.Join(setClauses, ", "))
	args = append(args, u.UserID.String)

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

// DeleteUsers deletes a user
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

// TableExists checks if the user table exists in MySQL
func (u *User) TableExists(db *sql.DB) bool {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'user'"
	var name string
	err := db.QueryRow(query).Scan(&name)
	return err == nil
}

// ColumnExists checks if a column exists in the user table
func (u *User) ColumnExists(db *sql.DB, columnName string) bool {
	query := "SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'user' AND column_name = ?"
	var name string
	err := db.QueryRow(query, columnName).Scan(&name)
	return err == nil
}

// QueryUserIds recursively gets all user IDs in a hierarchy
func (u *User) QueryUserIds(db *sql.DB, targetUserId string) ([]string, error) {
	var userIds []string
	err := u.queryUserIdsRecursive(db, targetUserId, &userIds)
	if err != nil {
		return nil, err
	}
	return userIds, nil
}

func (u *User) queryUserIdsRecursive(db *sql.DB, currentUserId string, userIds *[]string) error {
	*userIds = append(*userIds, currentUserId)

	rows, err := db.Query("SELECT user_id FROM user WHERE parent_id = ?", currentUserId)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var childUserId string
		if err := rows.Scan(&childUserId); err != nil {
			return err
		}
		if err := u.queryUserIdsRecursive(db, childUserId, userIds); err != nil {
			return err
		}
	}

	return rows.Err()
}

// CheckLogin verifies user credentials
func (u *User) CheckLogin(db *sql.DB) (bool, error) {
	var userID string
	query := "SELECT user_id FROM user WHERE user_name = ? AND user_passwd = ?"
	err := db.QueryRow(query, u.UserName.String, u.UserPasswd.String).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	u.UserID.String = userID
	return true, nil
}
