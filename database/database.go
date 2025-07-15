package database

import (
	"database/sql"
	"jwireguard/global"
	"time"

	_ "github.com/go-sql-driver/mysql" // 导入 MySQL 驱动
)

// ----------------------------------------------------------------------------------------------------------
// 初始化SQLite3 数据库
// ----------------------------------------------------------------------------------------------------------
func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("mysql", filepath)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// 定期检查数据库连接的状态
func MonitorDatabase(db *sql.DB) (*sql.DB, error) {
	if err := db.Ping(); err != nil {
		newDB, errOpen := InitDB(global.GlobalJWireGuardini.DataBasePath)
		if errOpen != nil {
			global.Log.Errorln("[MonitorDatabase] 数据库打开失败:", errOpen)
			return db, errOpen
		}
		global.Log.Debugln("[MonitorDatabase] 数据库重新连接.")
		return newDB, nil
	}
	return db, nil
}

// nullStringToString 将 sql.NullString 转换为 string，处理 NULL 值
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt64ToInt64(ns sql.NullInt64) int64 {
	if ns.Valid {
		return ns.Int64
	}
	return 0
}

func nullTimeToTime(ns sql.NullTime) time.Time {
	if ns.Valid {
		return ns.Time
	}
	return time.Time{}
}

func nullInt32ToInt32(ns sql.NullInt32) int32 {
	if ns.Valid {
		return ns.Int32
	}
	return 0
}

func NullBoolToBool(ns sql.NullBool) bool {
	if ns.Valid {
		return ns.Bool
	}
	return false
}

func ConvertCliConfigs(clis []CliConfig) []ExportedCliConfig {
	exported := make([]ExportedCliConfig, len(clis))
	for i, cli := range clis {
		exported[i] = cli.ToExported()
	}
	return exported
}

func ConvertSubnets(subnets []Subnet) []ExportedSubnet {
	exported := make([]ExportedSubnet, len(subnets))
	for i, subnet := range subnets {
		exported[i] = subnet.ToExported()
	}
	return exported
}

func ConvertUsers(users []User) []ExportedUser {
	exported := make([]ExportedUser, len(users))
	for i, user := range users {
		exported[i] = user.ToExported()
	}
	return exported
}
