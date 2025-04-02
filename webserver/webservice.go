// webservice/webservice.go
package webservice

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
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

func StartServer(port string, certfile string, keyfile string) {
	http.HandleFunc("/", homeHandler)

	// Register CLI-related routes
	registerCliRoutes()

	// Register user-related routes
	registerUserRoutes()

	// Register subnet-related routes
	registerSubnetRoutes()

	if (certfile != "") && (keyfile != "") {
		http.ListenAndServeTLS(port, certfile, keyfile, nil)
	} else {
		http.ListenAndServe(port, nil)
	}
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
					// log.Printf("[未判断] 文件中的IP:%s", fields[1])
					// 调用 splitIP 函数
					if len(fields) > 1 && fields[1] != "" {
						newIP, _ := SplitIP(fields[1])
						if newIP == ipPrefix {
							// log.Printf("[已判断] 文件中的IP:%s", fields[1])
							usedIPs[fields[1]] = true

						}
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
		useIPStatus := usedIPs[ip]
		log.Printf("[判断状态] 当前IP:%s 当前状态:%v", ip, useIPStatus)
		if !useIPStatus {
			return ip, nil
		}
	}

	// 如果所有IP地址都被占用，返回错误
	return "", errors.New("all IP addresses in the range are used")
}

// SplitIP 函数用于将 IP 地址分割成前三部分和最后一部分
func SplitIP(ip string) (string, string) {
	// 将 IP 地址按照 "." 分割
	parts := strings.Split(ip, ".")

	// 组合前三部分
	newIP := strings.Join(parts[:3], ".")

	// 获取最后一部分
	lastPart := parts[3]

	return newIP, lastPart
}
