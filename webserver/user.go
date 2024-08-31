// webservice/user.go
package webservice

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net/http"
)

// 定义一个结构体，用于存储用户登录信息
type PostCheckUserLogin struct {
	UserName   string `json:"user_name"`
	UserPasswd string `json:"user_passwd"`
}

// 定义一个结构体，用于存储登录结果
type ResponseCheckLogin struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// 注册用户路由
func registerUserRoutes() {
	http.HandleFunc("/check_users_login", CheckUsersLogin)
	http.HandleFunc("/add_user", AddUser)
	http.HandleFunc("/edit_user", EditUser)
	http.HandleFunc("/del_user", DelUser)
}

// 用户登录验证
func CheckUsersLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("[CheckUsersLogin] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[CheckUsersLogin] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1,
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
		log.Printf("[CheckUsersLogin] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portCheckUserLogin.UserName == "" ||
		portCheckUserLogin.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[CheckUsersLogin] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	dbuser := database.User{}
	// 初始化
	dbuser.CreateUser(global.GlobalDB)
	// 对密码进行加密
	encryptedPasswd, err := global.Encrypt(portCheckUserLogin.UserPasswd, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[CheckUsersLogin] 密码错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("密码错误, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 账号认证
	dbuser.UserName.String = portCheckUserLogin.UserName
	dbuser.UserPasswd.String = encryptedPasswd

	loginStatus, err := dbuser.CheckLogin(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[CheckUsersLogin] 密码错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("密码错误, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if !loginStatus {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[CheckUsersLogin] 密码错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("密码错误, err:%v", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseCheckLogin := ResponseCheckLogin{
		Status:  true,
		Message: "登录成功!",
		UserID:  dbuser.UserID.String,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseCheckLogin)
}

// 添加用户
func AddUser(w http.ResponseWriter, r *http.Request) {
	log.Println("[AddUser] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[AddUser] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	exporteUser := database.ExportedUser{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &exporteUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[AddUser] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据
	portUser := exporteUser.ConvertToUser()

	if portUser.UserName.String == "" ||
		portUser.SerID.String == "" ||
		portUser.UserPasswd.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[AddUser] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 生成用户ID
	if portUser.UserID.String == "" {
		portUser.UserID.String = global.GenerateMD5(portUser.UserName.String)
	}

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	// 判断用户是否存在
	err := portUser.GetUserByID(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[AddUser] 用户已存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户已存在, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 对密码进行加密
	encryptedPasswd, err := global.Encrypt(portUser.UserPasswd.String, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[AddUser] 密码错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("密码错误, err:%v", err),
			Error:   5,
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
		log.Printf("[AddUser] 添加用户失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("添加用户失败, err:%v", err),
			Error:   6,
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
		log.Printf("[AddUser] 添加用户配置失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("添加用户配置失败, err:%v", err),
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfig.CliName = portUser.UserName
	cliConfig.CliMapping.String = ""
	cliConfig.CliStatus.String = "false"
	cliConfig.EditStatus.String = "false"
	cliConfig.CliSN.String = ""
	cliConfig.SerID.String = ""
	cliConfig.SerName.String = ""
	cliConfig.Timestamp.Int64 = 0
	cliConfig.OnlineStatus.String = "true"

	err = cliConfig.InsertCliConfig(global.GlobalDB)
	if err != nil {
		log.Printf("[AddUser] 数据库创建客户端失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库创建客户端失败, err:%v", err),
			Error:   8,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
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
	log.Println("[EditUser] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[EditUser] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据

	exporteUser := database.ExportedUser{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &exporteUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[EditUser] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	portUser := exporteUser.ConvertToUser()

	if portUser.UserID.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[EditUser] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	portUserbak := portUser

	// 判断用户是否存在
	err := portUserbak.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[EditUser] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   4,
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
			log.Printf("[EditUser] 密码错误, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("密码错误, err:%v", err),
				Error:   5,
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
		log.Printf("[EditUser] 用户修改失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户修改失败, err:%v", err),
			Error:   6,
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
	log.Println("[DelUser] start")
	// 解析 URL 参数
	query := r.URL.Query()
	userID := query.Get("user_id")

	// 判断参数是否为空
	if userID == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[DelUser] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	user := database.User{}

	// 初始化数据库
	user.CreateUser(global.GlobalDB)

	// 查看子网是否存在
	user.UserID.String = userID

	err := user.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[DelUser] 用户不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户不存在, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除子网
	err = user.DeleteUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[DelUser] 用户删除失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("用户删除失败, err:%v", err),
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 执行SHELL命令
	err = global.ShellDelClient(userID)
	if err != nil {
		log.Printf("[DelUser] 无法删除客户端, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法删除客户端, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "用户删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
