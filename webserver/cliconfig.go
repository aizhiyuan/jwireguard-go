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
	"time"
)

type ResponseSubNetworkList struct {
	Status  bool                      `json:"status"`
	Message string                    `json:"message"`
	Data    []database.ExportedSubnet `json:"data"`
}

type ResponseCliConfig struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Address string `json:"address"`
	MD5     string `json:"md5"`
	Data    string `json:"data"`
}

type ResponseCliList struct {
	Status  bool                         `json:"status"`
	Message string                       `json:"message"`
	Data    []database.ExportedCliConfig `json:"data"`
}

type ResponseCliInfo struct {
	Status  bool                       `json:"status"`
	Message string                     `json:"message"`
	Data    database.ExportedCliConfig `json:"data"`
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
	log.Println("[GetSubNetworkList] start")
	// 解析 URL 参数
	query := r.URL.Query()
	userId := query.Get("user_id")

	// 判断参数是否为空
	if userId == "" {
		log.Println("[GetSubNetworkList] 参数为空")
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
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
		log.Printf("[GetSubNetworkList] 无法获取用户, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取用户, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(userIds) <= 0 {
		log.Println("[GetSubNetworkList] 用户列表为空")
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
	// fmt.Println("Ids:", userIds)
	// 遍历子网表
	serIds, err := user.GetSubnetIdsByUserIds(global.GlobalDB, userIds)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[GetSubNetworkList] 无法获取子网序号组, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取子网序号组, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(serIds) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[GetSubNetworkList] 子网列表为空")
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
		log.Printf("[GetSubNetworkList] 无法获取子网列表, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取子网列表, err:%v", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(subnets) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[GetSubNetworkList] 子网列表为空")
		responseError := ResponseError{
			Status:  false,
			Message: "子网列表为空",
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	exportedSubnets := database.ConvertSubnets(subnets)

	// 设置返回结构体
	responseSubNetworkList := ResponseSubNetworkList{
		Status:  true,
		Message: "获取子网列表成功",
		Data:    exportedSubnets,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseSubNetworkList)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[GetSubNetworkList] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
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
	log.Println("[GetCliConfig] start")
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")

	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[GetCliConfig] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	headClient := fmt.Sprintf("%s/openvpn.txt", global.GlobalOpenVPNPath.ConfigPath)
	caClient := fmt.Sprintf("%s/ca.crt", global.GlobalOpenVPNPath.PkiPath)
	taClient := fmt.Sprintf("%s/ta.key", global.GlobalOpenVPNPath.PkiPath)
	ovpnClient := fmt.Sprintf("%s/%s.ovpn", global.GlobalOpenVPNPath.ConfigPath, cliId)
	privateClient := fmt.Sprintf("%s/%s.key", global.GlobalOpenVPNPath.PrivatePath, cliId)
	issuedClient := fmt.Sprintf("%s/%s.crt", global.GlobalOpenVPNPath.IssuedPath, cliId)

	// Define file paths
	files := map[string]string{
		"key":      privateClient,
		"cert":     issuedClient,
		"ca":       caClient,
		"tls-auth": taClient,
	}

	for _, file := range files {
		if !global.CheckFileExists(file) {
			log.Printf("[GetCliConfig] 该%s文件不存在", file)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("该%s文件不存在", file),
				Error:   2,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// Create the .ovpn file

	err := global.CreateOVPNFile(headClient, ovpnClient, files)
	if err != nil {
		log.Printf("[GetCliConfig] 无法合成%s.ovpn, err:%v", cliId, err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法合成%s.ovpn, err:%v", cliId, err),
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
	cliConfig.CliID.String = cliId
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		log.Printf("[GetCliConfig] 获取客户端配置失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取客户端配置失败, err:%v", err),
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// 检查配置文件是否存在
	if !global.CheckFileExists(ovpnClient) {
		log.Println("[GetCliConfig] 客户端配置不存在")
		responseError := ResponseError{
			Status:  false,
			Message: "客户端配置不存在",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfigText, err := ioutil.ReadFile(ovpnClient)
	if err != nil {
		log.Printf("[GetCliConfig] 客户端配置读取失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端配置读取失败, err:%v", err),
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
		Address: cliConfig.CliAddress.String,
		Data:    cliConfigBase64,
		MD5:     cliConfigMd5,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliConfig)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[GetCliConfig] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
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
	log.Println("[GetCliList] start")
	// 解析 URL 参数
	query := r.URL.Query()
	serId := query.Get("ser_id")

	// 判断参数是否为空
	if serId == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[GetCliList] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
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
	cliConfig.SerID.String = serId
	cliConfigs, err := cliConfig.GetCliConfigBySerID(global.GlobalDB)
	if err != nil {
		log.Printf("[GetCliList] 获取客户端列表失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取客户端列表失败, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	exportedCliConfigs := database.ConvertCliConfigs(cliConfigs)

	if len(cliConfigs) <= 0 {
		log.Println("[GetCliList] 客户端列表为空")
		responseError := ResponseError{
			Status:  false,
			Message: "客户端列表为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	responseCliList := ResponseCliList{
		Status:  true,
		Message: "获取客户端列表成功!",
		Data:    exportedCliConfigs,
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliList)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[GetCliList] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
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
	log.Println("[GetCliInfo] start")
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")

	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[GetCliList] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
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
	cliConfig.CliID.String = cliId

	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		log.Printf("[GetCliList] 获取客户端信息失败!, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取客户端信息失败!, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	responseCliInfo := ResponseCliInfo{
		Status:  true,
		Message: "获取客户端列表成功!",
		Data:    cliConfig.ToExported(),
	}

	// 将JSON对象转为字符串
	jsonData, err := json.Marshal(responseCliInfo)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[GetCliList] 无法将JSON对象转为字符串:", err)
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
	log.Println("[AddCLiConfig] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[AddCLiConfig] 请求类型不是Post")
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
		log.Printf("[AddCLiConfig] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
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
		log.Println("[AddCLiConfig] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
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
	log.Println("[AddCLiConfig] KEY值校验错误")
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
	cliConfig.CliID.String = portCliConfig.CliID
	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		log.Printf("[AddCLiConfig] 客户端已存在!, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端已存在!, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询子网是否存在
	subnet.SerID.String = serNameSHA3
	err = subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		//获取新的网段
		newSubNum, err := subnet.GetNewSubnetNumber(global.GlobalDB)
		if err != nil {
			log.Printf("[AddCLiConfig] 子网网段已满, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("子网网段已满, err:%v", err),
				Error:   6,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		subnet.SerNum.Int32 = newSubNum
		subnet.CliNum.Int32 = 1
		subnet.SerName.String = portCliConfig.SerName
		err = subnet.InsertSubnet(global.GlobalDB)
		if err != nil {
			log.Printf("[AddCLiConfig] 子网添加失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("子网添加失败, err:%v", err),
				Error:   7,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
		// subnet.UserID = global.GlobalDefaultUserMd5
	}

	// 查看客户端在数据中是否存在
	cliConfig.CliID.String = portCliConfig.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		log.Printf("[AddCLiConfig] 客户端已存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端已存在, err:%v", err),
			Error:   8,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d", global.GlobalJWireGuardini.IPPrefix, subnet.SerNum.Int32)
	cliAddress, err := FindUnusedIP(global.GlobalOpenVPNPath.CcdPath, ipPrefix)
	if err != nil {
		log.Printf("[AddCLiConfig] 无法获取到当前可用的客户端IP, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取到当前可用的客户端IP, err:%v", err),
			Error:   9,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	// fmt.Println("cliAddress：", cliAddress)
	cliConfig.CliAddress.String = cliAddress

	// 添加客户端
	err = global.ShellAddClient(portCliConfig.CliID, cliAddress)
	if err != nil {
		log.Printf("[AddCLiConfig] 无法添加客户端, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法添加客户端, err:%v", err),
			Error:   10,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfig.CliName.String = portCliConfig.CliName
	cliConfig.CliMapping.String = ""
	cliConfig.CliStatus.String = "false"
	cliConfig.EditStatus.String = "false"
	cliConfig.CliSN.String = portCliConfig.CliSN
	cliConfig.SerID.String = serNameSHA3
	cliConfig.SerName.String = portCliConfig.SerName
	cliConfig.Timestamp.Int64 = time.Now().Unix()
	cliConfig.OnlineStatus.String = "true"

	err = cliConfig.InsertCliConfig(global.GlobalDB)
	if err != nil {
		log.Printf("[AddCLiConfig] 数据库创建客户端失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库创建客户端失败, err:%v", err),
			Error:   17,
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
	log.Println("[UpdateCliConfig] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		log.Println("[UpdateCliConfig] 请求类型不是Post")
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
		log.Printf("[UpdateCliConfig] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
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
		log.Println("[UpdateCliConfig] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
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
	cliConfig.CliID.String = portCliConfig.CliID
	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		log.Printf("[UpdateCliConfig] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
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
		log.Println("[UpdateCliConfig] KEY值校验错误")
		responseError := ResponseError{
			Status:  false,
			Message: "KEY值校验错误",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	err = global.ShellUpdateClient(portCliConfig.CliID)
	if err != nil {
		log.Printf("[UpdateCliConfig] 客户端更新失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端更新失败, err:%v", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
	}

	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端更新成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func DelCliConfig(w http.ResponseWriter, r *http.Request) {
	log.Println("[DelCliConfig] start")
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")

	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[DelCliConfig] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1,
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

	// 查询客户端
	cliConfig.CliID.String = cliId
	err := cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		log.Printf("[DelCliConfig] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除客户端

	err = cliConfig.DeleteCliConfig(global.GlobalDB)
	if err != nil {
		log.Printf("[DelCliConfig] 删除客户端失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("删除客户端失败, err:%v", err),
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询客户端
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		log.Printf("[DelCliConfig] 客户端删除失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端删除失败, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 执行SHELL命令
	err = global.ShellDelClient(cliId)
	if err != nil {
		log.Printf("[DelCliConfig] 无法删除客户端, err:%v", err)
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
		Message: "客户端删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdateCliInfo(w http.ResponseWriter, r *http.Request) {
	log.Println("[UpdateCliInfo] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdateCliInfo] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	//
	postClientInfoRecv := database.ExportedCliConfig{}
	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &postClientInfoRecv); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[UpdateCliInfo] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建数据对象
	postClientInfo := postClientInfoRecv.ConvertToCliConfig()
	// 初始化数据库.
	postClientInfo.CreateCliConfig(global.GlobalDB)

	if postClientInfo.CliID.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdateCliInfo] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 备份数据
	postClientInfoBak := postClientInfoRecv.ConvertToCliConfig()
	// 查看客户端是否存在
	err := postClientInfoBak.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdateCliInfo] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
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
		log.Printf("[UpdateCliInfo] 修改客户端信息失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("修改客户端信息失败, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 如果有IP地址更改则修改IP地址
	if postClientInfo.CliAddress.String != "" {
		changClientFile := fmt.Sprintf("%s/%s",
			global.GlobalOpenVPNPath.CcdPath,
			postClientInfo.CliID.String)

		changClientAddr := fmt.Sprintf("ifconfig-push %s %s",
			postClientInfo.CliAddress.String,
			global.GlobalJWireGuardini.SubnetMask)

		err = global.WriteToFile(changClientFile, changClientAddr)
		if err != nil {
			// 如果参数为空，返回 JSON 错误响应
			log.Printf("[UpdateCliInfo] 修改客户端IP地址失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("修改客户端IP地址失败, err:%v", err),
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
	log.Println("[UpdateCliAddr] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdateCliAddr] 请求类型不是Post")
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
		log.Printf("[UpdateCliAddr] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if postClientAddress.CliID == "" || postClientAddress.Address == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdateCliAddr] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查看客户端是否存在
	clientConfig.CliID.String = postClientAddress.CliID
	err := clientConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdateCliAddr] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新客户端IP地址
	clientConfig.CliAddress.String = postClientAddress.Address
	// 更新数据
	err = clientConfig.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdateCliAddr] 在数据库中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在数据库中修改客户端IP地址失败, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		postClientAddress.CliID)
	// changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", postClientAddress.Address)

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s",
		postClientAddress.Address,
		global.GlobalJWireGuardini.SubnetMask)

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdateCliAddr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
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
	log.Println("[UpdataSubnetCliAddr] start")
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdataSubnetCliAddr] 请求类型不是Post")
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
		log.Printf("[UpdataSubnetCliAddr] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if portUpdateClientAddress.CliID == "" || portUpdateClientAddress.SerID == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[UpdataSubnetCliAddr] 请求参数为空")
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
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}
	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 获取子网网段
	subnet.SerID.String = portUpdateClientAddress.SerID
	err := subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdataSubnetCliAddr] 子网不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网不存在, err:%v", err),
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查看客户端是否存在
	cliConfig.CliID.String = portUpdateClientAddress.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdataSubnetCliAddr] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d",
		global.GlobalJWireGuardini.IPPrefix,
		subnet.SerNum.Int32)

	ccdPath := fmt.Sprintf("%s/ccd",
		global.GlobalJWireGuardini.OpenVpnPath)
	cliAddress, err := FindUnusedIP(ccdPath, ipPrefix)
	if err != nil {
		log.Printf("[UpdataSubnetCliAddr] 无法获取到当前可用的客户端IP, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取到当前可用的客户端IP, err:%v", err),
			Error:   7,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新数据
	cliConfig.CliAddress.String = cliAddress
	err = cliConfig.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdataSubnetCliAddr] 在数据库中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在数据库中修改客户端IP地址失败, err:%v", err),
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		portUpdateClientAddress.CliID)
	// changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", cliAddress)

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s",
		cliAddress,
		global.GlobalJWireGuardini.SubnetMask)

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[UpdataSubnetCliAddr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
			Error:   6,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "更新客户端子网成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
