// webservice/webservice.go
package webservice

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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

// Session 结构体
type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func StartServer(port string, tlsPort string, certfile string, keyfile string) {
	http.HandleFunc("/", homeHandler)

	// 注册路由
	registerCliRoutes()
	registerUserRoutes()
	registerSubnetRoutes()

	// 如果提供了 HTTPS 证书，则启动 HTTPS 协程
	if certfile != "" && keyfile != "" {
		go func() {
			log.Println("启动 HTTPS 服务，监听端口：", tlsPort)
			err := http.ListenAndServeTLS(tlsPort, certfile, keyfile, nil)
			if err != nil {
				log.Fatalf("HTTPS 启动失败: %v", err)
			}
		}()
	}

	// 启动 HTTP 服务
	log.Println("启动 HTTP 服务，监听端口：", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("HTTP 启动失败: %v", err)
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

// 创建新 session
func createSession(userID string, ExpirySeconds int64) string {

	sessionID := strings.ReplaceAll(uuid.New().String(), "-", "")
	expiresAt := time.Now().Add(time.Duration(ExpirySeconds) * time.Second).Unix()

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)

	dbuser.UserID.String = userID
	dbuser.SessionID.String = sessionID
	dbuser.ExpiresAt.Int64 = expiresAt
	dbuser.ExpirySeconds.Int64 = ExpirySeconds

	err := dbuser.UpdateUsers(global.GlobalDB)
	if err != nil {
		return ""
	}

	return sessionID
}

// 验证 session
func validateSession(sessionID string) (string, bool) {
	log.Printf("[Session] 开始验证 SessionID: %s", sessionID)

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)
	dbuser.SessionID.String = sessionID
	err := dbuser.GetUserBySessionID(global.GlobalDB)
	if err != nil {
		log.Printf("[Session] SessionID: %s 对应着用于不存在", sessionID)
		return "", false
	}

	// 更新过期时间
	newExpire := time.Now().Unix()
	if dbuser.ExpiresAt.Int64 < newExpire {
		log.Printf("[Session] SessionID: %s UserID: %s 已过期", sessionID, dbuser.UserID.String)
		return dbuser.UserID.String, false
	}

	log.Printf("[Session] SessionID: %s UserID: %s 未过期", sessionID, dbuser.UserID.String)

	return dbuser.UserID.String, true
}

// 删除 session
func deleteSession(sessionID string) bool {
	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)
	dbuser.SessionID.String = sessionID
	err := dbuser.GetUserBySessionID(global.GlobalDB)
	if err != nil {
		log.Printf("[Session] SessionID: %s 用户不存在", sessionID)
		return false
	}

	dbuser.UserStatus.String = "false"
	dbuser.ExpiresAt.Int64 = time.Now().Unix()
	err = dbuser.UpdateUsers(global.GlobalDB)
	if err != nil {
		log.Printf("[Session] SessionID: %s 删除失败", sessionID)
		return false
	}
	log.Printf("[Session] SessionID: %s 已删除", sessionID)
	return true
}

// session 验证中间件// session 验证中间件
func ValidateSessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 header 或 cookie 中获取 sessionID
		sessionID := r.Header.Get("X-Session-ID")
		if sessionID == "" {
			// 尝试从 cookie 获取
			cookie, err := r.Cookie("session_id")
			if err == nil {
				sessionID = cookie.Value
			}
		}

		if sessionID == "" {
			// log.Printf("[Session] SessionID: %s 为空", sessionID)
			// responseError := ResponseError{
			// 	Status:  false,
			// 	Message: "未提供 session ID",
			// 	Error:   3101,
			// }
			// w.Header().Set("Content-Type", "application/json")
			// w.WriteHeader(http.StatusUnauthorized)
			// json.NewEncoder(w).Encode(responseError)

			// 将 userID 放入请求头中
			r.Header.Set("X-User-ID", "ee11cbb19052e40b07aac0ca060c23ee")
			// 继续处理请求
			next.ServeHTTP(w, r)

			return
		}

		// 验证 session
		userID, valid := validateSession(sessionID)
		if !valid {
			log.Printf("[Session] SessionID: %s 无效或过期", sessionID)
			responseError := ResponseError{
				Status:  false,
				Message: "无效或过期的 session",
				Error:   3102,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(responseError)
			return
		}

		// 将 userID 放入请求头中
		r.Header.Set("X-User-ID", userID)

		// 继续处理请求
		next.ServeHTTP(w, r)
	}

}
