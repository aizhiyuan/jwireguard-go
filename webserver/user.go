package webservice

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	"jwireguard/message"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type UserEditConfig struct {
	UserID     string `json:"user_id"`
	UserName   string `json:"user_name"`
	UserPasswd string `json:"user_passwd"`
	UserEmail  string `json:"user_email"`
	UserMac    string `json:"user_mac"`
}

// 新增用户有效期配置结构体
type SessionExpiryConfig struct {
	UserID        string `json:"user_id"`
	ExpirySeconds int32  `json:"expiry_seconds"`
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
	UserMac   string `json:"user_mac"`
	SessionID string `json:"session_id"`
}

// 定义一个结构体，用于存储通用的成功响应结果
type UserLoginConfig struct {
	Message        string `json:"message"`
	Status         bool   `json:"status"`
	UserID         string `json:"user_id"`
	LoginErrCount  int32  `json:"login_err_count"`
	LoginErrTime   int32  `json:"login_err_time"`
	LimitLoginTime int32  `json:"limit_login_time"`
}

type UserMail struct {
	UserID    string `json:"user_id"`
	EmailCode string `json:"mail_code"`
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
	http.HandleFunc("/edit_session", EditSession)

	http.HandleFunc("/get_user_config", ValidateSessionMiddleware(GetUserConfig))
	http.HandleFunc("/set_user_config", ValidateSessionMiddleware(SetUserConfig))

	http.HandleFunc("/get_mail_code", GetMailCode)
	http.HandleFunc("/check_mail_code", CheckMailCode)
}

// LogoutUser处理登出请求
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
		global.Log.Errorf("[check_users_login] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}

	log.Printf("[check_users_login] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[check_users_login] 请求类型不是Post")
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
		global.Log.Errorf("[check_users_login] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   3302,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[check_users_login] josn:[%+v]", portCheckUserLogin)
	if portCheckUserLogin.UserName == "" ||
		portCheckUserLogin.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[check_users_login] 请求参数为空")
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
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[check_users_login] 数据库连接失败, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库连接失败, err:%v", err),
			Error:   0001,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	dbuser := database.User{}
	dhloginHistory := database.LoginHistory{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)
	dhloginHistory.CreateLoginHistory(global.GlobalDB)

	// 查询账号
	dbuser.UserName.String = portCheckUserLogin.UserName
	err = dbuser.GetUserByName(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[check_users_login] 账号认证失败, err:%v", err)
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
		global.Log.Errorf("[check_users_login] 无法对密码进行解密, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法对密码进行解密, err:%v", err),
			Error:   3305,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	dhloginHistory.UserID.String = dbuser.UserID.String
	dhloginHistory.LoginTime.Int64 = time.Now().Unix()

	// 判断账号否被锁
	loginState, err := dhloginHistory.CheckLockStatus(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[check_users_login] 检查账号锁定状态失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("检查账号锁定状态失败, err:%v", err),
			Error:   3306,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if loginState {
		responseError := ResponseError{
			Status:  false,
			Message: "账号被锁定",
			Error:   3307,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
	}

	// 账号认证
	if decryptPasswd != portCheckUserLogin.UserPasswd {
		global.Log.Debugln("[check_users_login] 账号认证失败")

		dhloginHistory.LoginStatus.String = "false"
		// 记录登录失败
		err = dhloginHistory.InsertLoginHistory(global.GlobalDB)
		if err != nil {
			global.Log.Errorln("[check_users_login] 插入登录历史失败:", err)
		}

		// 检查是否需要锁定
		shouldLock, unlockTime, err := dhloginHistory.HandleFailedLogin(
			global.GlobalDB,
			dbuser.LoginErrTime.Int32,
			dbuser.LoginErrCount.Int32,
			dbuser.LimitLoginTime.Int32,
		)

		if err != nil {
			global.Log.Errorln("[check_users_login] 处理登录失败失败:", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("处理登录失败失败: %v", err),
				Error:   3308,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}

		if shouldLock {
			global.Log.Errorf("[check_users_login] 登录失败次数过多，账户已被锁定至 %s", unlockTime.Format("2006-01-02 15:04:05"))
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("登录失败次数过多，账户已被锁定至 %s", unlockTime.Format("2006-01-02 15:04:05")),
				Error:   3309,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}

		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("账号认证失败, err:%v", err),
			Error:   3310,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 登录成功后解除锁定（如果之前已被锁定）
	if err := dhloginHistory.UnlockUser(global.GlobalDB, true); err != nil {
		global.Log.Errorln("[check_users_login] 解锁用户失败:", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解锁用户失败, err:%v", err),
			Error:   3311,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
	}

	global.Log.Debugln("[check_users_login] 账号认证成功")
	dhloginHistory.LoginStatus.String = "true"

	// 记录登录成功
	err = dhloginHistory.InsertLoginHistory(global.GlobalDB)
	if err != nil {
		global.Log.Errorln("[check_users_login] 插入登录历史记录出错:", err)
	}

	if dbuser.ExpirySeconds.Int32 <= 0 {
		dbuser.ExpirySeconds.Int32 = 600 // 默认有效期为10分钟
	}

	// 创建 session
	sessionID := createSession(dbuser.UserID.String, dbuser.ExpirySeconds.Int32)

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
		UserMac:   dbuser.UserMac.String,
		SessionID: sessionID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseCheckLogin)
}

// 更新SessionID有效期
func EditSession(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[edit_session] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[edit_session] client [%s:%s]", ip, port)

	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[edit_session] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2501,
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
		global.Log.Errorf("[edit_session] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2502,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 参数验证
	// if config.ExpirySeconds < 60 {
	// 	global.Log.Errorf("[edit_session] seconds:%d 有效期不能小于60秒", config.ExpirySeconds)
	// 	responseError := ResponseError{
	// 		Status:  false,
	// 		Message: "有效期不能小于60秒",
	// 		Error:   3403,
	// 	}
	// 	w.Header().Set("Content-Type", "application/json")
	// 	json.NewEncoder(w).Encode(responseError)
	// 	return
	// }

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)

	dbuser.UserID.String = config.UserID

	// 判断用户是否存在
	err = dbuser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[edit_session] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   2504,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	dbuser.ExpirySeconds.Int32 = config.ExpirySeconds
	dbuser.ExpiresAt.Int64 = time.Now().Unix() + int64(config.ExpirySeconds)

	// 更新用户
	err = dbuser.UpdateUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[edit_session] 更新Session ID失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("更新Session ID失败, err:%v", err),
			Error:   2505,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	resp := map[string]interface{}{
		"status":  true,
		"message": "更新Session ID成功",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 添加用户
func AddUser(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[add_user] userID:", XUserID)

	if !global.IsAdmin(XUserID) { // 需要实现权限验证逻辑
		// 如果权限不足，返回 JSON 错误响应
		global.Log.Errorf("[add_user] 权限不足, userID:%s", XUserID)
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
		global.Log.Errorf("[add_user] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[add_user] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[add_user] 请求类型不是Post")
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
		global.Log.Errorf("[add_user] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2603,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	global.Log.Debugf("[add_user] josn:[%+v]", exporteUser)
	// 创建一个PostCliConfig实例来存储解析后的数据
	portUser := exporteUser.ConvertToUser()

	if portUser.UserName.String == "" ||
		portUser.SerID.String == "" ||
		portUser.UserPasswd.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[add_user] 请求参数为空")
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
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[add_user] 数据库连接失败, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库连接失败, err:%v", err),
			Error:   0001,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	// 判断用户是否存在
	err = portUser.GetUserByID(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[add_user] 用户已存在, err:%v", err)
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
		global.Log.Errorf("[add_user] 密码格式错误, err:%v", err)
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
		global.Log.Errorf("[add_user] 添加用户失败, err:%v", err)
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
		global.Log.Errorf("[add_user] 添加用户配置失败, err:%v", err)
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
			global.Log.Errorf("[add_user] 数据库创建客户端失败, err:%v", err)
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
			global.Log.Errorf("[add_user] 数据库创建客户端失败, err:%v", err)
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
		global.Log.Errorf("[edit_user] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[edit_user] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[edit_user] 请求类型不是Post")
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
			global.Log.Errorf("[edit_user] 解析JSON请求参数错误, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
				Error:   2702,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		global.Log.Debugf("[edit_user] josn:[%+v]", exporteUser)
		portUser = exporteUser.ConvertToUser()
	} else {
		userEditConfig := UserEditConfig{}
		// 使用封装的parseJSONBody函数解析请求体
		if err := parseJSONBody(r, &userEditConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			global.Log.Errorf("[edit_user] 解析JSON请求参数错误, err:%v", err)
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
		portUser.UserMac.String = userEditConfig.UserMac
	}

	if portUser.UserID.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[edit_user] 请求参数为空")
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
		global.Log.Errorf("[edit_user] 用户不存在, err:%v", err)
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
			global.Log.Errorf("[edit_user] 密码错误, err:%v", err)
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
		global.Log.Errorf("[edit_user] 用户修改失败, err:%v", err)
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
	global.Log.Errorln("[del_user] userID:", XUserID)
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
		global.Log.Errorf("[del_user] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[del_user] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	targetUserID := query.Get("user_id")
	global.Log.Debugf("[del_user] user_id:[%s]", targetUserID)
	// 判断参数是否为空
	if targetUserID == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[del_user] 参数为空")
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
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[del_user] 数据库连接失败, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库连接失败, err:%v", err),
			Error:   0001,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

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
		global.Log.Errorf("[del_user] 用户不存在, err:%v", err)
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
		global.Log.Errorf("[del_user] 用户删除失败, err:%v", err)
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
		global.Log.Errorf("[del_user] 无法删除客户端, err:%v", err)
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
			global.Log.Errorf("[del_user] 无法删除客户端, err:%v", err)
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

// 获取用户配置
func GetUserConfig(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[get_user_config] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_user_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_user_config] client [%s:%s]", ip, port)

	// 解析 URL 参数
	query := r.URL.Query()
	userId := query.Get("user_id")
	global.Log.Debugf("[get_user_config] user_id:[%s]", userId)
	// 判断参数是否为空
	if userId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_user_config] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   2901,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_user_config] 数据库连接失败, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库连接失败, err:%v", err),
			Error:   2902,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)

	// 查询账号
	dbuser.UserID.String = userId
	err = dbuser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[get_user_config] 获取用户配置错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取用户配置错误, err:%v", err),
			Error:   2903,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	UserLoginConfig := UserLoginConfig{
		Message:        "获取用户配置成功",
		Status:         true,
		UserID:         dbuser.UserID.String,
		LoginErrCount:  dbuser.LoginErrCount.Int32,
		LoginErrTime:   dbuser.LoginErrTime.Int32,
		LimitLoginTime: dbuser.LimitLoginTime.Int32,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserLoginConfig)
}

// 设置用户配置
func SetUserConfig(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[set_user_config] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[set_user_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[set_user_config] client [%s:%s]", ip, port)

	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[set_user_config] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   3001,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	var userLoginConfig UserLoginConfig
	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &userLoginConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		global.Log.Errorf("[set_user_config] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   3002,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[set_user_config] josn:[%+v]", userLoginConfig)

	if userLoginConfig.UserID == "" {
		global.Log.Errorln("[set_user_config] 用户ID不能为空")
		responseError := ResponseError{
			Status:  false,
			Message: "用户ID不能为空",
			Error:   3003,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if userLoginConfig.LoginErrTime < 0 || userLoginConfig.LoginErrCount < 0 || userLoginConfig.LimitLoginTime < 0 {
		global.Log.Errorln("[set_user_config] 登录错误时间不能小于0")
		responseError := ResponseError{
			Status:  false,
			Message: "登录错误时间不能小于0",
			Error:   3004,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
	}

	portUser := database.User{}
	portUser.UserID.String = userLoginConfig.UserID

	// 判断用户是否存在
	err = portUser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[set_user_config] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   3005,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	portUser.LoginErrTime.Int32 = userLoginConfig.LoginErrTime
	portUser.LoginErrCount.Int32 = userLoginConfig.LoginErrCount
	portUser.LimitLoginTime.Int32 = userLoginConfig.LimitLoginTime

	// 更新用户
	err = portUser.UpdateUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[set_user_config] 用户修改失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户修改失败, err:%v", err),
			Error:   3006,
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

func CheckMailCode(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[check_user_mail] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[check_user_mail] client [%s:%s]", ip, port)

	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[check_user_mail] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   3101,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	var userMail UserMail
	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &userMail); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		global.Log.Errorf("[check_user_mail] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   3102,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[check_user_mail] josn:[%+v]", userMail)

	if userMail.UserID == "" {
		global.Log.Errorln("[check_user_mail] 用户ID不能为空")
		responseError := ResponseError{
			Status:  false,
			Message: "用户ID不能为空",
			Error:   3103,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if userMail.EmailCode == "" {
		global.Log.Errorln("[check_user_mail] 验证码不能为空")
		responseError := ResponseError{
			Status:  false,
			Message: "验证码不能为空",
			Error:   3104,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	portUser := database.User{}
	portUser.UserID.String = userMail.UserID

	// 判断用户是否存在
	err = portUser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[check_user_mail] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   3105,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 验证邮箱验证码是否过期
	if portUser.MailTime.Int64 <= 0 || portUser.MailTime.Int64 < time.Now().Unix() {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[check_user_mail] 验证邮箱验证码已过期")
		responseError := ResponseError{
			Status:  false,
			Message: "验证邮箱验证码已过期",
			Error:   3106,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 验证邮箱验证码
	if portUser.MailCode.String != userMail.EmailCode {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[check_user_mail] 验证邮箱验证码错误")
		responseError := ResponseError{
			Status:  false,
			Message: "验证邮箱验证码错误",
			Error:   3107,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回 JSON 响应
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "邮箱认证成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)

}

// 获取邮箱验证码
func GetMailCode(w http.ResponseWriter, r *http.Request) {
	var sender message.EmailSender

	var emailTable string = "登录验证码"
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
			您本次登录的验证码是：<b>{{mailcode}}</b><br><br>
			请勿将验证码透露给其他人。<br>
			如非本人操作，请联系管理员修改密码。<br><br>
			本邮件由系统自动发送，请勿直接回复！<br>
			感谢您的访问，祝您使用愉快！
		</div>
	</div>
</div>
</body>
</html>`

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_mail_code] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_mail_code] client [%s:%s]", ip, port)

	// 解析 URL 参数
	query := r.URL.Query()
	userId := query.Get("user_id")
	global.Log.Debugf("[get_mail_code] user_id:[%s]", userId)

	// 判断参数是否为空
	if userId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_mail_code] 用户不存在")
		responseError := ResponseError{
			Status:  false,
			Message: "用户不存在",
			Error:   3201,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	user := database.User{}
	// 初始化数据库
	user.CreateUser(global.GlobalDB)

	user.UserID.String = userId

	err = user.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[get_mail_code] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   3202,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 判断邮箱是否正确
	if user.UserEmail.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_mail_code] 用户邮箱未设置")
		responseError := ResponseError{
			Status:  false,
			Message: "用户邮箱未设置",
			Error:   3203,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	valid := global.IsValidEmail(user.UserEmail.String)
	global.Log.Debugf("邮箱: %-25s 格式: %t", user.UserEmail.String, valid)

	if !valid {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_mail_code] 用户邮箱格式错误")
		responseError := ResponseError{
			Status:  false,
			Message: "用户邮箱格式错误",
			Error:   3204,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	sender = message.EmailSender{
		Host:     global.GlobalJWireGuardini.EmailHost,
		Port:     global.GlobalJWireGuardini.EmailPort,
		Username: global.GlobalJWireGuardini.EmailUser,
		Password: global.GlobalJWireGuardini.EmailPass,
		From:     global.GlobalJWireGuardini.FormEmail,
		Name:     global.GlobalJWireGuardini.FormName,
	}

	// 获取 6位随机验证码
	emailCode, err := global.Random6DigitString()
	if err != nil {
		global.Log.Errorf("[get_mail_code] 获取邮箱验证码错误：%+v", err)
	}

	// html 拼接
	doc, _ := html.Parse(strings.NewReader(htmlTemplate))
	global.ReplaceText(doc, "{{mailcode}}", emailCode)
	var b strings.Builder
	html.Render(&b, doc)

	err = sender.SendMail(
		[]string{user.UserEmail.String},
		emailTable,
		b.String(),
		true, // 使用 HTML 格式
	)

	if err != nil {
		global.Log.Errorf("[get_mail_code] 邮件发送失败：%+v", err)
	} else {
		global.Log.Debugln("[get_mail_code] 邮件发送成功！")
	}

	// 更新数据库
	user.MailCode.String = emailCode
	user.MailTime.Int64 = time.Now().Unix() + 300

	err = user.UpdateUsers(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_mail_code] 邮件发送成功，但无法将邮件验证码更新到数据库中：%+v", err)
	}

	// 返回 JSON 响应
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "邮件发送成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)

}
