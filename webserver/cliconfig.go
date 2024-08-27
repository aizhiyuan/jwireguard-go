// webservice/cliconfig.go
package webservice

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net/http"
	"os/exec"
	"time"
)

type ResponseSubNetworkList struct {
	Status  bool              `json:"status"`
	Message string            `json:"message"`
	Data    []database.Subnet `json:"data"`
}

type ResponseCliConfig struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Address string `json:"address"`
	MD5     string `json:"md5"`
	Data    string `json:"data"`
}

type ResponseCliList struct {
	Status  bool                 `json:"status"`
	Message string               `json:"message"`
	Data    []database.CliConfig `json:"data"`
}

type ResponseCliInfo struct {
	Status  bool               `json:"status"`
	Message string             `json:"message"`
	Data    database.CliConfig `json:"data"`
}

type PostCliConfig struct {
	CliID   string `json:"cli_id"`
	CliName string `json:"cli_name"`
	SerName string `json:"ser_name"`
	CliSN   string `json:"cli_sn"`
	CliKey  string `json:"cli_key"`
}

type PostDelCliConfig struct {
	CliID string `json:"cli_id"`
	SerID string `json:"ser_id"`
}

type PostUpdateCliAddr struct {
	CliID string `json:"cli_id"`
	SerID string `json:"ser_id"`
}

type PostCliAddr struct {
	CliID   string `json:"cli_id"`
	Address string `json:"address"`
}

func registerCliRoutes() {
	http.HandleFunc("/get_sub_network_list", GetSubNetworkList)
	http.HandleFunc("/get_cli_config", GetCliConfig)
	http.HandleFunc("/get_cli_list", GetCliList)
	http.HandleFunc("/get_cli_info", GetCliInfo)
	http.HandleFunc("/add_cli_config", AddCLiConfig)
	http.HandleFunc("/del_cli_config", DelCliConfig)
	http.HandleFunc("/update_cli_config", UpdateCliConfig)
	http.HandleFunc("/update_cli_info", UpdateCliInfo)
	http.HandleFunc("/update_cli_addr", UpdateCliAddr)
	http.HandleFunc("/update_subnet_cli_addr", UpdataSubnetCliAddr)
}

func GetSubNetworkList(w http.ResponseWriter, r *http.Request) {

	// 解析 URL 参数
	query := r.URL.Query()
	userId := query.Get("user_id")

	// 判断参数是否为空
	if userId == "" {
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

	var user = database.User{}
	var subnet = database.Subnet{}

	// 初始化
	user.CreateUser(global.GlobalDB)
	subnet.CreateSubnet(global.GlobalDB)

	// 遍历用户表
	userIds, err := user.QueryUserIds(global.GlobalDB, userId)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法获取用户:", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(userIds) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户列表为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 遍历子网表
	serIds, err := user.GetSubnetIdsByUserIds(global.GlobalDB, userIds)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法获取子网序号组:", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(serIds) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网列表为空",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取子网列表
	subnets, err := subnet.GetSubnetBySerIDs(global.GlobalDB, serIds)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法获取子网列表:", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(subnets) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网列表为空",
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 设置返回结构体
	responseSubNetworkList := ResponseSubNetworkList{
		Status:  true,
		Message: "获取子网列表成功",
		Data:    subnets,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseSubNetworkList)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法将JSON对象转为字符串:", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 设置响应头，指明内容类型为 JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}

func GetCliConfig(w http.ResponseWriter, r *http.Request) {
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")

	// 判断参数是否为空
	if cliId == "" {
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

	scriptPath := fmt.Sprintf("%s/server/easy-rsa/client.sh", global.GlobalJWireGuardini.OpenVpnPath)

	cmd := exec.Command("/bin/bash", scriptPath, cliId)

	// 执行SHELL命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[GetCliConfig] 无法执行 [%s %s] 错误: %v 执行输出: %s", scriptPath, cliId, err, output)
		responseError := ResponseError{
			Status:  false,
			Message: "无法执行client.sh脚本!",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 创建用户配置
	var cliConfig = database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 获取配置
	cliConfig.CliID = cliId
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		log.Fatalln("[GetCliConfig] 获取客户端配置失败 ", err)
		responseError := ResponseError{
			Status:  false,
			Message: "获取客户端配置失败!",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 检查配置文件是否存在
	cliConfigPath := fmt.Sprintf("%s/client/%s.ovpn", global.GlobalJWireGuardini.OpenVpnPath, cliId)
	if !global.CheckFileExists(cliConfigPath) {
		log.Fatalf("[GetCliConfig] 获取客户端配置不存在")
		responseError := ResponseError{
			Status:  false,
			Message: "客户端配置不存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfigText, err := ioutil.ReadFile(cliConfigPath)
	if err != nil {
		log.Fatalf("客户端配置读取失败: %v", err)

		responseError := ResponseError{
			Status:  false,
			Message: "客户端配置读取失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfigByte := []byte(cliConfigText)
	cliConfigBase64 := base64.StdEncoding.EncodeToString(cliConfigByte)
	cliConfigMd5 := global.GenerateMD5(cliConfigBase64)

	responseCliConfig := ResponseCliConfig{
		Status:  true,
		Message: "获取客户端配置成功!",
		Address: cliConfig.CliAddress,
		Data:    cliConfigBase64,
		MD5:     cliConfigMd5,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliConfig)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法将JSON对象转为字符串:", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 设置响应头，指明内容类型为 JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func GetCliList(w http.ResponseWriter, r *http.Request) {
	// 解析 URL 参数
	query := r.URL.Query()
	serId := query.Get("ser_id")

	// 判断参数是否为空
	if serId == "" {
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

	// 创建用户配置
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 获取用户配置
	cliConfig.SerID = serId
	cliConfigs, err := cliConfig.GetCliConfigBySerID(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "获取客户端列表失败!",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(cliConfigs) <= 0 {
		responseError := ResponseError{
			Status:  false,
			Message: "客户端列表为空!",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	responseCliList := ResponseCliList{
		Status:  true,
		Message: "获取客户端列表成功!",
		Data:    cliConfigs,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliList)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法将JSON对象转为字符串:", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 设置响应头，指明内容类型为 JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func GetCliInfo(w http.ResponseWriter, r *http.Request) {
	// 解析 URL 参数

	query := r.URL.Query()
	cliId := query.Get("cli_id")

	// 判断参数是否为空
	if cliId == "" {
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

	// 创建用户配置
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 获取用户配置
	cliConfig.CliID = cliId

	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "获取客户端信息失败!",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	responseCliInfo := ResponseCliInfo{
		Status:  true,
		Message: "获取客户端列表成功!",
		Data:    cliConfig,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliInfo)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintln("无法将JSON对象转为字符串:", err),
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 设置响应头，指明内容类型为 JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func AddCLiConfig(w http.ResponseWriter, r *http.Request) {

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
	portCliConfig := PostCliConfig{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portCliConfig); err != nil {
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

	// 判断参数是否为空
	if portCliConfig.CliID == "" ||
		portCliConfig.SerName == "" ||
		portCliConfig.CliName == "" ||
		portCliConfig.CliSN == "" ||
		portCliConfig.CliKey == "" {
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

	// 计算SerName SHA3
	serNameSHA3 := global.GenerateMD5(portCliConfig.SerName)
	// 计算CliKeyString
	cliKeyString := fmt.Sprintf("%s%s", serNameSHA3, portCliConfig.CliSN)
	cliKeyValue := global.GenerateMD5(cliKeyString)
	if cliKeyValue != portCliConfig.CliKey {
		responseError := ResponseError{
			Status:  false,
			Message: "KEY值校验错误",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据库对象
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}

	// 数据库初始化
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 查看客户端是否存在
	cliConfig.CliID = portCliConfig.CliID
	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "客户端已存在",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询子网是否存在
	subnet.SerID = serNameSHA3
	err = subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		//获取新的网段
		newSubNum, err := subnet.GetNewSubnetNumber(global.GlobalDB)
		if err != nil {
			responseError := ResponseError{
				Status:  false,
				Message: "子网网段已满",
				Error:   6,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		subnet.SerNum = newSubNum
		subnet.CliNum = 1
		subnet.SerName = portCliConfig.SerName
		// subnet.UserID = global.GlobalDefaultUserMd5
	}

	// 查看客户端在数据中是否存在
	cliConfig.CliID = portCliConfig.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		responseError := ResponseError{
			Status:  false,
			Message: "客户端已存在",
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d", global.GlobalJWireGuardini.IPPrefix, subnet.SerNum)
	ccdPath := fmt.Sprintf("%s/ccd", global.GlobalJWireGuardini.OpenVpnPath)
	cliAddress, err := FindUnusedIP(ccdPath, ipPrefix)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "无法获取到当前可用的客户端IP",
			Error:   8,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// fmt.Println("cliAddress：", cliAddress)
	cliConfig.CliAddress = cliAddress

	ovpnClient := fmt.Sprintf("%s/client/%s.ovpn", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	ccdClient := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	privateClient := fmt.Sprintf("%s/server/easy-rsa/pki/private/%s.key", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	reqsClient := fmt.Sprintf("%s/server/easy-rsa/pki/reqs/%s.req", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	issuedClient := fmt.Sprintf("%s/server/easy-rsa/issued/private/%s.crt", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("ccdClient: ", ccdClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 添加客户端
	clientAddSh := fmt.Sprintf("%s/server/easy-rsa/add.sh", global.GlobalJWireGuardini.OpenVpnPath)
	// log.Println("clientAddSh: ", clientAddSh)
	cmd := exec.Command("/bin/bash", clientAddSh, portCliConfig.CliID, cliAddress)

	// 执行SHELL命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[GetCliConfig] 无法执行 [%s %s %s] 错误: %v 执行输出: %s", clientAddSh, portCliConfig.CliID, cliAddress, err, output)
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

	cliConfig.CliName = portCliConfig.CliName
	cliConfig.CliMapping = ""
	cliConfig.CliStatus = "false"
	cliConfig.EditStatus = "false"
	cliConfig.CliSN = portCliConfig.CliSN
	cliConfig.SerID = serNameSHA3
	cliConfig.SerName = portCliConfig.SerName
	cliConfig.Timestamp = time.Now().Unix()

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

	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端创建成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdateCliConfig(w http.ResponseWriter, r *http.Request) {
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
	portCliConfig := PostCliConfig{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portCliConfig); err != nil {
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

	// 判断参数是否为空
	if portCliConfig.CliID == "" ||
		portCliConfig.SerName == "" ||
		portCliConfig.CliName == "" ||
		portCliConfig.CliSN == "" ||
		portCliConfig.CliKey == "" {
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

	// 创建客户端对象
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 判断是否有客户端
	// 查看客户端在数据中是否存在
	cliConfig.CliID = portCliConfig.CliID
	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "客户端不存在",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 计算SerName MD5
	serNameSHA3 := global.GenerateMD5(portCliConfig.SerName)
	// 计算CliKeyString
	cliKeyString := fmt.Sprintf("%s%s", serNameSHA3, portCliConfig.CliSN)
	cliKeyValue := global.GenerateMD5(cliKeyString)
	if cliKeyValue != portCliConfig.CliKey {
		responseError := ResponseError{
			Status:  false,
			Message: "KEY值校验错误",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 检查配置是否存在
	ovpnClient := fmt.Sprintf("%s/client/%s.ovpn", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	ccdClient := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	privateClient := fmt.Sprintf("%s/server/easy-rsa/pki/private/%s.key", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	reqsClient := fmt.Sprintf("%s/server/easy-rsa/pki/reqs/%s.req", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	issuedClient := fmt.Sprintf("%s/server/easy-rsa/issued/private/%s.crt", global.GlobalJWireGuardini.OpenVpnPath, portCliConfig.CliID)
	// log.Println("ovpnClient: ", ovpnClient)
	// log.Println("ccdClient: ", ccdClient)
	// log.Println("privateClient: ", privateClient)
	// log.Println("reqsClient: ", reqsClient)
	// log.Println("issuedClient: ", issuedClient)

	// 更新客户端
	clientAddSh := fmt.Sprintf("%s/server/easy-rsa/update.sh", global.GlobalJWireGuardini.OpenVpnPath)
	// log.Println("clientAddSh: ", clientAddSh)
	cmd := exec.Command("/bin/bash", clientAddSh, portCliConfig.CliID, cliConfig.CliAddress)

	// 执行SHELL命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[GetCliConfig] 无法执行 [%s %s %s] 错误: %v 执行输出: %s", clientAddSh, portCliConfig.CliID, cliConfig.CliAddress, err, output)
		responseError := ResponseError{
			Status:  false,
			Message: "无法执行update.sh脚本!",
			Error:   8,
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
			Message: "客户端更新失败!",
			Error:   9,
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
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端更新成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func DelCliConfig(w http.ResponseWriter, r *http.Request) {
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
	portDelCliConfig := PostDelCliConfig{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portDelCliConfig); err != nil {
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

	// 判断参数是否为空
	if portDelCliConfig.CliID == "" ||
		portDelCliConfig.SerID == "" {
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

	// 删除客户端
	clientDelSh := fmt.Sprintf("%s/server/easy-rsa/del.sh", global.GlobalJWireGuardini.OpenVpnPath)
	// log.Println("clientAddSh: ", clientAddSh)
	cmd := exec.Command("/bin/bash", clientDelSh, portDelCliConfig.CliID)

	// 执行SHELL命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("[GetCliConfig] 无法执行 [%s %s] 错误: %v 执行输出: %s", clientDelSh, portDelCliConfig.CliID, err, output)
		responseError := ResponseError{
			Status:  false,
			Message: "无法执行del.sh脚本!",
			Error:   8,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// log.Printf("Output:\n%s", output)

	// 创建数据库对象
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}

	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 删除客户端
	cliConfig.CliID = portDelCliConfig.CliID
	err = cliConfig.DeleteCliConfig(global.GlobalDB)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "删除客户端失败",
			Error:   9,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询客户端
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		responseError := ResponseError{
			Status:  false,
			Message: "删除客户端失败",
			Error:   10,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdateCliInfo(w http.ResponseWriter, r *http.Request) {
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

	// 创建数据对象
	postClientInfo := database.CliConfig{}
	// 初始化数据库
	postClientInfo.CreateCliConfig(global.GlobalDB)

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &postClientInfo); err != nil {
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

	if postClientInfo.CliID == "" {
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

	// 备份数据
	postClientInfoBak := postClientInfo
	// 查看客户端是否存在
	err := postClientInfoBak.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "客户端不存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新数据
	err = postClientInfo.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "修改客户端信息失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 如果有IP地址更改则修改IP地址
	if postClientInfo.CliAddress != "" {
		changClientFile := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, postClientInfo.CliID)
		changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", postClientInfo.CliAddress)
		err := global.WriteToFile(changClientFile, changClientAddr)
		if err != nil {
			// 如果参数为空，返回 JSON 错误响应
			responseError := ResponseError{
				Status:  false,
				Message: "修改客户端IP地址失败!",
				Error:   6,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端修改成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)

}

func UpdateCliAddr(w http.ResponseWriter, r *http.Request) {
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
	postClientAddress := PostCliAddr{}
	clientConfig := database.CliConfig{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &postClientAddress); err != nil {
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

	if postClientAddress.CliID == "" || postClientAddress.Address == "" {
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

	// 查看客户端是否存在
	clientConfig.CliID = postClientAddress.CliID
	err := clientConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "客户端不存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新客户端IP地址
	clientConfig.CliAddress = postClientAddress.Address
	// 更新数据
	err = clientConfig.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "在数据库中修改客户端IP地址失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, postClientAddress.CliID)
	changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", postClientAddress.Address)
	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "在文件中修改客户端IP地址失败!",
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端IP修改成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdataSubnetCliAddr(w http.ResponseWriter, r *http.Request) {

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
	portUpdateClientAddress := PostUpdateCliAddr{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portUpdateClientAddress); err != nil {
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

	if portUpdateClientAddress.CliID == "" || portUpdateClientAddress.SerID == "" {
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
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}
	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 获取子网网段
	subnet.SerID = portUpdateClientAddress.SerID
	err := subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网不存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查看客户端是否存在
	cliConfig.CliID = portUpdateClientAddress.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "客户端不存在!",
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d", global.GlobalJWireGuardini.IPPrefix, subnet.SerNum)
	ccdPath := fmt.Sprintf("%s/ccd", global.GlobalJWireGuardini.OpenVpnPath)
	cliAddress, err := FindUnusedIP(ccdPath, ipPrefix)
	if err != nil {
		responseError := ResponseError{
			Status:  false,
			Message: "无法获取到当前可用的客户端IP",
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新数据
	cliConfig.CliAddress = cliAddress
	err = cliConfig.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "在数据库中修改客户端IP地址失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/ccd/%s", global.GlobalJWireGuardini.OpenVpnPath, portUpdateClientAddress.CliID)
	changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", cliAddress)
	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "在文件中修改客户端IP地址失败!",
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "更新客户端IP成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
