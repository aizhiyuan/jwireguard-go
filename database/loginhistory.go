package database

import (
	"database/sql"
	"errors"
	"fmt"
	"jwireguard/global"
	"time"
)

const (
	MAXRECORDSPERUSER = 10 // 每个用户最大记录数
	PAGINNATIONLIMIT  = 50 // 分页查询每页最大记录数
)

type LoginHistory struct {
	Id          sql.NullString `json:"id"`
	UserID      sql.NullString `json:"user_id"`
	LoginTime   sql.NullInt64  `json:"login_time"`
	LoginStatus sql.NullString `json:"login_status"`
}

// CreateLoginHistory 创建登录历史表
func (lh *LoginHistory) CreateLoginHistory(db *sql.DB) {
	if !lh.TableExists(db) {
		createTableSQL := `CREATE TABLE IF NOT EXISTS login_history (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			login_time BIGINT NOT NULL,
			login_status VARCHAR(255) NOT NULL,
			INDEX idx_user_id (user_id),
			INDEX idx_login_time (login_time),
			INDEX idx_user_status (user_id, login_status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

		if _, err := db.Exec(createTableSQL); err != nil {
			global.Log.Errorf("创建login_history表失败: %v", err)
		} else {
			global.Log.Info("成功创建login_history表")
		}
	}
}

// InsertLoginHistory 插入登录记录（自动维护记录数限制）
func (lh *LoginHistory) InsertLoginHistory(db *sql.DB) error {
	if lh.UserID.String == "" || lh.LoginTime.Int64 <= 0 || lh.LoginStatus.String == "" {
		return errors.New("user_id, login_time和login_status不能为空")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	// 确保事务回滚
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// 1. 检查并维护记录数限制
	countQuery := `SELECT COUNT(*) FROM login_history WHERE user_id = ?`
	var count int
	if err := tx.QueryRow(countQuery, lh.UserID.String).Scan(&count); err != nil {
		return fmt.Errorf("查询记录数失败: %w", err)
	}

	if count >= MAXRECORDSPERUSER {
		deleteQuery := `DELETE FROM login_history 
                      WHERE user_id = ? 
                      ORDER BY login_time ASC 
                      LIMIT ?`
		if _, err := tx.Exec(deleteQuery, lh.UserID.String, count-MAXRECORDSPERUSER+1); err != nil {
			return fmt.Errorf("删除旧记录失败: %w", err)
		}
	}

	// 2. 插入新记录
	insertSQL := `INSERT INTO login_history (id, user_id, login_time, login_status) VALUES(?, ?, ?, ?)`
	if _, err := tx.Exec(insertSQL, time.Now().Unix(), lh.UserID.String, lh.LoginTime.Int64, lh.LoginStatus.String); err != nil {
		return fmt.Errorf("插入记录失败: %w", err)
	}

	return tx.Commit()
}

// CheckLockStatus 检查用户锁定状态并自动解锁
func (lh *LoginHistory) CheckLockStatus(db *sql.DB) (bool, error) {
	if lh.UserID.String == "" {
		return false, errors.New("user_id不能为空")
	}

	lockQuery := `SELECT lock_until FROM user WHERE user_id = ?`
	var lockUntil sql.NullInt64
	err := db.QueryRow(lockQuery, lh.UserID.String).Scan(&lockUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.New("用户不存在")
		}
		return false, fmt.Errorf("查询锁定状态失败: %w", err)
	}

	// 1. 用户未锁定
	if lockUntil.Int64 <= 0 {
		return false, nil
	}

	// 2. 锁定已到期，自动解锁
	currentTime := time.Now().Unix()
	if lockUntil.Int64 > 0 && lockUntil.Int64 <= currentTime {
		// 锁定已过期，自动解锁
		if err := lh.UnlockUser(db, true); err != nil {
			return true, fmt.Errorf("自动解锁失败: %w", err) // 返回锁定但解锁失败
		}
		return false, nil
	}

	// 3. 用户仍然在锁定中
	return true, nil
}

// HandleFailedLogin 处理登录失败并检查是否需要锁定用户
func (lh *LoginHistory) HandleFailedLogin(db *sql.DB, defaultLockDuration int32, maxConsecutiveFails int32, lockoutWindow int32) (bool, time.Time, error) {
	if lh.UserID.String == "" {
		return false, time.Time{}, errors.New("user_id不能为空")
	}

	// 1. 检查最近的登录失败记录
	query := `SELECT login_status, login_time 
            FROM login_history 
            WHERE user_id = ? 
            ORDER BY login_time DESC
            LIMIT ?`

	rows, err := db.Query(query, lh.UserID.String, maxConsecutiveFails)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("查询最近登录记录失败: %w", err)
	}
	defer rows.Close()

	var consecutiveFails int32 = 0
	firstFailTime := time.Now().Unix() // 设置为当前时间作为默认值

	for rows.Next() {
		var status string
		var loginTimeStamp int64
		if err := rows.Scan(&status, &loginTimeStamp); err != nil {
			return false, time.Time{}, fmt.Errorf("扫描登录记录失败: %w", err)
		}

		if status == "false" { // 只检查失败记录
			if consecutiveFails == 0 {
				firstFailTime = loginTimeStamp
			}
			consecutiveFails++

			// 达到最大失败次数
			if consecutiveFails >= maxConsecutiveFails {
				// 检查时间窗口
				lastFail := time.Unix(firstFailTime, 0)
				currentTime := time.Now()
				windowEnd := lastFail.Add(time.Duration(lockoutWindow) * time.Second)

				if currentTime.Before(windowEnd) || currentTime.Equal(windowEnd) {
					// 锁定用户
					lockDuration := time.Duration(defaultLockDuration) * time.Second
					unlockTime := time.Now().Add(lockDuration)
					if err := lh.LockUser(db, unlockTime); err != nil {
						return false, unlockTime, fmt.Errorf("锁定用户失败: %w", err)
					}
					return true, unlockTime, nil
				}
			}
		} else {
			// 遇到成功记录，终止检查
			break
		}
	}

	if err := rows.Err(); err != nil {
		return false, time.Time{}, fmt.Errorf("处理结果集出错: %w", err)
	}

	return false, time.Time{}, nil
}

// LockUser 锁定用户
func (lh *LoginHistory) LockUser(db *sql.DB, unlockTime time.Time) error {
	if lh.UserID.String == "" {
		return errors.New("user_id不能为空")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// 1. 更新用户锁定状态
	lockSQL := `UPDATE user SET lock_until = ? WHERE user_id = ?`
	if _, err := tx.Exec(lockSQL, unlockTime.Unix(), lh.UserID.String); err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}

	return tx.Commit()
}

// UnlockUser 解锁用户
func (lh *LoginHistory) UnlockUser(db *sql.DB, clearHistory bool) error {
	if lh.UserID.String == "" {
		return errors.New("user_id不能为空")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	// 1. 更新用户锁定状态
	if _, err := tx.Exec("UPDATE user SET lock_until = NULL WHERE user_id = ?", lh.UserID.String); err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}

	// 2. 清理部分登录历史（如果需要）
	if clearHistory {
		// 保留最后一次成功登录记录
		query := `SELECT id FROM login_history 
               WHERE user_id = ? AND login_status = 'true'
               ORDER BY login_time DESC 
               LIMIT 1`
		var keepID int
		err = tx.QueryRow(query, lh.UserID.String).Scan(&keepID)

		var deleteSQL string
		if err == nil { // 找到成功记录
			deleteSQL = `DELETE FROM login_history 
                      WHERE user_id = ? 
                      AND id <> ?`
			_, err = tx.Exec(deleteSQL, lh.UserID.String, keepID)
		} else if err == sql.ErrNoRows { // 无成功记录
			deleteSQL = `DELETE FROM login_history 
                      WHERE user_id = ? 
                      ORDER BY login_time DESC 
                      LIMIT 1` // 保留最后一条失败记录
			_, err = tx.Exec(deleteSQL, lh.UserID.String)
		}

		if err != nil {
			return fmt.Errorf("清理登录记录失败: %w", err)
		}
	}

	return tx.Commit()
}

// GetLoginHistoriesByUserID 分页获取用户登录历史
func (lh *LoginHistory) GetLoginHistoriesByUserID(db *sql.DB, page int) ([]LoginHistory, int, error) {
	if lh.UserID.String == "" {
		return nil, 0, errors.New("user_id不能为空")
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * PAGINNATIONLIMIT

	// 查询记录
	query := `SELECT user_id, login_time, login_status 
            FROM login_history 
            WHERE user_id = ? 
            ORDER BY login_time DESC 
            LIMIT ? OFFSET ?`

	rows, err := db.Query(query, lh.UserID.String, PAGINNATIONLIMIT, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询登录历史失败: %w", err)
	}
	defer rows.Close()

	var histories []LoginHistory
	for rows.Next() {
		var history LoginHistory
		if err := rows.Scan(&history.UserID, &history.LoginTime, &history.LoginStatus); err != nil {
			return nil, 0, fmt.Errorf("扫描记录失败: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("处理结果集出错: %w", err)
	}

	// 查询总数
	countQuery := `SELECT COUNT(*) FROM login_history WHERE user_id = ?`
	var total int
	if err := db.QueryRow(countQuery, lh.UserID.String).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("查询记录总数失败: %w", err)
	}

	return histories, total, nil
}

// DeleteOldHistories 清理旧的历史记录
func (lh *LoginHistory) DeleteOldHistories(db *sql.DB, days int) error {
	if days <= 0 {
		return errors.New("天数必须大于0")
	}

	threshold := time.Now().AddDate(0, 0, -days).Unix()

	_, err := db.Exec("DELETE FROM login_history WHERE login_time < ?", threshold)
	if err != nil {
		return fmt.Errorf("删除旧记录失败: %w", err)
	}

	return nil
}

// TableExists 检查表是否存在
func (lh *LoginHistory) TableExists(db *sql.DB) bool {
	var tableName string
	query := "SHOW TABLES LIKE 'login_history'"
	if err := db.QueryRow(query).Scan(&tableName); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		global.Log.Errorf("检查表是否存在失败: %v", err)
		return false
	}
	return true
}
