package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"jwireguard/database"
	"jwireguard/global"
	"jwireguard/message"
	webservice "jwireguard/webserver"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
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
	CliMac     string `json:"cli_mac"`
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

func initLogger(logName string, logLevel logrus.Level) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	logger.SetLevel(logLevel)

	// 同时输出到文件和控制台
	fileOutput := &lumberjack.Logger{
		Filename:   logName,
		MaxSize:    2, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
		Compress:   true,
	}

	// 多输出源
	logger.SetOutput(io.MultiWriter(os.Stdout, fileOutput))

	// 可选：设置JSON格式
	logger.SetFormatter(&logrus.JSONFormatter{})

	return logger
}

func OpenVPNServer() {
	global.Log.Infof("[OpenVPNServer] start")
	// 启动WEB服务
	server_port := fmt.Sprintf(":%d", global.GlobalJWireGuardini.ServerPort)
	server_port_tls := fmt.Sprintf(":%d", global.GlobalJWireGuardini.ServerPortTls)
	global.Log.Infoln("[OpenVPNServer] OpenVPN API的端口:", server_port)
	global.Log.Infoln("[OpenVPNServer] OpenVPN API的端口:", server_port_tls)
	webservice.StartServer(server_port, server_port_tls, global.GlobalJWireGuardini.SslCertFile, global.GlobalJWireGuardini.SslKeyFiel)
}

func IsDevOnline() {
	global.Log.Infof("[IsDevOnline] start")
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

					global.Log.Debugf("[IsDevOnline] 客户端编码:[%s] 客户端名称:[%s] 时间戳:[%d] 客户端在线状态:[%s]",
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
						global.Log.Errorf("[IsDevOnline] 邮件发送失败：%+v", err)
					} else {
						global.Log.Debugln("[IsDevOnline] 邮件发送成功！")
					}

					// 将数据更新到数据库中
					err = person.UpdateCliConfig(global.GlobalDB)
					if err != nil {
						global.Log.Errorf("[IsDevOnline] 无法将客户端ID: [%s]的状态转为false, err:%v", person.CliID.String, err)
					}

				}
			}
		}

	}
}

func IsIptablesSubnet() {
	global.Log.Infof("[IsIptablesSubnet] start")
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
					global.Log.Errorf("[IsIptablesSubnet] 路由配置错误 '%s': %v", rules, err)
				} else {
					global.Log.Debugf("[IsIptablesSubnet] 路由配置成功 '%s'", rules)
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

	global.Log.Infoln("[main] UDP 服务器已启动，监听端口:", addr.Port)

	// 3. 处理接收到的数据包
	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			global.Log.Errorf("[main] 无法接收到设备心跳信息, err:%v", err)
			continue
		}

		// 4. 打印接收到的原始消息
		// message := string(buffer[:n])
		// log.Printf("[main] 收到来自 %s cli_id: %s", clientAddr.String(), message)

		// 5. 解析 JSON 数据
		var recvData UdpRecvData
		err = json.Unmarshal(buffer[:n], &recvData)
		if err != nil {
			global.Log.Errorf("[main] 无法将接收UDP数据转为JSON数据, err:%v", err)
			continue
		}

		// 6. 打印解析后的数据
		// log.Printf("[main] 解析后的数据: %+v", recvData)
		global.Log.Debugf("[main] 收到来自 %s cli_id %s 的心跳数据", clientAddr.String(), recvData.CliID)

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
			// 设置客户端ID和MAC地址
			clientConfig.CliID.String = recvData.CliID
			clientConfig.CliMac.String = recvData.CliMac
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
						global.Log.Errorf("[main] 邮件发送 cli_id %s 设备在线状态失败, err:%v", clientConfig.CliID.String, err)
					} else {
						global.Log.Debugf("[main] 邮件发送 cli_id %s 设备在线状态成功!", clientConfig.CliID.String)
					}
				}

				// 判断MAC地址是否一样
				// if recvData.CliMac != "" {
				// 	if clientConfig.CliMac.String != recvData.CliMac {
				// 		clientConfig.OnlineStatus.String = "false"
				// 	} else {
				// 		clientConfig.OnlineStatus.String = "true"
				// 	}
				// }

				err = clientConfig.UpdateCliConfig(global.GlobalDB)
				if err != nil {
					sendData.Status = false
					sendData.Message = "Client status update failed!"
					global.Log.Errorf("[main] cli_id %s 心跳更新失败", clientConfig.CliID.String)
				} else {
					sendData.Status = true
					sendData.Message = "Client status updated successfully!"
					global.Log.Debugf("[main] cli_id %s 心跳更新成功", clientConfig.CliID.String)
				}
			}

			// 7. 回复客户端
			jsonData, err := json.Marshal(sendData)
			if err != nil {
				global.Log.Errorf("[main] 无法将 cli_id %s 的设备转换回复信息, err:%v", clientConfig.CliID.String, err)
				continue
			}

			// 8、返回数据
			// log.Printf("[main] 返回数据的数据: %s", string(jsonData))

			// Send data
			_, err = conn.WriteToUDP(jsonData, clientAddr)
			if err != nil {
				global.Log.Errorf("[main] 无法向 cli_id %s 的设备回复心跳信息, err:%v", clientConfig.CliID.String, err)
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

	global.Log.Infof("透明UDP代理已启动: :%d ⇄ %s:%d", fromPort, toHost, toPort)

	// 存储 clientAddr，以便从 dstAddr 收到数据后返回给客户端
	var clientAddr *net.UDPAddr
	// var lastSeen time.Time

	go func() {
		defer listener.Close()
		buffer := make([]byte, 2048)

		for {
			n, addr, err := listener.ReadFromUDP(buffer)
			if err != nil {
				global.Log.Errorf("读取UDP失败: %v", err)
				continue
			}

			data := buffer[:n]

			// 判断来源地址
			if addr.String() != dstAddr.String() {
				// 是客户端发来的，转发给目标服务器
				clientAddr = addr
				// lastSeen = time.Now()

				global.Log.Debugf("[→] 来自客户端 %s，转发 %d 字节到 %s", clientAddr, n, dstAddr)
				_, err = listener.WriteToUDP(data, dstAddr)
				if err != nil {
					global.Log.Errorf("发送到目标失败: %v", err)
				}
			} else if clientAddr != nil {
				// 是目标服务器返回的数据，转发给客户端
				global.Log.Debugf("[←] 来自目标 %s, 返回 %d 字节到客户端 %s", dstAddr, n, clientAddr)
				_, err = listener.WriteToUDP(data, clientAddr)
				if err != nil {
					global.Log.Errorf("返回客户端失败: %v", err)
				}
			} else {
				global.Log.Errorf("收到目标数据，但无客户端记录，丢弃。")
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
		fmt.Printf("Error getting executable path: %v\n", err)
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
	// log.SetOutput(&lumberjack.Logger{
	// 	Filename:   logName, // 日志文件路径
	// 	MaxSize:    100,     // 文件大小限制为100MB
	// 	MaxBackups: 3,       // 保留10个备份文件
	// 	MaxAge:     30,      // 保留日志的最大天数
	// 	Compress:   true,    // 启用压缩
	// })
	global.Log = initLogger(logName, logrus.DebugLevel)

	global.Log.Infof("[main] 程序路径 %s\n", execPath)
	global.Log.Infof("[main] 程序目录 %s\n", execDir)
	global.Log.Infof("[main] 程序名称 %s\n", programName)
	global.Log.Infof("[main] 配置文件 %s 文件\n", jwireguardini_file)
	global.Log.Infof("[main] 设置日志 %s 文件\n", logName)

	// 加密密钥
	global.GlobalEncryptKey = "@junmix61632320."

	// 初始化配置
	global.Log.Infof("[main] 解析 %s 配置文件", jwireguardini_file)
	global.GlobalJWireGuardini, err = global.LoadOrCreateJWireGuardIni(jwireguardini_file)
	if err != nil {
		global.Log.Fatalf("[main] 无法打开 %s 配置文件, err:%v", jwireguardini_file, err)
		return
	}

	// 计算默认用户的MD5
	global.GlobalDefaultUserMd5 = global.GenerateMD5(global.GlobalJWireGuardini.DefaultUser)

	// 初始化 SQLITE数据库
	global.GlobalDB, err = database.InitDB(global.GlobalJWireGuardini.DataBasePath)
	if err != nil {
		global.Log.Fatalf("[main] 无法打开 %s 数据库, err:%v", global.GlobalJWireGuardini.DataBasePath, err)
		return
	}
	global.Log.Infof("[main] 数据库 %s 打开成功!", global.GlobalJWireGuardini.DataBasePath)
	defer global.GlobalDB.Close()

	global.Log.Infof("[main] [GENERAL SETTING] DATA_BASE_PATH %s\n", global.GlobalJWireGuardini.DataBasePath)
	global.Log.Infof("[main] [GENERAL SETTING] IP_PREFIX %s\n", global.GlobalJWireGuardini.IPPrefix)
	global.Log.Infof("[main] [GENERAL SETTING] DEFAULT_USER %s\n", global.GlobalJWireGuardini.DefaultUser)
	global.Log.Infof("[main] [GENERAL SETTING] SUBNET_MAKE %s\n", global.GlobalJWireGuardini.SubnetMask)
	global.Log.Infof("[main] [GENERAL SETTING] SERVER_PORT %d\n", global.GlobalJWireGuardini.ServerPort)

	global.Log.Infof("[main] [SSL PUSH] CERT_FILE %s\n", global.GlobalJWireGuardini.SslCertFile)
	global.Log.Infof("[main] [SSL PUSH] KEY_FILE %s\n", global.GlobalJWireGuardini.SslKeyFiel)

	global.Log.Infof("[main] [EMAIL SETTING] EmailHost %s\n", global.GlobalJWireGuardini.EmailHost)
	global.Log.Infof("[main] [EMAIL SETTING] EmailPort %d\n", global.GlobalJWireGuardini.EmailPort)
	global.Log.Infof("[main] [EMAIL SETTING] EmailUser %s\n", global.GlobalJWireGuardini.EmailUser)
	global.Log.Infof("[main] [EMAIL SETTING] EmailPass %s\n", global.GlobalJWireGuardini.EmailPass)
	global.Log.Infof("[main] [EMAIL SETTING] FormEmail %s\n", global.GlobalJWireGuardini.FormEmail)
	global.Log.Infof("[main] [EMAIL SETTING] FormName %s\n", global.GlobalJWireGuardini.FormName)
	global.Log.Infof("[main] [EMAIL SETTING] To %s\n", global.GlobalJWireGuardini.To)

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
