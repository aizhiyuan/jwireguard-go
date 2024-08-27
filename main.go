package main

import (
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	webservice "jwireguard/webserver"
)

// var DB *sql.DB
func main() {
	// 加密密钥
	global.GlobalEncryptKey = "@junmix61632320."
	var err error
	// 初始化配置
	jwireguardini_file := "jwireguard.ini"
	global.GlobalJWireGuardini, err = global.LoadOrCreateJWireGuardIni(jwireguardini_file)
	if err != nil {
		fmt.Println("Failed to read jwireguard.ini configuration:", err)
		return
	}

	// 计算默认用户的MD5
	global.GlobalDefaultUserMd5 = global.GenerateSHA256Hash(global.GlobalJWireGuardini.DefaultUser)

	// 初始化 SQLITE数据库
	global.GlobalDB, err = database.InitDB("jwireguard.db")
	if err != nil {
		fmt.Println("Failed to initialize database:", err)
		return
	}
	defer global.GlobalDB.Close()

	// 启动WEB服务
	webservice.StartServer(":8080")

}
