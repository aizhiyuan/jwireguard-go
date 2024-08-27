// webservice/webservice.go
package webservice

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ResponseError struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Error   int    `json:"error"`
}
type ResponseSuccess struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func StartServer(port string) {
	http.HandleFunc("/", homeHandler)

	// Register CLI-related routes
	registerCliRoutes()

	// Register user-related routes
	registerUserRoutes()

	// Register subnet-related routes
	registerSubnetRoutes()

	http.ListenAndServe(port, nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("{\"service\":\"API interface is normal\"}"))
}

// 从请求中解析JSON并封装到指定的结构体中
func parseJSONBody(r *http.Request, v interface{}) error {
	// 确保请求体在使用后被关闭
	defer r.Body.Close()

	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body: %v", err)
	}

	// 将JSON解码到指定的结构体中
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid JSON format: %v", err)
	}

	return nil
}

// ----------------------------------------------------------------------------------------------------------
//
//	FindUnusedIP 找到未使用的IP地址，如果全部IP都被占用则返回错误
//
// ----------------------------------------------------------------------------------------------------------
func FindUnusedIP(ccdDir string, ipPrefix string) (string, error) {
	usedIPs := make(map[string]bool)

	// 遍历CCD目录下的所有文件
	err := filepath.Walk(ccdDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// 读取文件内容
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "ifconfig-push") {
					fields := strings.Fields(line)
					if len(fields) > 1 && strings.HasPrefix(fields[1], ipPrefix) {
						usedIPs[fields[1]] = true
					}
				}
			}

			if err := scanner.Err(); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	// 找到未使用的IP地址
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s.%d", ipPrefix, i)
		if !usedIPs[ip] {
			return ip, nil
		}
	}

	// 如果所有IP地址都被占用，返回错误
	return "", errors.New("all IP addresses in the range are used")
}
