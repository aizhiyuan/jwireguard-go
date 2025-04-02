// myapp/global/global.go
package global

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
	"gopkg.in/ini.v1"
)

// JWireGuardIni 结构体用于封装 INI 文件的数据
type JWireGuardIni struct {
	DataBasePath string
	IPPrefix     string // IP前缀
	DefaultUser  string // 默认用户
	OpenVpnPath  string // OpenVpn 路径
	OpenSslFile  string // OpenSSL 程序路径
	SubnetMask   string //子网掩码
	NetworkMask  string //网络子网掩码
	ServerPort   uint16 //服务器端口
	UDPPort      uint16 //UDP服务器端口
	SslCertFile  string //SSL CertFile 文件
	SslKeyFiel   string //SSL KeyFile 文件
	CorpID       string //企业微信的CorpID
	Secret       string //企业微信的Secret
	AgentID      int    //企业微信的AgentID
	Touser       string //企业微信的Touser
}

type OpenVPNPath struct {
	CcdPath     string
	BinPath     string
	ConfigPath  string
	ServerPath  string
	EasyRsaPath string
	EasyRsaFile string
	PkiPath     string
	IssuedPath  string
	PrivatePath string
	ReqsPath    string
}

// ----------------------------------------------------------------------------------------------------------
// 全局变量
// ----------------------------------------------------------------------------------------------------------
var GlobalDB *sql.DB // This is a global variable
var GlobalDefaultUserMd5 string
var GlobalJWireGuardini *JWireGuardIni
var GlobalEncryptKey string
var GlobalJWireGuardDBFile string
var GlobalOpenVPNPath OpenVPNPath

// 定义多个时间服务器
var timeServers = []string{
	"http://worldtimeapi.org/api/timezone/Etc/UTC",
	"http://www.timeapi.io/api/Time/current/zone?timeZone=UTC",
	"http://time.jsontest.com/",
	"http://time.nist.gov/",
	"http://api.timezonedb.com/v2.1/current?key=YOUR_API_KEY&format=json&by=zone&zone=UTC",
	"http://www.google.com/ig/api",
	"http://worldtimeapi.org/api/timezone/Etc/UTC",
	"http://current-time-api.herokuapp.com/api/time",
	"http://timeapi.org/utc/now",
	"http://pool.ntp.org/",
}

// ----------------------------------------------------------------------------------------------------------
// CheckFileExists 检查文件是否存在
// ----------------------------------------------------------------------------------------------------------
func CheckFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// ----------------------------------------------------------------------------------------------------------
// DeleteFileIfExists 检查文件是否存在，如果存在则删除文件
// ----------------------------------------------------------------------------------------------------------
func DeleteFileIfExists(filePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); err == nil {
		// 文件存在，执行删除操作
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to delete file: %v", err)
		}
	} else if os.IsNotExist(err) {
		// 文件不存在
		return nil
	} else {
		// 其他错误
		return fmt.Errorf("error checking file existence: %v", err)
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
// uniqueStrings 去除重复值
// ----------------------------------------------------------------------------------------------------------
func UniqueStrings(input []string) []string {
	// 使用 map[string]struct{} 来跟踪唯一的字符串
	uniqueMap := make(map[string]struct{})
	var result []string

	for _, str := range input {
		// 如果字符串已经存在于 map 中，则跳过
		if _, exists := uniqueMap[str]; exists {
			continue
		}
		// 否则，将字符串添加到 map 和结果切片中
		uniqueMap[str] = struct{}{}
		result = append(result, str)
	}

	return result
}

// ----------------------------------------------------------------------------------------------------------
// GenerateMD5 生成字符串的 MD5 哈希值
// ----------------------------------------------------------------------------------------------------------
func GenerateMD5(input string) string {
	hasher := md5.New()
	hasher.Write([]byte(input))
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

// ----------------------------------------------------------------------------------------------------------
// GenerateSHA256Hash 生成SHA-256哈希值
// ----------------------------------------------------------------------------------------------------------
func GenerateSHA256Hash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

// ----------------------------------------------------------------------------------------------------------
// GenerateSHA3Hash 生成SHA-3-256哈希值
// ----------------------------------------------------------------------------------------------------------
func GenerateSHA3Hash(data string) string {
	h := sha3.New256()
	h.Write([]byte(data))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

// ----------------------------------------------------------------------------------------------------------
// 读取或创建 INI 文件并加载配置
// ----------------------------------------------------------------------------------------------------------
func LoadOrCreateJWireGuardIni(filePath string) (*JWireGuardIni, error) {
	var cfg *ini.File
	var err error

	// 检查文件是否存在
	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，创建一个新的配置文件
		cfg = ini.Empty()

		// 创建默认设置
		cfg.Section("GENERAL SETTING").Key("DATA_BASE_PATH").SetValue("jwireguard.db")
		cfg.Section("GENERAL SETTING").Key("IP_PREFIX").SetValue("10.100")
		cfg.Section("GENERAL SETTING").Key("DEFAULT_USER").SetValue("admin")
		cfg.Section("GENERAL SETTING").Key("OPENVPN_PATH").SetValue("/etc/openvpn")
		cfg.Section("GENERAL SETTING").Key("OPENSSL_FILE").SetValue("/usr/bin/openssl")
		cfg.Section("GENERAL SETTING").Key("SUBNET_MAKE").SetValue("255.255.0.0")
		cfg.Section("GENERAL SETTING").Key("NETWORK_MASK").SetValue("255.255.255.0")
		cfg.Section("GENERAL SETTING").Key("SERVER_PORT").SetValue("1092")
		cfg.Section("GENERAL SETTING").Key("UDP_PORT").SetValue("1092")
		cfg.Section("SSL SETTING").Key("CERT_FILE").SetValue("")
		cfg.Section("SSL SETTING").Key("KEY_FILE").SetValue("")
		cfg.Section("MESSAGE PUSH").Key("CORP_ID").SetValue("")
		cfg.Section("MESSAGE PUSH").Key("SECRET").SetValue("")
		cfg.Section("MESSAGE PUSH").Key("AGENT_ID").SetValue("1000002")
		cfg.Section("MESSAGE PUSH").Key("TOUSER").SetValue("@all")

		// 保存到文件
		if err = cfg.SaveTo(filePath); err != nil {
			return nil, fmt.Errorf("failed to create new ini file: %v", err)
		}
	} else {
		// 文件存在，读取配置
		cfg, err = ini.Load(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load ini file: %v", err)
		}
	}

	// 加载配置到结构体
	jwg := &JWireGuardIni{
		DataBasePath: cfg.Section("GENERAL SETTING").Key("DATA_BASE_PATH").String(),
		IPPrefix:     cfg.Section("GENERAL SETTING").Key("IP_PREFIX").String(),
		DefaultUser:  cfg.Section("GENERAL SETTING").Key("DEFAULT_USER").String(),
		OpenVpnPath:  cfg.Section("GENERAL SETTING").Key("OPENVPN_PATH").String(),
		OpenSslFile:  cfg.Section("GENERAL SETTING").Key("OPENSSL_FILE").String(),
		SubnetMask:   cfg.Section("GENERAL SETTING").Key("SUBNET_MAKE").String(),
		NetworkMask:  cfg.Section("GENERAL SETTING").Key("NETWORK_MASK").String(),
		ServerPort:   uint16(cfg.Section("GENERAL SETTING").Key("SERVER_PORT").MustUint(1092)),
		UDPPort:      uint16(cfg.Section("GENERAL SETTING").Key("UDP_PORT").MustUint(1092)),
		SslCertFile:  cfg.Section("SSL SETTING").Key("CERT_FILE").String(),
		SslKeyFiel:   cfg.Section("SSL SETTING").Key("KEY_FILE").String(),
		CorpID:       cfg.Section("MESSAGE PUSH").Key("CORP_ID").String(),
		Secret:       cfg.Section("MESSAGE PUSH").Key("SECRET").String(),
		AgentID:      cfg.Section("MESSAGE PUSH").Key("AGENT_ID").MustInt(1000002),
		Touser:       cfg.Section("MESSAGE PUSH").Key("TOUSER").String(),
	}

	return jwg, nil
}

// ----------------------------------------------------------------------------------------------------------
// WriteToFile 将内容写入指定的文件
// ----------------------------------------------------------------------------------------------------------
func WriteToFile(filename string, content string) error {
	return ioutil.WriteFile(filename, []byte(content), 0644)
}

// ----------------------------------------------------------------------------------------------------------
// Encrypt 使用AES加密算法对给定的文本进行加密
// ----------------------------------------------------------------------------------------------------------
func Encrypt(plainText string, key string) (string, error) {
	// Ensure key length is 16 bytes (128 bits)
	if len(key) != 16 {
		return "", fmt.Errorf("key length must be 16 bytes")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	plainTextBytes := []byte(plainText)
	ciphertext := make([]byte, aes.BlockSize+len(plainTextBytes))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plainTextBytes)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// ----------------------------------------------------------------------------------------------------------
// Decrypt 使用AES解密算法对给定的加密文本进行解密
// ----------------------------------------------------------------------------------------------------------
func Decrypt(cipherText string, key string) (string, error) {
	if len(key) != 16 {
		return "", fmt.Errorf("key length must be 16 bytes")
	}

	ciphertextBytes, err := base64.URLEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	iv := ciphertextBytes[:aes.BlockSize]
	ciphertextBytes = ciphertextBytes[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertextBytes, ciphertextBytes)

	return string(ciphertextBytes), nil
}

// ----------------------------------------------------------------------------------------------------------
// ConvertToNetworkAddress IP地址和子网掩码生成网络地址
// ----------------------------------------------------------------------------------------------------------
func ConvertToNetworkAddress(ipStr, maskStr string) (string, error) {
	// Parse the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Parse the subnet mask
	mask := net.ParseIP(maskStr)
	if mask == nil {
		return "", fmt.Errorf("invalid subnet mask: %s", maskStr)
	}

	// Convert IP and mask to 4-byte representations
	ip = ip.To4()
	mask = mask.To4()

	if ip == nil || mask == nil {
		return "", fmt.Errorf("invalid IP or subnet mask")
	}

	// Apply the subnet mask to the IP address
	networkIP := make(net.IP, len(ip))
	for i := 0; i < len(ip); i++ {
		networkIP[i] = ip[i] & mask[i]
	}

	return networkIP.String(), nil
}

// ----------------------------------------------------------------------------------------------------------
// FetchTimeFromURL 从指定 URL 获取时间
// ----------------------------------------------------------------------------------------------------------
func FetchTimeFromURL(url string) (time.Time, error) {
	resp, err := http.Get(url)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return time.Time{}, err
	}

	var networkTime time.Time
	switch url {
	case "http://worldtimeapi.org/api/timezone/Etc/UTC":
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return time.Time{}, err
		}
		timeStr, ok := result["utc_datetime"].(string)
		if !ok {
			return time.Time{}, fmt.Errorf("unexpected response format")
		}
		networkTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return time.Time{}, err
		}

	case "http://www.timeapi.io/api/Time/current/zone?timeZone=UTC":
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return time.Time{}, err
		}
		timeStr, ok := result["dateTime"].(string)
		if !ok {
			return time.Time{}, fmt.Errorf("unexpected response format")
		}
		networkTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return time.Time{}, err
		}

	case "http://time.jsontest.com/":
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return time.Time{}, err
		}
		timeStr, ok := result["date"].(string)
		if !ok {
			return time.Time{}, fmt.Errorf("unexpected response format")
		}
		networkTime, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
		if err != nil {
			return time.Time{}, err
		}

	case "http://current-time-api.herokuapp.com/api/time":
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return time.Time{}, err
		}
		timeStr, ok := result["currentDateTime"].(string)
		if !ok {
			return time.Time{}, fmt.Errorf("unexpected response format")
		}
		networkTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return time.Time{}, err
		}

	default:
		dateHeader := resp.Header.Get("Date")
		if dateHeader == "" {
			return time.Time{}, fmt.Errorf("no Date header in response")
		}
		networkTime, err = time.Parse(time.RFC1123, dateHeader)
		if err != nil {
			return time.Time{}, err
		}
	}

	return networkTime, nil
}

// 尝试从多个时间服务器获取时间
func GetNetworkTime() (time.Time, error) {
	for _, server := range timeServers {
		networkTime, err := FetchTimeFromURL(server)
		if err == nil {
			return networkTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to fetch time from all servers")
}

// ----------------------------------------------------------------------------------------------------------
// ShellAddClient 添加客户端证书
// ----------------------------------------------------------------------------------------------------------
func ShellAddClient(cliId string, cliAddr string) error {
	headClient := fmt.Sprintf("%s/openvpn.txt", GlobalOpenVPNPath.ConfigPath)
	caClient := fmt.Sprintf("%s/ca.crt", GlobalOpenVPNPath.PkiPath)
	taClient := fmt.Sprintf("%s/ta.key", GlobalOpenVPNPath.PkiPath)
	ovpnClient := fmt.Sprintf("%s/%s.ovpn", GlobalOpenVPNPath.ConfigPath, cliId)
	ccdClient := fmt.Sprintf("%s/%s", GlobalOpenVPNPath.CcdPath, cliId)
	privateClient := fmt.Sprintf("%s/%s.key", GlobalOpenVPNPath.PrivatePath, cliId)
	reqsClient := fmt.Sprintf("%s/%s.req", GlobalOpenVPNPath.ReqsPath, cliId)
	issuedClient := fmt.Sprintf("%s/%s.crt", GlobalOpenVPNPath.IssuedPath, cliId)

	// Define file paths
	files := map[string]string{
		"key":      privateClient,
		"cert":     issuedClient,
		"ca":       caClient,
		"tls-auth": taClient,
	}

	// log.Println("headClient: ", headClient)
	// log.Println("caClient: ", caClient)
	// log.Println("taClient: ", taClient)
	// log.Println("ccdClient: ", ccdClient)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 删除配置文件
	err := DeleteFileIfExists(ovpnClient)
	if err != nil {
		return fmt.Errorf("无法删除原配置文件")
	}

	// 删除CCD文件
	err = DeleteFileIfExists(ccdClient)
	if err != nil {
		return fmt.Errorf("无法删除原CCD文件")
	}

	// 删除REQS文件
	err = DeleteFileIfExists(reqsClient)
	if err != nil {
		return fmt.Errorf("无法删除原REQS文件")
	}

	// 删除PRIVATE文件
	err = DeleteFileIfExists(privateClient)
	if err != nil {
		return fmt.Errorf("无法删除原PRIVATE文件")
	}

	// 删除ISSUE文件
	err = DeleteFileIfExists(issuedClient)
	if err != nil {
		return fmt.Errorf("无法删除原ISSUE文件")
	}

	// 调用封装的函数
	if err := ChangeDir(GlobalOpenVPNPath.EasyRsaPath); err != nil {
		fmt.Println(err)
	}

	// 创建证书请求 (生成客户端私钥和CSR)
	cmd := exec.Command(GlobalOpenVPNPath.EasyRsaFile, "build-client-full", cliId, "nopass")
	cmd.Env = append(os.Environ(), "EASYRSA_BATCH=1") // 跳过确认步骤

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("无法创建私钥: %v Output: %s", err, string(output))
	}

	// 签发证书
	cmd = exec.Command(GlobalOpenVPNPath.EasyRsaFile, "sign-req", "client", cliId)
	cmd.Env = append(os.Environ(), "EASYRSA_BATCH=1") // 跳过确认步骤

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("无法签发证书: %v Output: %s", err, string(output))
	}

	// // 生成网络地址：
	// clientNetworkAddr, err := ConvertToNetworkAddress(cliAddr, GlobalJWireGuardini.NetworkMask)
	// if err != nil {
	// 	return fmt.Errorf("无法生成网络地址: %s 目标地址 %s", err, clientNetworkAddr)
	// }

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
		cliAddr,
		GlobalJWireGuardini.SubnetMask,
		GlobalJWireGuardini.IPPrefix,
		GlobalJWireGuardini.NetworkMask,
		cliAddr)

	err = WriteToFile(ccdClient, changClientAddr)
	if err != nil {
		return fmt.Errorf("无法对ccd文件写入: %s file:%s ", err, ccdClient)
	}

	for _, file := range files {
		if !CheckFileExists(file) {
			return fmt.Errorf("该%s文件不存在, err:%v", file, err)
		}
	}

	// Create the .ovpn file
	err = CreateOVPNFile(headClient, ovpnClient, files)
	if err != nil {
		fmt.Println("Error creating .ovpn file:", err)
		return fmt.Errorf("无法合成%s.ovpn, err:%v", cliId, err)
	}

	// log.Printf("Output:\n%s", output)
	if (!CheckFileExists(privateClient)) ||
		(!CheckFileExists(reqsClient)) ||
		(!CheckFileExists(ccdClient)) ||
		(!CheckFileExists(ovpnClient)) ||
		(!CheckFileExists(issuedClient)) {

		// 删除错误客户端文件
		DeleteFileIfExists(ovpnClient)
		DeleteFileIfExists(ccdClient)
		DeleteFileIfExists(privateClient)
		DeleteFileIfExists(reqsClient)
		DeleteFileIfExists(issuedClient)

		return fmt.Errorf("文件中创建客户端失败")
	}

	// fmt.Println("Client certificate generated and signed successfully.")
	return nil
}

// ----------------------------------------------------------------------------------------------------------
// ShellUpdateClient 更新客户端证书
// ----------------------------------------------------------------------------------------------------------
func ShellUpdateClient(cliId string) error {
	// log.Println("ShellUpdateClient")
	headClient := fmt.Sprintf("%s/openvpn.txt", GlobalOpenVPNPath.ConfigPath)
	caClient := fmt.Sprintf("%s/ca.crt", GlobalOpenVPNPath.PkiPath)
	taClient := fmt.Sprintf("%s/ta.key", GlobalOpenVPNPath.PkiPath)
	ccdClient := fmt.Sprintf("%s/%s", GlobalOpenVPNPath.CcdPath, cliId)
	ovpnClient := fmt.Sprintf("%s/%s.ovpn", GlobalOpenVPNPath.ConfigPath, cliId)
	privateClient := fmt.Sprintf("%s/%s.key", GlobalOpenVPNPath.PrivatePath, cliId)
	reqsClient := fmt.Sprintf("%s/%s.req", GlobalOpenVPNPath.ReqsPath, cliId)
	issuedClient := fmt.Sprintf("%s/%s.crt", GlobalOpenVPNPath.IssuedPath, cliId)

	// Define file paths
	files := map[string]string{
		"key":      privateClient,
		"cert":     issuedClient,
		"ca":       caClient,
		"tls-auth": taClient,
	}

	// log.Println("headClient: ", headClient)
	// log.Println("caClient: ", caClient)
	// log.Println("taClient: ", taClient)
	// log.Println("ccdClient: ", ccdClient)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 删除配置文件
	err := DeleteFileIfExists(ovpnClient)
	if err != nil {
		return fmt.Errorf("无法删除原配置文件")
	}

	// 删除REQS文件
	err = DeleteFileIfExists(reqsClient)
	if err != nil {
		return fmt.Errorf("无法删除原REQS文件")
	}

	// 删除PRIVATE文件
	err = DeleteFileIfExists(privateClient)
	if err != nil {
		return fmt.Errorf("无法删除原PRIVATE文件")
	}

	// 删除ISSUE文件
	err = DeleteFileIfExists(issuedClient)
	if err != nil {
		return fmt.Errorf("无法删除原ISSUE文件")
	}

	// 调用封装的函数
	if err := ChangeDir(GlobalOpenVPNPath.EasyRsaPath); err != nil {
		fmt.Println(err)
	}

	// 创建证书请求 (生成客户端私钥和CSR)
	cmd := exec.Command(GlobalOpenVPNPath.EasyRsaFile, "build-client-full", cliId, "nopass")
	cmd.Env = append(os.Environ(), "EASYRSA_BATCH=1") // 跳过确认步骤

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("无法创建私钥: %v Output: %s", err, string(output))
	}

	// 签发证书
	cmd = exec.Command(GlobalOpenVPNPath.EasyRsaFile, "sign-req", "client", cliId)
	cmd.Env = append(os.Environ(), "EASYRSA_BATCH=1") // 跳过确认步骤

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("无法签发证书: %v Output: %s", err, string(output))
	}

	for _, file := range files {
		if !CheckFileExists(file) {
			return fmt.Errorf("该%s文件不存在", file)
		}
	}

	// Create the .ovpn file
	err = CreateOVPNFile(headClient, ovpnClient, files)
	if err != nil {
		fmt.Println("Error creating .ovpn file:", err)
		return fmt.Errorf("无法合成%s.ovpn, err:%v", cliId, err)
	}

	// log.Printf("Output:\n%s", output)
	if (!CheckFileExists(privateClient)) ||
		(!CheckFileExists(reqsClient)) ||
		(!CheckFileExists(ovpnClient)) ||
		(!CheckFileExists(issuedClient)) {

		// 删除错误客户端文件
		DeleteFileIfExists(ovpnClient)
		DeleteFileIfExists(ccdClient)
		DeleteFileIfExists(privateClient)
		DeleteFileIfExists(reqsClient)
		DeleteFileIfExists(issuedClient)

		return fmt.Errorf("文件中创建客户端失败")
	}

	// fmt.Println("Client certificate generated and signed successfully.")
	return nil
}

// ----------------------------------------------------------------------------------------------------------
// ShellUpdateClient 更新客户端证书
// ----------------------------------------------------------------------------------------------------------
func ShellDelClient(cliId string) error {
	// log.Println("ShellDelClient")
	ccdClient := fmt.Sprintf("%s/%s", GlobalOpenVPNPath.CcdPath, cliId)
	ovpnClient := fmt.Sprintf("%s/%s.ovpn", GlobalOpenVPNPath.ConfigPath, cliId)
	privateClient := fmt.Sprintf("%s/%s.key", GlobalOpenVPNPath.PrivatePath, cliId)
	reqsClient := fmt.Sprintf("%s/%s.req", GlobalOpenVPNPath.ReqsPath, cliId)
	issuedClient := fmt.Sprintf("%s/%s.crt", GlobalOpenVPNPath.IssuedPath, cliId)

	// log.Println("ccdClient: ", ccdClient)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 删除CCD
	err := DeleteFileIfExists(ccdClient)
	if err != nil {
		return fmt.Errorf("无法删除原CCD文件")
	}

	// 删除配置文件
	err = DeleteFileIfExists(ovpnClient)
	if err != nil {
		return fmt.Errorf("无法删除原配置文件")
	}

	// 删除REQS文件
	err = DeleteFileIfExists(reqsClient)
	if err != nil {
		return fmt.Errorf("无法删除原REQS文件")
	}

	// 删除PRIVATE文件
	err = DeleteFileIfExists(privateClient)
	if err != nil {
		return fmt.Errorf("无法删除原PRIVATE文件")
	}

	// 删除ISSUE文件
	err = DeleteFileIfExists(issuedClient)
	if err != nil {
		return fmt.Errorf("无法删除原ISSUE文件")
	}

	return nil

}

// ----------------------------------------------------------------------------------------------------------
// ChangeDir 切换当前工作目录到指定路径
// ----------------------------------------------------------------------------------------------------------
func ChangeDir(dir string) error {
	// 切换到目标目录
	err := os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("切换目录失败: %v", err)
	}

	// 获取并返回当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}

	log.Println("当前目录:", currentDir)
	return nil
}

// ----------------------------------------------------------------------------------------------------------
// ParseConfigFile 解析给定路径的配置文件，并返回 ifconfig-push 和 iroute 的值
// ----------------------------------------------------------------------------------------------------------
func ParseConfigFile(filePath string) (ipAddress, netmask string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "ifconfig-push") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				ipAddress = parts[1]
				netmask = parts[2]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("读取文件错误: %w", err)
	}

	return ipAddress, netmask, nil
}

func ReadFile(filepath string) ([]byte, error) {
	return ioutil.ReadFile(filepath)
}

func WriteFileWithTags(outputFile *os.File, tag, filepath string) error {
	content, err := ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = outputFile.WriteString(fmt.Sprintf("<%s>\n", tag))
	if err != nil {
		return err
	}
	_, err = outputFile.Write(content)
	if err != nil {
		return err
	}
	_, err = outputFile.WriteString(fmt.Sprintf("</%s>\n", tag))
	return err
}

func CreateOVPNFile(templatePath, outputPath string, files map[string]string) error {
	// Open the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Read template file
	templateContent, err := ReadFile(templatePath)
	if err != nil {
		return err
	}
	// Write template content
	_, err = outputFile.Write(templateContent)
	if err != nil {
		return err
	}

	// Write all sections
	for tag, filepath := range files {
		err = WriteFileWithTags(outputFile, tag, filepath)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkIptablesRule checks if a specific iptables rule exists.
func CheckIptablesRule(rule string) bool {
	args := append([]string{"-C", "FORWARD"}, strings.Fields(rule)...)
	cmd := exec.Command("iptables", args...)
	err := cmd.Run()
	return err == nil
}

// addIptablesRule adds an iptables rule.
func AddIptablesRule(rule string) error {
	args := append([]string{"-A", "FORWARD"}, strings.Fields(rule)...)
	cmd := exec.Command("iptables", args...)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to add iptables rule: %v", err)
	}
	return nil
}

// deleteIptablesRule deletes an iptables rule.
func DeleteIptablesRule(rule string) error {
	args := append([]string{"-D", "FORWARD"}, strings.Fields(rule)...)
	cmd := exec.Command("iptables", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to delete iptables rule: %v, output: %s", err, out.String())
	}
	return nil
}

func SplitIP(ip string) (string, string) {
	// 使用 strings.LastIndex 分割最后一个点的位置
	lastDotIndex := strings.LastIndex(ip, ".")
	if lastDotIndex == -1 {
		return "", ""
	}

	// 分割 IP 地址和最后一部分
	ipPart := ip[:lastDotIndex]
	lastPart := ip[lastDotIndex+1:]

	return ipPart, lastPart
}

// 解析 CIDR 并输出 IP、子网掩码和网络地址
func ParseCIDR(cidr string) (string, string, string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", "", err
	}

	// 获取 IP 地址
	ipAddress := ip.String()

	// 获取子网掩码
	subnetMask := net.IP(ipNet.Mask).String()

	// 计算网络地址
	networkAddress := ip.Mask(ipNet.Mask).String()

	return ipAddress, subnetMask, networkAddress, nil
}

// 每个连接都通过一个goroutine独立处理
func HandleConnection(conn net.Conn) {
	// 在函数结束时关闭连接，确保资源被释放
	defer conn.Close()

	// 打印连接的客户端地址信息
	// fmt.Printf("Accepted connection from %v\n", conn.RemoteAddr())

	// 创建一个缓冲读取器，用于从TCP连接中逐行读取数据
	reader := bufio.NewReader(conn)

	// 无限循环，处理客户端发送的数据
	for {
		// 从连接中读取一行数据，直到遇到换行符 '\n'
		message, err := reader.ReadString('\n')
		if err != nil {
			// 如果读取数据时出错（如客户端断开连接），打印错误并退出循环
			fmt.Println("[global] Error reading from connection:", err)
			break
		}

		// 打印接收到的消息
		// fmt.Printf("Received message: %s", message)

		// 将接收到的消息原样返回给客户端
		_, err = conn.Write([]byte(message))
		if err != nil {
			// 如果写入数据时发生错误，打印错误并退出循环
			log.Println("[global] Error writing to connection:", err)
			break
		}
	}
}
