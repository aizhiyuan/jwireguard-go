// webservice/user.go
package webservice

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net/http"
	"os/exec"
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
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
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
		responseError := ResponseError{
			Status:  false,
			Message: "解析JSON请求参数错误",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portCheckUserLogin.UserName == "" ||
		portCheckUserLogin.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空!",
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
		responseError := ResponseError{
			Status:  false,
			Message: "密码错误!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 账号认证
	dbuser.UserName = portCheckUserLogin.UserName
	dbuser.UserPasswd = encryptedPasswd

	loginStatus, err := dbuser.CheckLogin(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "密码错误!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if !loginStatus {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "密码错误!",
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
		UserID:  dbuser.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseCheckLogin)
}

// 添加用户
func AddUser(w http.ResponseWriter, r *http.Request) {
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
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
	portUser := database.User{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		responseError := ResponseError{
			Status:  false,
			Message: "解析JSON请求参数错误",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portUser.UserName == "" ||
		portUser.SerID == "" ||
		portUser.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空!",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 生成用户ID
	if portUser.UserID == "" {
		portUser.UserID = global.GenerateMD5(portUser.UserName)
	}
	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	// 判断用户是否存在
	err := portUser.GetUserByID(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户已存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 对密码进行加密
	encryptedPasswd, err := global.Encrypt(portUser.UserPasswd, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "密码错误!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加用户
	portUser.UserPasswd = encryptedPasswd
	err = portUser.InsertUser(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "添加用户失败!",
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

	// 添加客户端
	cliConfig.CliID = portUser.UserID
	cliConfig.CliAddress = fmt.Sprintf("%s.0.1", global.GlobalJWireGuardini.IPPrefix)
	ovpnClient := fmt.Sprintf("%s/client/%s.ovpn", global.GlobalJWireGuardini.OpenVpnPath, cliConfig.CliID)
	ccdClient := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, cliConfig.CliID)
	privateClient := fmt.Sprintf("%s/server/easy-rsa/pki/private/%s.key", global.GlobalJWireGuardini.OpenVpnPath, cliConfig.CliID)
	reqsClient := fmt.Sprintf("%s/server/easy-rsa/pki/reqs/%s.req", global.GlobalJWireGuardini.OpenVpnPath, cliConfig.CliID)
	issuedClient := fmt.Sprintf("%s/server/easy-rsa/issued/private/%s.crt", global.GlobalJWireGuardini.OpenVpnPath, cliConfig.CliID)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("ccdClient: ", ccdClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 添加客户端
	clientAddSh := fmt.Sprintf("%s/server/easy-rsa/add.sh", global.GlobalJWireGuardini.OpenVpnPath)
	// log.Println("clientAddSh: ", clientAddSh)
	cmd := exec.Command("/bin/bash", clientAddSh, cliConfig.CliID, cliConfig.CliAddress)

	// 执行SHELL命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[GetCliConfig] 无法执行 [%s %s %s] 错误: %v 执行输出: %s", clientAddSh, cliConfig.CliID, cliConfig.CliAddress, err, output)
		responseError := ResponseError{
			Status:  false,
			Message: "无法执行add.sh脚本!",
			Error:   9,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// log.Printf("Output:\n%s", output)
	if (!global.CheckFileExists(privateClient)) ||
		(!global.CheckFileExists(reqsClient)) ||
		(!global.CheckFileExists(issuedClient)) {
		responseError := ResponseError{
			Status:  false,
			Message: "文件中创建客户端失败!",
			Error:   10,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)

		// 删除错误客户端文件
		global.DeleteFileIfExists(ovpnClient)
		global.DeleteFileIfExists(ccdClient)
		global.DeleteFileIfExists(privateClient)
		global.DeleteFileIfExists(reqsClient)
		global.DeleteFileIfExists(issuedClient)

		return
	}

	cliConfig.CliName = portUser.UserName
	cliConfig.CliMapping = ""
	cliConfig.CliStatus = "false"
	cliConfig.EditStatus = "false"
	cliConfig.CliSN = ""
	cliConfig.SerID = ""
	cliConfig.SerName = ""
	cliConfig.Timestamp = 0

	err = cliConfig.InsertCliConfig(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "数据库创建客户端失败!",
			Error:   10,
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
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
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
	portUser := database.User{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		responseError := ResponseError{
			Status:  false,
			Message: "解析JSON请求参数错误",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portUser.UserID == "" ||
		portUser.UserName == "" ||
		portUser.UserPasswd == "" {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空!",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 初始化数据库
	portUser.CreateUser(global.GlobalDB)

	// 判断用户是否存在
	err := portUser.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户不存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 对密码进行加密
	encryptedPasswd, err := global.Encrypt(portUser.UserPasswd, global.GlobalEncryptKey)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "密码错误!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加用户
	portUser.UserPasswd = encryptedPasswd
	err = portUser.UpdateUsers(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户修改失败!",
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
	// 解析 URL 参数
	query := r.URL.Query()
	userID := query.Get("user_id")

	// 判断参数是否为空
	if userID == "" {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空!",
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
	user.UserID = userID

	err := user.GetUserByID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户不存在!",
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
		responseError := ResponseError{
			Status:  false,
			Message: "用户删除失败!",
			Error:   3,
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
