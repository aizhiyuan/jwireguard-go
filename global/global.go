// myapp/global/global.go
package global

import (
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
	"net/http"
	"os"
	"time"

	"gopkg.in/ini.v1"
)

// JWireGuardIni 结构体用于封装 INI 文件的数据
type JWireGuardIni struct {
	IPPrefix    string // IP前缀
	DefaultUser string // 默认用户
	OpenVpnPath string // OpenVpn 路径
}

// ----------------------------------------------------------------------------------------------------------
// 全局变量
// ----------------------------------------------------------------------------------------------------------
var GlobalDB *sql.DB // This is a global variable
var GlobalDefaultUserMd5 string
var GlobalJWireGuardini *JWireGuardIni
var GlobalEncryptKey string

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
func uniqueStrings(input []string) []string {
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
// func GenerateSHA3Hash(data string) string {
// 	h := sha3.New256()
// 	h.Write([]byte(data))
// 	hash := h.Sum(nil)
// 	return hex.EncodeToString(hash)
// }

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
		cfg.Section("GENERAL SETTING").Key("IP_PREFIX").SetValue("10.100")
		cfg.Section("GENERAL SETTING").Key("DEFAULT_USER").SetValue("admin")
		cfg.Section("GENERAL SETTING").Key("OPENVPN_PATH").SetValue("/etc/openvpn")

		// 保存到文件
		if err = cfg.SaveTo(filePath); err != nil {
			return nil, fmt.Errorf("failed to create new ini file: %v", err)
		}
		log.Println("INI file created with default values")
	} else {
		// 文件存在，读取配置
		cfg, err = ini.Load(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load ini file: %v", err)
		}
	}

	// 加载配置到结构体
	jwg := &JWireGuardIni{
		IPPrefix:    cfg.Section("GENERAL SETTING").Key("IP_PREFIX").String(),
		DefaultUser: cfg.Section("GENERAL SETTING").Key("DEFAULT_USER").String(),
		OpenVpnPath: cfg.Section("GENERAL SETTING").Key("OPENVPN_PATH").String(),
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

// 从指定 URL 获取时间
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
