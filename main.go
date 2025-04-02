package main

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	webservice "jwireguard/webserver"
	"jwireguard/wechat"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
)

type UdpRecvData struct {
	CliID      string `json:"cli_id"`
	CliMapping string `json:"cli_mapping"`
	CliStatus  string `json:"cli_status"`
}

type UdpSendData struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func OpenVPNServer() {
	log.Println("[OpenVPNServer] start")
	// 启动WEB服务
	server_port := fmt.Sprintf(":%d", global.GlobalJWireGuardini.ServerPort)
	log.Println("[OpenVPNServer] OpenVPN API的端口:", server_port)
	webservice.StartServer(server_port, global.GlobalJWireGuardini.SslCertFile, global.GlobalJWireGuardini.SslKeyFiel)
}

func IsDevOnline() {
	log.Println("[IsDevOnline] start")
	for {
		time.Sleep(60 * time.Second) // 等待 60 秒
		// 查询连接状态
		database.MonitorDatabase(global.GlobalDB)
		// 创建数据库连接
		clientConfig := database.CliConfig{}
		// 初始化数据库
		clientConfig.CreateCliConfig(global.GlobalDB)
		// 遍历所有的数据
		clientConfigs, err := clientConfig.GetAllCliConfig(global.GlobalDB)
		if err != nil {
			continue
		}

		for _, person := range clientConfigs {
			// log.Println("person:", person)
			// 获取当前时间戳
			currentTime := time.Now().Unix()

			// log.Println("Timestamp:", person.Timestamp.Int64)
			// 时间戳大于60秒
			if (currentTime - person.Timestamp.Int64) >= 60 {

				// 只需要更改在线的设备
				if person.CliStatus.String == "true" {

					// 第一步：获取 Access Token
					if global.GlobalJWireGuardini.CorpID != "" && global.GlobalJWireGuardini.Secret != "" {
						accessToken, err := wechat.GetAccessToken(global.GlobalJWireGuardini.CorpID, global.GlobalJWireGuardini.Secret)
						if err != nil {
							log.Println("[main] 获取Access Token失败:", err)
							continue
						}

						// 第二步：发送消息
						message := wechat.WeChatMessage{
							Touser:  global.GlobalJWireGuardini.Touser, // 接收消息的用户ID，可以是多个用户，逗号分隔，@all为所有成员
							AgentID: global.GlobalJWireGuardini.AgentID,
							MsgType: "text",
							Safe:    0, // 是否保密消息，0为否，1为是
						}
						// 更新设备状态
						person.CliStatus.String = "false"

						message.Text.Content = fmt.Sprintf("客户端ID：%s\n客户端名称：%s\n所在子网：%s\n网络映射：%s\n内网地址：%s\n在线状态：%s",
							person.CliID.String,
							person.CliName.String,
							person.SerName.String,
							person.CliMapping.String,
							person.CliAddress.String,
							person.CliStatus.String)

						log.Printf("[IsDevOnline] 客户端编码:[%s] 客户端名称:[%s] 时间戳:[%d] 客户端在线状态:[%s]",
							person.CliID.String,
							person.CliName.String,
							person.Timestamp.Int64,
							person.CliStatus.String)

						log.Printf("[IsDevOnline] message:%+v", message)
						err = wechat.SendWeChatMessage(accessToken, message)
						if err != nil {
							log.Println("[IsDevOnline] 发送消息失败:", err)
							continue
						}
						log.Println("[IsDevOnline] 发送成功")
					}

					// 将数据更新到数据库中
					err = person.UpdateCliConfig(global.GlobalDB)
					if err != nil {
						log.Printf("[IsDevOnline] 无法将客户端ID: [%s]的状态转为false, err:%v", person.CliID.String, err)
					}

				}
			}
		}

	}
}

func IsIptablesSubnet() {
	log.Println("[IsIptablesSubnet] start")
	// 初始化 SQLITE数据库
	for {
		// 查询连接状态
		database.MonitorDatabase(global.GlobalDB)
		// 创建数据库连接
		subnet := database.Subnet{}
		// 初始化数据库
		subnet.CreateSubnet(global.GlobalDB)
		// 遍历所有的数据
		subnets, err := subnet.GetAllSubnet(global.GlobalDB)
		if err != nil {
			continue
		}

		for _, person := range subnets {
			// 配置Iptables
			rules := fmt.Sprintf("-s %s.%d.0/24 -d %s.0.0/16 -j ACCEPT",
				global.GlobalJWireGuardini.IPPrefix,
				person.SerNum.Int32,
				global.GlobalJWireGuardini.IPPrefix)

			if !global.CheckIptablesRule(rules) {
				err := global.AddIptablesRule(rules)
				if err != nil {
					log.Printf("[IsIptablesSubnet] 路由配置错误 '%s': %v", rules, err)
				} else {
					log.Printf("[IsIptablesSubnet] 路由配置成功 '%s'", rules)
				}
			}
		}
		time.Sleep(60 * time.Second) // 等待 60 秒
	}
}

// var DB *sql.DB
func main() {

	// 获取当前程序的可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v\n", err)
		return
	}

	// 提取程序所在目录
	execDir := filepath.Dir(execPath)

	// 提取程序名称
	programName := filepath.Base(execPath)

	jwireguardini_file := fmt.Sprintf("%s/%s.ini", execDir, programName)

	logName := fmt.Sprintf("/var/log/%s.log", programName)
	// 设置日志文件的轮转
	log.SetOutput(&lumberjack.Logger{
		Filename:   logName, // 日志文件路径
		MaxSize:    100,     // 文件大小限制为100MB
		MaxBackups: 3,       // 保留10个备份文件
		MaxAge:     30,      // 保留日志的最大天数
		Compress:   true,    // 启用压缩
	})

	log.Printf("[main] 程序路径 %s\n", execPath)
	log.Printf("[main] 程序目录 %s\n", execDir)
	log.Printf("[main] 程序名称 %s\n", programName)
	log.Printf("[main] 配置文件 %s 文件\n", jwireguardini_file)
	log.Printf("[main] 设置日志 %s 文件\n", logName)

	// 加密密钥
	global.GlobalEncryptKey = "@junmix61632320."

	// 初始化配置
	log.Printf("[main] 解析 %s 配置文件", jwireguardini_file)
	global.GlobalJWireGuardini, err = global.LoadOrCreateJWireGuardIni(jwireguardini_file)
	if err != nil {
		log.Fatalf("[main] 无法打开 %s 配置文件, err:%v", jwireguardini_file, err)
		return
	}

	// 计算默认用户的MD5
	global.GlobalDefaultUserMd5 = global.GenerateMD5(global.GlobalJWireGuardini.DefaultUser)

	// 初始化 SQLITE数据库
	global.GlobalDB, err = database.InitDB(global.GlobalJWireGuardini.DataBasePath)
	if err != nil {
		log.Fatalf("[main] 无法打开 %s 数据库, err:%v", global.GlobalJWireGuardini.DataBasePath, err)
		return
	}
	log.Printf("[main] 数据库 %s 打开成功!", global.GlobalJWireGuardini.DataBasePath)
	defer global.GlobalDB.Close()

	log.Printf("[main] [GENERAL SETTING] DATA_BASE_PATH %s\n", global.GlobalJWireGuardini.DataBasePath)
	log.Printf("[main] [GENERAL SETTING] IP_PREFIX %s\n", global.GlobalJWireGuardini.IPPrefix)
	log.Printf("[main] [GENERAL SETTING] DEFAULT_USER %s\n", global.GlobalJWireGuardini.DefaultUser)
	log.Printf("[main] [GENERAL SETTING] SUBNET_MAKE %s\n", global.GlobalJWireGuardini.SubnetMask)
	log.Printf("[main] [GENERAL SETTING] SERVER_PORT %d\n", global.GlobalJWireGuardini.ServerPort)
	log.Printf("[main] [GENERAL SETTING] UDP_PORT %d\n", global.GlobalJWireGuardini.UDPPort)

	log.Printf("[main] [SSL PUSH] CERT_FILE %s\n", global.GlobalJWireGuardini.SslCertFile)
	log.Printf("[main] [SSL PUSH] KEY_FILE %s\n", global.GlobalJWireGuardini.SslKeyFiel)

	log.Printf("[main] [MESSAGE PUSH] CORP_ID %s\n", global.GlobalJWireGuardini.CorpID)
	log.Printf("[main] [MESSAGE PUSH] SECRET %s\n", global.GlobalJWireGuardini.Secret)
	log.Printf("[main] [MESSAGE PUSH] AGENT_ID %d\n", global.GlobalJWireGuardini.AgentID)
	log.Printf("[main] [MESSAGE PUSH] TOUSER %s\n", global.GlobalJWireGuardini.Touser)

	// 初始化OpenVPN路径
	global.GlobalOpenVPNPath.BinPath = global.GlobalJWireGuardini.OpenVpnPath + "/bin"
	global.GlobalOpenVPNPath.CcdPath = global.GlobalJWireGuardini.OpenVpnPath + "/ccd"
	global.GlobalOpenVPNPath.ConfigPath = global.GlobalJWireGuardini.OpenVpnPath + "/client"
	global.GlobalOpenVPNPath.ServerPath = global.GlobalJWireGuardini.OpenVpnPath + "/server"
	global.GlobalOpenVPNPath.EasyRsaFile = global.GlobalOpenVPNPath.ServerPath + "/easy-rsa/easyrsa"
	global.GlobalOpenVPNPath.EasyRsaPath = global.GlobalOpenVPNPath.ServerPath + "/easy-rsa"
	global.GlobalOpenVPNPath.PkiPath = global.GlobalOpenVPNPath.ServerPath + "/easy-rsa/pki"
	global.GlobalOpenVPNPath.IssuedPath = global.GlobalOpenVPNPath.PkiPath + "/issued"
	global.GlobalOpenVPNPath.PrivatePath = global.GlobalOpenVPNPath.PkiPath + "/private"
	global.GlobalOpenVPNPath.ReqsPath = global.GlobalOpenVPNPath.PkiPath + "/reqs"

	// fmt.Println("BinPath:", global.GlobalOpenVPNPath.BinPath)
	// fmt.Println("CcdPath:", global.GlobalOpenVPNPath.CcdPath)
	// fmt.Println("ConfigPath:", global.GlobalOpenVPNPath.ConfigPath)
	// fmt.Println("ServerPath:", global.GlobalOpenVPNPath.ServerPath)
	// fmt.Println("EasyRsaFile:", global.GlobalOpenVPNPath.EasyRsaFile)
	// fmt.Println("EasyRsaPath:", global.GlobalOpenVPNPath.EasyRsaPath)
	// fmt.Println("PkiPath:", global.GlobalOpenVPNPath.PkiPath)
	// fmt.Println("IssuedPath:", global.GlobalOpenVPNPath.IssuedPath)
	// fmt.Println("PrivatePath:", global.GlobalOpenVPNPath.PrivatePath)
	// fmt.Println("ReqsPath:", global.GlobalOpenVPNPath.ReqsPath)

	// 启动WEB线程
	go OpenVPNServer()

	// 启动判断设备在线线程
	go IsDevOnline()

	// 启动判断IPTABLES
	go IsIptablesSubnet()

	// fmt.Println("UDPPort:", global.GlobalJWireGuardini.IPPrefix)

	// 1. 设置监听地址和端口
	addr := net.UDPAddr{
		Port: int(global.GlobalJWireGuardini.UDPPort),
		IP:   net.ParseIP("0.0.0.0"),
	}

	// 2. 开启 UDP 监听
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("[main] 无法开启UDP服务器, 监听端口：%d, err:%v", addr.Port, err)
		return
	}
	defer conn.Close()

	log.Println("[main] UDP 服务器已启动，监听端口:", addr.Port)

	// 3. 处理接收到的数据包
	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("[main] 无法接收到UDP数据, err:%v", err)
			continue
		}

		// 4. 打印接收到的原始消息
		message := string(buffer[:n])
		log.Printf("[main] 收到来自 %s 的消息: %s", clientAddr.String(), message)

		// 5. 解析 JSON 数据
		var recvData UdpRecvData
		err = json.Unmarshal(buffer[:n], &recvData)
		if err != nil {
			log.Printf("[main] 无法将接收UDP数据转为JSON数据, err:%v", err)
			continue
		}

		// 6. 打印解析后的数据
		log.Printf("[main] 解析后的数据: %+v", recvData)

		if recvData.CliStatus == "true" {
			var sendData UdpSendData
			// 获取当前时间戳
			currentTime := time.Now().Unix()
			// 查询连接状态
			database.MonitorDatabase(global.GlobalDB)
			// 创建客户端
			clientConfig := database.CliConfig{}
			// 初始化数据库
			clientConfig.CreateCliConfig(global.GlobalDB)

			clientConfig.CliID.String = recvData.CliID
			// 查询客户端信息
			err = clientConfig.GetCliConfigByCliID(global.GlobalDB)
			if err != nil {
				sendData.Status = false
				sendData.Message = "客户端不存在"
				continue
			} else {
				numFunc := clientConfig.EditStatus.Int32 & 0x02
				if numFunc == 0 {
					clientConfig.CliMapping.String = recvData.CliMapping
				}
				clientConfig.Timestamp.Int64 = currentTime
				if clientConfig.CliStatus.String != "true" {
					// 更新在线状态
					clientConfig.CliStatus.String = "true"
					if global.GlobalJWireGuardini.CorpID != "" && global.GlobalJWireGuardini.Secret != "" {
						// 第一步：获取 Access Token
						accessToken, err := wechat.GetAccessToken(global.GlobalJWireGuardini.CorpID, global.GlobalJWireGuardini.Secret)
						if err != nil {
							log.Println("[main] 获取Access Token失败:", err)
							continue
						}

						// 第二步：发送消息
						message := wechat.WeChatMessage{
							Touser:  global.GlobalJWireGuardini.Touser, // 接收消息的用户ID，可以是多个用户，逗号分隔，@all为所有成员
							AgentID: global.GlobalJWireGuardini.AgentID,
							MsgType: "text",
							Safe:    0, // 是否保密消息，0为否，1为是
						}

						message.Text.Content = fmt.Sprintf("客户端ID：%s\n客户端名称：%s\n所在子网：%s\n网络映射：%s\n内网地址：%s\n在线状态：%s",
							clientConfig.CliID.String,
							clientConfig.CliName.String,
							clientConfig.SerName.String,
							clientConfig.CliMapping.String,
							clientConfig.CliAddress.String,
							clientConfig.CliStatus.String)

						log.Printf("[main] message:%+v", message)
						err = wechat.SendWeChatMessage(accessToken, message)
						if err != nil {
							log.Println("[main] 发送消息失败:", err)
							continue
						}
						log.Println("[main] 发送成功")
					}
				}

				err = clientConfig.UpdateCliConfig(global.GlobalDB)
				if err != nil {
					sendData.Status = false
					sendData.Message = "Client status update failed!"
				} else {
					sendData.Status = true
					sendData.Message = "Client status updated successfully!"
				}

			}

			// 7. 回复客户端
			jsonData, err := json.Marshal(sendData)
			if err != nil {
				log.Printf("[main] 无法将DUP发送数据转为JSON, err:%v", err)
				continue
			}

			// 8、返回数据
			log.Printf("[main] 返回数据的数据: %s", string(jsonData))

			// Send data
			_, err = conn.WriteToUDP(jsonData, clientAddr)
			if err != nil {
				log.Printf("[main] 无法将JSON数据使用UDP发送, err:%v", err)
			}

		}
	}
}
