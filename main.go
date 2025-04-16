package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"jwireguard/database"
	"jwireguard/global"
	"jwireguard/message"
	webservice "jwireguard/webserver"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
)

var sender message.EmailSender

var emailTable string = "[VPN-上海服务器] 设备状态通知"
var htmlTemplate string = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style type="text/css">
	body { margin: 0; padding: 0; font-family: Arial, sans-serif; }
	.email-container {
		width: 100%;
		margin: 0 auto;
		background: #0e9dbb url('https://rescdn.qqmail.com/zh_CN/htmledition/images/xinzhi/bg/a_09.jpg') repeat-x;
	}
	.content {
		padding: 30px 15px;
		color: #ffffff;
		line-height: 1.6;
		font-size: 16px;
	}
</style>
</head>
<body>
<div class="email-container">
	<div class="content">
		<div class="message">
			<b>客户端ID：</b>  {{.CliID}}<br>
			<b>客户端名称:</b> {{.CliName}}<br>
			<b>所在子网： </b> {{.SerName}}<br>
			<b>网络映射： </b> {{.CliMapping}}<br>
			<b>内网地址： </b> {{.CliAddress}}<br>
			<b>在线状态： </b> {{.CliStatus}}<br>
		</div>
	</div>
</div>
</body>
</html>`

type UdpRecvData struct {
	CliID      string `json:"cli_id"`
	CliMapping string `json:"cli_mapping"`
	CliStatus  string `json:"cli_status"`
}

type UdpSendData struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type EditCliStatus struct {
	CliID      string
	CliName    string
	SerName    string
	CliMapping string
	CliAddress string
	CliStatus  string
}

func OpenVPNServer() {
	log.Println("[OpenVPNServer] start")
	// 启动WEB服务
	server_port := fmt.Sprintf(":%d", global.GlobalJWireGuardini.ServerPort)
	server_port_tls := fmt.Sprintf(":%d", global.GlobalJWireGuardini.ServerPortTls)
	log.Println("[OpenVPNServer] OpenVPN API的端口:", server_port)
	log.Println("[OpenVPNServer] OpenVPN API的端口:", server_port_tls)
	webservice.StartServer(server_port, server_port_tls, global.GlobalJWireGuardini.SslCertFile, global.GlobalJWireGuardini.SslKeyFiel)
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

					person.CliStatus.String = "false"
					data := EditCliStatus{
						CliID:      person.CliID.String,
						CliName:    person.CliName.String,
						SerName:    person.SerName.String,
						CliMapping: person.CliMapping.String,
						CliAddress: person.CliAddress.String,
						CliStatus:  person.CliStatus.String,
					}

					// 渲染 HTML
					var tpl bytes.Buffer
					t := template.Must(template.New("html").Parse(htmlTemplate))
					t.Execute(&tpl, data)
					htmlBody := tpl.String()

					log.Printf("[IsDevOnline] 客户端编码:[%s] 客户端名称:[%s] 时间戳:[%d] 客户端在线状态:[%s]",
						person.CliID.String,
						person.CliName.String,
						person.Timestamp.Int64,
						person.CliStatus.String)

					// log.Printf("[IsDevOnline] message:%+v", message)
					err := sender.SendMail(
						[]string{global.GlobalJWireGuardini.To},
						emailTable,
						htmlBody,
						true, // 使用 HTML 格式
					)

					if err != nil {
						log.Printf("[IsDevOnline] 邮件发送失败：%+v", err)
					} else {
						log.Println("[IsDevOnline] 邮件发送成功！")
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

func startUDPListener(port int) {
	// 1. 设置监听地址和端口
	addr := net.UDPAddr{
		Port: port,
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
					data := EditCliStatus{
						CliID:      clientConfig.CliID.String,
						CliName:    clientConfig.CliName.String,
						SerName:    clientConfig.SerName.String,
						CliMapping: clientConfig.CliMapping.String,
						CliAddress: clientConfig.CliAddress.String,
						CliStatus:  clientConfig.CliStatus.String,
					}

					// 渲染 HTML
					var tpl bytes.Buffer
					t := template.Must(template.New("html").Parse(htmlTemplate))
					t.Execute(&tpl, data)
					htmlBody := tpl.String()

					err = sender.SendMail(
						[]string{global.GlobalJWireGuardini.To},
						emailTable,
						htmlBody,
						true, // 使用 HTML 格式
					)

					if err != nil {
						log.Printf("[main] 邮件发送失败：%+v", err)
					} else {
						log.Println("[main] 邮件发送成功！")
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

// 简单的 int 转 string（不引入 strconv）
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

// StartUDPForwarder 启动一个 UDP 转发器，将 fromPort 接收的数据转发到 toHost:toPort
// StartTransparentUDPProxy 启动透明 UDP 转发（支持返回流量）
func StartTransparentUDPProxy(fromPort int, toHost string, toPort int) error {
	// 监听 fromPort
	srcAddr, err := net.ResolveUDPAddr("udp", ":"+itoa(fromPort))
	if err != nil {
		return err
	}
	listener, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return err
	}

	// 目标地址（toHost:toPort）
	dstAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(toHost, itoa(toPort)))
	if err != nil {
		return err
	}

	log.Printf("透明UDP代理已启动: :%d ⇄ %s:%d", fromPort, toHost, toPort)

	// 存储 clientAddr，以便从 dstAddr 收到数据后返回给客户端
	var clientAddr *net.UDPAddr
	// var lastSeen time.Time

	go func() {
		defer listener.Close()
		buffer := make([]byte, 2048)

		for {
			n, addr, err := listener.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("读取UDP失败: %v", err)
				continue
			}

			data := buffer[:n]

			// 判断来源地址
			if addr.String() != dstAddr.String() {
				// 是客户端发来的，转发给目标服务器
				clientAddr = addr
				// lastSeen = time.Now()

				log.Printf("[→] 来自客户端 %s，转发 %d 字节到 %s", clientAddr, n, dstAddr)
				_, err = listener.WriteToUDP(data, dstAddr)
				if err != nil {
					log.Printf("发送到目标失败: %v", err)
				}
			} else if clientAddr != nil {
				// 是目标服务器返回的数据，转发给客户端
				log.Printf("[←] 来自目标 %s，返回 %d 字节到客户端 %s", dstAddr, n, clientAddr)
				_, err = listener.WriteToUDP(data, clientAddr)
				if err != nil {
					log.Printf("返回客户端失败: %v", err)
				}
			} else {
				log.Println("收到目标数据，但无客户端记录，丢弃。")
			}
		}
	}()

	return nil
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
	fmt.Printf("[main] 程序路径 %s \n程序名称 %s\n", execDir, programName)
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

	log.Printf("[main] [SSL PUSH] CERT_FILE %s\n", global.GlobalJWireGuardini.SslCertFile)
	log.Printf("[main] [SSL PUSH] KEY_FILE %s\n", global.GlobalJWireGuardini.SslKeyFiel)

	log.Printf("[main] [EMAIL SETTING] EmailHost %s\n", global.GlobalJWireGuardini.EmailHost)
	log.Printf("[main] [EMAIL SETTING] EmailPort %d\n", global.GlobalJWireGuardini.EmailPort)
	log.Printf("[main] [EMAIL SETTING] EmailUser %s\n", global.GlobalJWireGuardini.EmailUser)
	log.Printf("[main] [EMAIL SETTING] EmailPass %s\n", global.GlobalJWireGuardini.EmailPass)
	log.Printf("[main] [EMAIL SETTING] FormEmail %s\n", global.GlobalJWireGuardini.FormEmail)
	log.Printf("[main] [EMAIL SETTING] FormName %s\n", global.GlobalJWireGuardini.FormName)
	log.Printf("[main] [EMAIL SETTING] To %s\n", global.GlobalJWireGuardini.To)

	sender = message.EmailSender{
		Host:     global.GlobalJWireGuardini.EmailHost,
		Port:     global.GlobalJWireGuardini.EmailPort,
		Username: global.GlobalJWireGuardini.EmailUser,
		Password: global.GlobalJWireGuardini.EmailPass,
		From:     global.GlobalJWireGuardini.FormEmail,
		Name:     global.GlobalJWireGuardini.FormName,
	}

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
	go startUDPListener(int(global.GlobalJWireGuardini.ServerPort))
	go StartTransparentUDPProxy(int(global.GlobalJWireGuardini.ServerPortTls), "127.0.0.1", int(global.GlobalJWireGuardini.ServerPort))

	select {}
}
