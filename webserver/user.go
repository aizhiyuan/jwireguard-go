package webservice

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net"
	"net/http"
	"time"
)

type UserEditConfig struct {
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	UserPasswd string `json:"user_passwd"`
	UserEmail  string `json:"user_email"`
}

// 新增用户有效期配置结构体
type SessionExpiryConfig struct {
	UserID        string `json:"user_id"`
	ExpirySeconds int64  `json:"expiry_seconds"` // 以秒为单位
}

// 定义一个结构体，用于存储用户登录信息
type PostCheckUserLogin struct {
	UserName   string `json:"user_name"`
	UserPasswd string `json:"user_passwd"`
}

// 定义一个结构体，用于存储登录结果
type ResponseCheckLogin struct {
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	SessionID string `json:"session_id"` // 新增 SessionID 字段
}

// 注册用户路由
func registerUserRoutes() {
	// 不需要 session 验证的路由
	http.HandleFunc("/check_users_login", CheckUsersLogin)
	http.HandleFunc("/logout", LogoutUser)

	// 需要 session 验证的路由
	http.HandleFunc("/add_user", ValidateSessionMiddleware(AddUser))
	http.HandleFunc("/edit_user", ValidateSessionMiddleware(EditUser))
	http.HandleFunc("/del_user", ValidateSessionMiddleware(DelUser))
	http.HandleFunc("/edit_session", ValidateSessionMiddleware(EditSession))
}

// LogoutUser 处理登出请求
func LogoutUser(w http.ResponseWriter, r *http.Request) {
	// 从 header 或 cookie 中获取 sessionID
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		// 尝试从 cookie 获取
		cookie, err := r.Cookie("session_id")
		if err == nil {
			sessionID = cookie.Value
		}
	}

	if sessionID != "" {
		deleteSession(sessionID)
	}

	// 清除 cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   "",
		Expires: time.Unix(0, 0),
		Path:    "/",
	})

	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "登出成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

// 用户登录验证
func CheckUsersLogin(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[check_users_login] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}

	log.Printf("[check_users_login] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[check_users_login] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   3301,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据
	portCheckUserLogin := PostCheckUserLogin{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portCheckUserLogin); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[check_users_login] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   3302,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	log.Printf("[check_users_login] josn:[%+v]", portCheckUserLogin)
	if portCheckUserLogin.UserName == "" ||
		portCheckUserLogin.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[check_users_login] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3303,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)

	// 查询账号
	dbuser.UserName.String = portCheckUserLogin.UserName
	err = dbuser.GetUserByName(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[check_users_login] 账号认证失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("账号认证失败, err:%v", err),
			Error:   3304,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 对密码进行解密
	decryptPasswd, err := global.Decrypt(dbuser.UserPasswd.String, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[check_users_login] 无法对密码进行解密, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法对密码进行解密, err:%v", err),
			Error:   3305,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 账号认证
	if decryptPasswd != portCheckUserLogin.UserPasswd {
		log.Println("[check_users_login] 账号认证失败")
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("账号认证失败, err:%v", err),
			Error:   3306,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	if dbuser.ExpirySeconds.Int64 <= 0 {
		dbuser.ExpirySeconds.Int64 = 60 * 10
	}
	// 创建 session
	sessionID := createSession(dbuser.UserID.String, dbuser.ExpirySeconds.Int64)

	// 设置 session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // 生产环境应该启用
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// 返回结果
	responseCheckLogin := ResponseCheckLogin{
		Status:    true,
		Message:   "登录成功!",
		UserID:    dbuser.UserID.String,
		UserEmail: dbuser.UserEmail.String,
		SessionID: sessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseCheckLogin)
}

// 更新Session ID 有效期
// webservice/handler_session_expiry.go
func EditSession(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[edit_session] userID:", XUserID)
	if !global.IsAdmin(XUserID) { // 需要实现权限验证逻辑
		responseError := ResponseError{
			Status:  false,
			Message: "权限不足",
			Error:   2501,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[edit_session] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[edit_session] client [%s:%s]", ip, port)

	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[edit_session] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2502,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	var config SessionExpiryConfig
	// 创建数据库对象
	user := database.User{}
	// 初始化数据库
	user.CreateUser(global.GlobalDB)

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[edit_session] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2503,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 参数验证
	if config.ExpirySeconds < 60 {
		log.Printf("[edit_session] seconds:%d 有效期不能小于60秒", config.ExpirySeconds)
		responseError := ResponseError{
			Status:  false,
			Message: "有效期不能小于60秒",
			Error:   3402,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)

	dbuser.UserID.String = config.UserID
	dbuser.ExpirySeconds.Int64 = config.ExpirySeconds

	// 判断用户是否存在
	err = dbuser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_session] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   2503,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新用户
	err = dbuser.UpdateUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_session] 用户修改失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户修改失败, err:%v", err),
			Error:   2504,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	resp := map[string]interface{}{
		"status":     true,
		"message":    "更新Session ID成功",
		"session_id": "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 添加用户
func AddUser(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[add_user] userID:", XUserID)

	if !global.IsAdmin(XUserID) { // 需要实现权限验证逻辑
		responseError := ResponseError{
			Status:  false,
			Message: "权限不足",
			Error:   2601,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[add_user] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[add_user] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_user] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2602,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	exporteUser := database.ExportedUser{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &exporteUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[add_user] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2603,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	log.Printf("[add_user] josn:[%+v]", exporteUser)
	// 创建一个PostCliConfig实例来存储解析后的数据
	portUser := exporteUser.ConvertToUser()

	if portUser.UserName.String == "" ||
		portUser.SerID.String == "" ||
		portUser.UserPasswd.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[add_user] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2604,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 生成用户ID
	if portUser.UserID.String == "" {
		portUser.UserID.String = global.GenerateMD5(portUser.UserName.String)
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	// 判断用户是否存在
	err = portUser.GetUserByID(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_user] 用户已存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户已存在, err:%v", err),
			Error:   2605,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 对密码进行加密
	encryptedPasswd, err := global.Encrypt(portUser.UserPasswd.String, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_user] 密码格式错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("密码格式错误, err:%v", err),
			Error:   2606,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加用户
	portUser.UserPasswd.String = encryptedPasswd
	err = portUser.InsertUser(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_user] 添加用户失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("添加用户失败, err:%v", err),
			Error:   2607,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	cliConfig := database.CliConfig{}
	// 初始化数据库
	cliConfig.CreateCliConfig(global.GlobalDB)

	userAddr := fmt.Sprintf("%s.0.1", global.GlobalJWireGuardini.IPPrefix)

	// 添加客户端
	err = global.ShellAddClient(portUser.UserID.String, userAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_user] 添加用户配置失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("添加用户配置失败, err:%v", err),
			Error:   2608,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfig.CliID = portUser.UserID
	cliConfig.CliName = portUser.UserName
	cliConfig.CliMapping.String = ""
	cliConfig.CliStatus.String = "false"
	cliConfig.EditStatus.Int32 = 0
	cliConfig.CliSN.String = ""
	cliConfig.SerID.String = ""
	cliConfig.SerName.String = ""
	cliConfig.Timestamp.Int64 = 0
	cliConfig.OnlineStatus.String = "true"

	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		err = cliConfig.InsertCliConfig(global.GlobalDB)
		if err != nil {
			log.Printf("[add_user] 数据库创建客户端失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("数据库创建客户端失败, err:%v", err),
				Error:   2609,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	} else {
		err = cliConfig.UpdateCliConfig(global.GlobalDB)
		if err != nil {
			log.Printf("[add_user] 数据库创建客户端失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("数据库创建客户端失败, err:%v", err),
				Error:   2610,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// 返回 JSON 响应
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "添加用户成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

// 编辑用户
func EditUser(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[edit_user] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[edit_user] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[edit_user] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[edit_user] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2701,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据

	portUser := database.User{}

	if global.IsAdmin(XUserID) {
		exporteUser := database.ExportedUser{}
		// 使用封装的parseJSONBody函数解析请求体
		if err := parseJSONBody(r, &exporteUser); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Printf("[edit_user] 解析JSON请求参数错误, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
				Error:   2702,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		log.Printf("[edit_user] josn:[%+v]", exporteUser)
		portUser = exporteUser.ConvertToUser()
	} else {
		userEditConfig := UserEditConfig{}
		// 使用封装的parseJSONBody函数解析请求体
		if err := parseJSONBody(r, &userEditConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Printf("[edit_user] 解析JSON请求参数错误, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
				Error:   2702,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		portUser.UserID.String = userEditConfig.UserID
		portUser.UserName.String = userEditConfig.UserName
		portUser.UserPasswd.String = userEditConfig.UserPasswd
		portUser.UserEmail.String = userEditConfig.UserEmail
	}

	if portUser.UserID.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[edit_user] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2703,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)
	portUserbak := portUser

	// 判断用户是否存在
	err = portUserbak.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_user] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   2704,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portUser.UserPasswd.String != "" {
		// 对密码进行加密
		encryptedPasswd, err := global.Encrypt(portUser.UserPasswd.String, global.GlobalEncryptKey)
		if err != nil {
			// 如果参数为空，返回 JSON 错误响应
			log.Printf("[edit_user] 密码错误, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("密码错误, err:%v", err),
				Error:   2705,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		portUser.UserPasswd.String = encryptedPasswd
	}

	// 更新用户
	err = portUser.UpdateUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_user] 用户修改失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户修改失败, err:%v", err),
			Error:   2706,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回 JSON 响应
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "用户修改成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

// 删除用户
func DelUser(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[del_user] userID:", XUserID)
	if !global.IsAdmin(XUserID) { // 需要实现权限验证逻辑
		responseError := ResponseError{
			Status:  false,
			Message: "权限不足",
			Error:   2801,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[del_user] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[del_user] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	targetUserID := query.Get("user_id")
	log.Printf("[del_user] user_id:[%s]", targetUserID)
	// 判断参数是否为空
	if targetUserID == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[del_user] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   2802,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 创建数据库对象
	user := database.User{}
	// 创建数据库对象
	cliConfig := database.CliConfig{}

	// 初始化数据库
	user.CreateUser(global.GlobalDB)
	// 初始化数据库
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 查看子网是否存在
	user.UserID.String = targetUserID

	err = user.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[del_user] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   2803,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	cliConfig.CliID.String = targetUserID

	// 删除子网
	err = user.DeleteUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[del_user] 用户删除失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户删除失败, err:%v", err),
			Error:   2804,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 执行SHELL命令
	err = global.ShellDelClient(targetUserID)
	if err != nil {
		log.Printf("[del_user] 无法删除客户端, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法删除客户端, err:%v", err),
			Error:   2805,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		err = cliConfig.DeleteCliConfig(global.GlobalDB)
		if err != nil {
			log.Printf("[del_user] 无法删除客户端, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("无法删除客户端, err:%v", err),
				Error:   2806,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "用户删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
