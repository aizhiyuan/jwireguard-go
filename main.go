package main

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	webservice "jwireguard/webserver"
	"log"
	"net"
	"time"
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
	webservice.StartServer(server_port)
}

func IsDevOnline() {
	log.Println("[IsDevOnline] start")
	for {
		time.Sleep(60 * time.Second) // 等待 60 秒

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
					log.Printf("[IsDevOnline] 客户端编码:[%s] 客户端名称:[%s] 时间戳:[%d] 客户端在线状态:[%s]",
						person.CliID.String,
						person.CliName.String,
						person.Timestamp.Int64,
						person.CliStatus.String)

					// 更新设备状态
					person.CliStatus.String = "false"
					// 将数据更新到数据库中
					err := person.UpdateCliConfig(global.GlobalDB)
					if err != nil {
						log.Fatalf("[IsDevOnline] 无法将客户端ID: [%s]的状态转为false, err:%v", person.CliID.String, err)
					}
				}
			}
		}

	}
}

// var DB *sql.DB
func main() {
	// 加密密钥
	global.GlobalEncryptKey = "@junmix61632320."
	var err error
	// 初始化配置
	jwireguardini_file := "jwireguard.ini"
	global.GlobalJWireGuardini, err = global.LoadOrCreateJWireGuardIni(jwireguardini_file)
	if err != nil {
		log.Fatalf("[main] 无法打开 jwireguard.ini 配置文件, err:%v", err)
		return
	}

	// 计算默认用户的MD5
	global.GlobalDefaultUserMd5 = global.GenerateMD5(global.GlobalJWireGuardini.DefaultUser)

	// 初始化 SQLITE数据库
	global.GlobalDB, err = database.InitDB("jwireguard.db")
	if err != nil {
		log.Fatalf("[main] 无法打开 jwireguard.db 数据库, err:%v", err)
		return
	}
	defer global.GlobalDB.Close()

	// fmt.Println("IPPrefix:", global.GlobalJWireGuardini.IPPrefix)
	// fmt.Println("DefaultUser:", global.GlobalJWireGuardini.DefaultUser)
	// fmt.Println("OpenVpnPath:", global.GlobalJWireGuardini.OpenVpnPath)
	// fmt.Println("SubnetMask:", global.GlobalJWireGuardini.SubnetMask)
	// fmt.Println("ServerPort:", global.GlobalJWireGuardini.ServerPort)
	// fmt.Println("UDPPort:", global.GlobalJWireGuardini.UDPPort)

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
			log.Fatalf("[main] 无法接收到UDP数据, err:%v", err)
			continue
		}

		// 4. 打印接收到的原始消息
		message := string(buffer[:n])
		log.Printf("收到来自 %s 的消息: %s", clientAddr.String(), message)

		// 5. 解析 JSON 数据
		var recvData UdpRecvData
		err = json.Unmarshal(buffer[:n], &recvData)
		if err != nil {
			log.Fatalf("[main] 无法将接收UDP数据转为JSON数据, err:%v", err)
			continue
		}

		// 6. 打印解析后的数据
		log.Printf("解析后的数据: %+v", recvData)

		if recvData.CliStatus == "true" {
			var sendData UdpSendData
			// 获取当前时间戳
			currentTime := time.Now().Unix()

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
			} else {
				clientConfig.CliMapping.String = recvData.CliMapping
				clientConfig.Timestamp.Int64 = currentTime
				clientConfig.CliStatus.String = "true"

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
				log.Fatalf("[main] 无法将DUP发送数据转为JSON, err:%v", err)
			}

			// 8、返回数据
			log.Printf("返回数据的数据: %s", string(jsonData))

			// Send data
			_, err = conn.WriteToUDP(jsonData, clientAddr)
			if err != nil {
				log.Fatalf("[main] 无法将JSON数据使用UDP发送, err:%v", err)
			}
		}
	}
}
