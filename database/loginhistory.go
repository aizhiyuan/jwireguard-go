package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type LoginHistory struct {
	UserID      sql.NullString `json:"user_id"`
	LoginTime   sql.NullString `json:"login_time"`
	LoginStatus sql.NullString `json:"login_status"`
}

type ExportedLoginHistory struct {
	UserID      string `json:"user_id"`
	LoginTime   string `json:"login_time"`
	LoginStatus string `json:"login_status"`
}

func (u *LoginHistory) ToExported() ExportedLoginHistory {
	return ExportedLoginHistory{
		UserID:      nullStringToString(u.UserID),
		LoginTime:   nullStringToString(u.LoginTime),
		LoginStatus: nullStringToString(u.LoginStatus),
	}
}

// 将 ExportedCliConfig 转换为 CliConfig
func (exported *ExportedLoginHistory) ConvertToUser() LoginHistory {
	return LoginHistory{
		UserID:      sql.NullString{String: exported.UserID, Valid: exported.UserID != ""},
		LoginTime:   sql.NullString{String: exported.LoginTime, Valid: exported.LoginTime != ""},
		LoginStatus: sql.NullString{String: exported.LoginStatus, Valid: exported.LoginStatus != ""},
	}
}

// ----------------------------------------------------------------------------------------------------------
// 创建登录历史记录表
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) CreateLoginHistory(db *sql.DB) {
	if !lh.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS login_history (
		"user_id" TEXT NOT NULL,
		"login_time" DATETIME,
		"login_status" TEXT
	);`
		_, err := db.Exec(createTableSQL)
		if err != nil {
			log.Println("[CreateLoginHistoryTable] Error creating table:", err)
			return
		}
		// log.Println("[CreateLoginHistoryTable] Table 'login_history' created successfully!")
	} else {
		// log.Println("[CreateLoginHistoryTable] Table 'login_history' already exists.")
	}
}

// ----------------------------------------------------------------------------------------------------------
// 插入登录历史记录
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) InsertLoginHistory(db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO login_history (user_id, login_time, login_status) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(lh.UserID.String, lh.LoginTime.String, lh.LoginStatus.String)
	if err != nil {
		return err
	}
	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 根据用户ID查询登录历史记录
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) GetLoginHistoriesByUserID(db *sql.DB) ([]LoginHistory, error) {
	query := "SELECT user_id, login_time, login_status FROM login_history WHERE user_id = ?"
	rows, err := db.Query(query, lh.UserID.String)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []LoginHistory
	for rows.Next() {
		var loginHistory LoginHistory
		if err := rows.Scan(&loginHistory.UserID, &loginHistory.LoginTime, &loginHistory.LoginStatus); err != nil {
			return nil, err
		}
		histories = append(histories, loginHistory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return histories, nil
}

// ----------------------------------------------------------------------------------------------------------
// 查询所有的登录历史记录
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) GetAllLoginHistories(db *sql.DB) ([]LoginHistory, error) {
	query := "SELECT user_id, login_time, login_status FROM login_history"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []LoginHistory
	for rows.Next() {
		var loginHistory LoginHistory
		if err := rows.Scan(&loginHistory.UserID, &loginHistory.LoginTime, &loginHistory.LoginStatus); err != nil {
			return nil, err
		}
		histories = append(histories, loginHistory)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return histories, nil
}

// ----------------------------------------------------------------------------------------------------------
// 更新登录历史记录
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) UpdateLoginHistory(db *sql.DB) error {
	// 确保 UserID 和 LoginTime 不能为空
	if !lh.UserID.Valid || !lh.LoginTime.Valid {
		return errors.New("user_id and login_time cannot be empty")
	}

	setClauses := []string{}
	args := []interface{}{}

	if lh.LoginStatus.Valid {
		setClauses = append(setClauses, "login_status = ?")
		args = append(args, lh.LoginStatus.String)
	}

	if len(setClauses) == 0 {
		return errors.New("no fields to update")
	}

	// 构造安全 SQL 语句
	query := fmt.Sprintf("UPDATE login_history SET %s WHERE user_id = ? AND login_time = ?", strings.Join(setClauses, ", "))
	args = append(args, lh.UserID.String, lh.LoginTime.String)

	// 直接执行 SQL，避免不必要的 Prepare
	_, err := db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 删除登录历史记录
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) DeleteLoginHistory(db *sql.DB) error {
	if lh.UserID.String == "" || lh.LoginTime.String == "" {
		return errors.New("user_id and login_time cannot be empty")
	}

	stmt, err := db.Prepare("DELETE FROM login_history WHERE user_id = ? AND login_time = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(lh.UserID.String, lh.LoginTime.String)
	if err != nil {
		return err
	}
	return nil
}

func (lh *LoginHistory) CanBeLocked(db *sql.DB) (bool, error) {
	if lh.UserID.String == "" {
		return false, errors.New("user_id cannot be empty")
	}

	// 计算 5 分钟前的时间
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")

	// 查询最近 5 分钟内的失败登录记录，按时间倒序排列，最多取 5 条
	query := `SELECT login_time 
	          FROM login_history 
	          WHERE user_id = ? 
	          AND login_status = 'false' 
	          AND login_time >= ? 
	          ORDER BY login_time DESC 
	          LIMIT 5`

	// log.Println("[CanBeLocked] Query:", query, "Params:", lh.UserID.String, fiveMinutesAgo)
	rows, err := db.Query(query, lh.UserID.String, fiveMinutesAgo)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	// 统计查询到的失败次数
	failuresCount := 0
	for rows.Next() {
		var loginTime string
		if err := rows.Scan(&loginTime); err != nil {
			return false, err
		}
		log.Println("[CanBeLocked] login_time:", loginTime)
		failuresCount++
	}

	// 直接返回判断结果
	return failuresCount >= 5, nil
}

// 解除用户登录限制
func (lh *LoginHistory) UnblockUser(db *sql.DB) error {
	if lh.UserID.String == "" {
		return errors.New("user_id cannot be empty")
	}

	_, err := db.Exec("UPDATE user SET user_status = 'true' WHERE user_id = ?", lh.UserID.String)
	if err != nil {
		return fmt.Errorf("failed to unblock user: %v", err)
	}

	_, err = db.Exec("DELETE FROM login_history WHERE user_id = ? AND login_status = 'false'", lh.UserID.String)
	if err != nil {
		return fmt.Errorf("failed to delete failed login attempts: %v", err)
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// 检查表格是否存在
// ----------------------------------------------------------------------------------------------------------
func (lh *LoginHistory) TableExists(db *sql.DB) bool {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name='login_history';"
	var name string
	err := db.QueryRow(query).Scan(&name)
	return err == nil
}
