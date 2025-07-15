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
	"net"
	"net/http"
	"strings"
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

type PostCliAddrMap struct {
	CliID      string `json:"cli_id"`
	Address    string `json:"address"`
	CliMapping string `json:"cli_mapping"`
}

type ResponseAddrSuccess struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Address string `json:"address"`
}

func registerCliRoutes() {
	http.HandleFunc("/get_sub_network_list", ValidateSessionMiddleware(GetSubNetworkList))
	http.HandleFunc("/get_cli_config", ValidateSessionMiddleware(GetCliConfig))
	http.HandleFunc("/get_cli_list", ValidateSessionMiddleware(GetCliList))
	http.HandleFunc("/get_cli_info", ValidateSessionMiddleware(GetCliInfo))
	http.HandleFunc("/add_cli_config", ValidateSessionMiddleware(AddCLiConfig))
	http.HandleFunc("/del_cli_config", ValidateSessionMiddleware(DelCliConfig))
	http.HandleFunc("/update_cli_config", ValidateSessionMiddleware(UpdateCliConfig))
	http.HandleFunc("/update_cli_info", ValidateSessionMiddleware(UpdateCliInfo))
	http.HandleFunc("/update_cli_addr", ValidateSessionMiddleware(UpdateCliAddr))
	http.HandleFunc("/update_cli_map", ValidateSessionMiddleware(UpdateCliMap))
	http.HandleFunc("/update_subnet_cli_addr", ValidateSessionMiddleware(UpdataSubnetCliAddr))
}

func GetSubNetworkList(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[get_sub_network_list] userID:", XUserID)
	// 获取客户端的 IP 和端口
	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_sub_network_list] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_sub_network_list] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	userId := query.Get("user_id")
	global.Log.Debugf("[get_sub_network_list] user_id:[%s]", userId)
	// 判断参数是否为空
	if userId == "" {
		global.Log.Errorln("[get_sub_network_list] 参数为空")
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1101,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_sub_network_list] 数据库连接失败, err:%v", err)
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

	var user = database.User{}
	var subnet = database.Subnet{}

	// 初始化
	user.CreateUser(global.GlobalDB)
	subnet.CreateSubnet(global.GlobalDB)

	// 遍历用户表
	userIds, err := user.QueryUserIds(global.GlobalDB, userId)
	if err != nil {
		global.Log.Errorf("[get_sub_network_list] 无法获取用户, err:%v", err)
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取用户, err:%v", err),
			Error:   1102,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(userIds) <= 0 {
		global.Log.Errorln("[get_sub_network_list] 用户列表为空")
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "用户列表为空",
			Error:   1103,
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
		global.Log.Errorf("[get_sub_network_list] 无法获取子网序号组, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取子网序号组, err:%v", err),
			Error:   1104,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(serIds) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_sub_network_list] 子网列表为空")
		responseError := ResponseError{
			Status:  false,
			Message: "子网列表为空",
			Error:   1105,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取子网列表
	subnets, err := subnet.GetSubnetBySerIDs(global.GlobalDB, serIds)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[get_sub_network_list] 无法获取子网列表, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取子网列表, err:%v", err),
			Error:   1105,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	if len(subnets) <= 0 {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_sub_network_list] 子网列表为空")
		responseError := ResponseError{
			Status:  false,
			Message: "子网列表为空",
			Error:   1106,
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
		global.Log.Errorf("[get_sub_network_list] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
			Error:   1107,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[get_cli_config] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_cli_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_cli_config] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")
	global.Log.Debugf("[get_cli_config] cli_id:[%s]", cliId)

	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_cli_config] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1201,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_config] 数据库连接失败, err:%v", err)
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

	// 创建用户配置
	var cliConfig = database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 获取配置
	cliConfig.CliID.String = cliId
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_config] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   1202,
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
			global.Log.Errorf("[get_cli_config] 该%s文件不存在", file)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("该%s文件不存在", file),
				Error:   1203,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// Create the .ovpn file
	err = global.CreateOVPNFile(headClient, ovpnClient, files)
	if err != nil {
		global.Log.Errorf("[get_cli_config] 无法合成%s.ovpn, err:%v", cliId, err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法合成%s.ovpn, err:%v", cliId, err),
			Error:   1204,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 检查配置文件是否存在
	if !global.CheckFileExists(ovpnClient) {
		global.Log.Errorln("[get_cli_config] 客户端配置不存在")
		responseError := ResponseError{
			Status:  false,
			Message: "客户端配置不存在",
			Error:   1205,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfigText, err := ioutil.ReadFile(ovpnClient)
	if err != nil {
		global.Log.Errorf("[get_cli_config] 客户端配置读取失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端配置读取失败, err:%v", err),
			Error:   1206,
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
		global.Log.Errorf("[get_cli_config] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
			Error:   1207,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugf("[get_cli_list] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_cli_list] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_cli_list] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	serId := query.Get("ser_id")
	global.Log.Debugf("[get_cli_list] ser_id:[%s]", serId)
	// 判断参数是否为空
	if serId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_cli_list] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1301,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_list] 数据库连接失败, err:%v", err)
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

	// 创建用户配置
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 获取用户配置
	cliConfig.SerID.String = serId
	cliConfigs, err := cliConfig.GetCliConfigBySerID(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_list] 获取客户端列表失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取客户端列表失败, err:%v", err),
			Error:   1302,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	exportedCliConfigs := database.ConvertCliConfigs(cliConfigs)

	if len(cliConfigs) <= 0 {
		global.Log.Errorln("[get_cli_list] 客户端列表为空")
		responseError := ResponseError{
			Status:  false,
			Message: "客户端列表为空",
			Error:   1303,
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
		global.Log.Errorf("[get_cli_list] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
			Error:   1304,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[get_cli_info] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_cli_info] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_cli_info] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")
	global.Log.Debugf("[get_cli_info] cli_id:[%s]", cliId)
	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[get_cli_info] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   1401,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_info] 数据库连接失败, err:%v", err)
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

	// 创建用户配置
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 获取用户配置
	cliConfig.CliID.String = cliId

	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[get_cli_list] 获取客户端信息失败!, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("获取客户端信息失败!, err:%v", err),
			Error:   1402,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		cliConfig.CliID.String)

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
		cliConfig.CliAddress.String,
		global.GlobalJWireGuardini.SubnetMask,
		global.GlobalJWireGuardini.IPPrefix,
		global.GlobalJWireGuardini.NetworkMask,
		cliConfig.CliAddress.String)

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_addr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
			Error:   1403,
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
		global.Log.Errorf("[get_cli_list] 无法将JSON对象转为字符串, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法将JSON对象转为字符串, err:%v", err),
			Error:   1404,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[add_cli_config] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[add_cli_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Errorf("[add_cli_config] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[add_cli_config] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1501,
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
		global.Log.Errorf("[add_cli_config] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   1502,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	global.Log.Debugf("[add_cli_config] json:[%+v]", portCliConfig)
	// 判断参数是否为空
	if portCliConfig.CliID == "" ||
		portCliConfig.SerName == "" ||
		portCliConfig.CliName == "" ||
		portCliConfig.CliSN == "" ||
		portCliConfig.CliKey == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[add_cli_config] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1503,
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
		global.Log.Errorln("[add_cli_config] KEY值校验错误")
		responseError := ResponseError{
			Status:  false,
			Message: "KEY值校验错误",
			Error:   1504,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[add_cli_config] 数据库连接失败, err:%v", err)
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
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}

	// 数据库初始化
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 查看客户端是否存在
	cliConfig.CliID.String = portCliConfig.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		global.Log.Errorf("[add_cli_config] 客户端已存在!, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端已存在!, err:%v", err),
			Error:   1505,
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
			global.Log.Errorf("[add_cli_config] 子网网段已满, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("子网网段已满, err:%v", err),
				Error:   1506,
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
			global.Log.Errorf("[add_cli_config] 子网添加失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("子网添加失败, err:%v", err),
				Error:   1507,
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
		global.Log.Errorf("[add_cli_config] 客户端已存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端已存在, err:%v", err),
			Error:   1508,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d", global.GlobalJWireGuardini.IPPrefix, subnet.SerNum.Int32)
	cliAddress, err := FindUnusedIP(global.GlobalOpenVPNPath.CcdPath, ipPrefix)
	if err != nil {
		global.Log.Errorf("[add_cli_config] 无法获取到当前可用的客户端IP, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取到当前可用的客户端IP, err:%v", err),
			Error:   1509,
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
		global.Log.Errorf("[add_cli_config] 无法添加客户端, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法添加客户端, err:%v", err),
			Error:   1510,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	cliConfig.CliName.String = portCliConfig.CliName
	cliConfig.CliMapping.String = ""
	cliConfig.CliStatus.String = "false"
	cliConfig.EditStatus.Int32 = 0
	cliConfig.CliSN.String = portCliConfig.CliSN
	cliConfig.SerID.String = serNameSHA3
	cliConfig.SerName.String = portCliConfig.SerName
	cliConfig.Timestamp.Int64 = time.Now().Unix()
	cliConfig.OnlineStatus.String = "true"

	err = cliConfig.InsertCliConfig(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[add_cli_config] 数据库创建客户端失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("数据库创建客户端失败, err:%v", err),
			Error:   1511,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[update_cli_config] userID:", XUserID)

	currentUserID := r.Context().Value("userID").(string)
	if !global.IsAdmin(currentUserID) { // 需要实现权限验证逻辑
		responseError := ResponseError{
			Status:  false,
			Message: "权限不足",
			Error:   1600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[update_cli_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[update_cli_config] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		global.Log.Errorln("[update_cli_config] 请求类型不是Post")
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1601,
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
		global.Log.Debugf("[update_cli_config] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   1602,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	global.Log.Debugf("[update_cli_config] json:[%+v]", portCliConfig)
	// 判断参数是否为空
	if portCliConfig.CliID == "" ||
		portCliConfig.SerName == "" ||
		portCliConfig.CliName == "" ||
		portCliConfig.CliSN == "" ||
		portCliConfig.CliKey == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_config] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1603,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[update_cli_config] 数据库连接失败, err:%v", err)
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

	// 创建客户端对象
	cliConfig := database.CliConfig{}
	// 初始化
	cliConfig.CreateCliConfig(global.GlobalDB)
	// 判断是否有客户端
	// 查看客户端在数据中是否存在
	cliConfig.CliID.String = portCliConfig.CliID
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[update_cli_config] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   1604,
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
		global.Log.Errorln("[update_cli_config] KEY值校验错误")
		responseError := ResponseError{
			Status:  false,
			Message: "KEY值校验错误",
			Error:   1605,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	err = global.ShellUpdateClient(portCliConfig.CliID)
	if err != nil {
		global.Log.Errorf("[update_cli_config] 客户端更新失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端更新失败, err:%v", err),
			Error:   1606,
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

	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[del_cli_config] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[del_cli_config] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[del_cli_config] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	cliId := query.Get("cli_id")
	global.Log.Debugf("[del_cli_config] cli_id:[%s]", cliId)
	// 判断参数是否为空
	if cliId == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[del_cli_config] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1701,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[del_cli_config] 数据库连接失败, err:%v", err)
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
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}

	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 查询客户端
	cliConfig.CliID.String = cliId
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[del_cli_config] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   1702,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除客户端

	err = cliConfig.DeleteCliConfig(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[del_cli_config] 删除客户端失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("删除客户端失败, err:%v", err),
			Error:   1703,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询客户端
	err = cliConfig.GetCliConfigByCliID(global.GlobalDB)
	if err == nil {
		global.Log.Errorf("[del_cli_config] 客户端删除失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端删除失败, err:%v", err),
			Error:   1704,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 执行SHELL命令
	err = global.ShellDelClient(cliId)
	if err != nil {
		global.Log.Errorf("[del_cli_config] 无法删除客户端, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法删除客户端, err:%v", err),
			Error:   1705,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[update_cli_info] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[update_cli_info] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[update_cli_info] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_info] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1801,
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
		global.Log.Errorf("[update_cli_info] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   1802,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[update_cli_info] json:[%+v]", postClientInfoRecv)

	// 创建数据对象
	postClientInfo := postClientInfoRecv.ConvertToCliConfig()

	if postClientInfo.CliID.String == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_info] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1803,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[update_cli_info] 数据库连接失败, err:%v", err)
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

	// 初始化数据库.
	postClientInfo.CreateCliConfig(global.GlobalDB)

	// 备份数据
	postClientInfoBak := postClientInfoRecv.ConvertToCliConfig()
	// 查看客户端是否存在
	err = postClientInfoBak.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_info] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   1804,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新数据
	err = postClientInfo.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_info] 修改客户端信息失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("修改客户端信息失败, err:%v", err),
			Error:   1805,
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

		// changClientAddr := fmt.Sprintf("ifconfig-push %s %s\n",
		// 	postClientInfo.CliAddress.String,
		// 	global.GlobalJWireGuardini.SubnetMask)

		changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
			postClientInfo.CliAddress.String,
			global.GlobalJWireGuardini.SubnetMask,
			global.GlobalJWireGuardini.IPPrefix,
			global.GlobalJWireGuardini.NetworkMask,
			postClientInfo.CliAddress.String)

		err = global.WriteToFile(changClientFile, changClientAddr)
		if err != nil {
			// 如果参数为空，返回 JSON 错误响应
			global.Log.Errorf("[update_cli_info] 修改客户端IP地址失败, err:%v", err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("修改客户端IP地址失败, err:%v", err),
				Error:   1806,
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
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[update_cli_addr] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[update_cli_addr] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[update_cli_addr] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_addr] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   1901,
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
		global.Log.Errorf("[update_cli_addr] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   1902,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[update_cli_addr] json:[%+v]", postClientAddress)

	if postClientAddress.CliID == "" || postClientAddress.Address == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_addr] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   1903,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查看客户端是否存在
	clientConfig.CliID.String = postClientAddress.CliID
	err = clientConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_addr] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   1904,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	ipPrefix, _ := global.SplitIP(postClientAddress.Address)

	cliAddress, err := FindUnusedIP(global.GlobalOpenVPNPath.CcdPath, ipPrefix)
	if err != nil {
		global.Log.Errorf("[update_subnet_cli_addr] 无法获取到当前可用的客户端IP, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取到当前可用的客户端IP, err:%v", err),
			Error:   1905,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 更新客户端IP地址
	clientConfig.CliAddress.String = cliAddress
	// 更新数据
	err = clientConfig.UpdateCliConfig(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_addr] 在数据库中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在数据库中修改客户端IP地址失败, err:%v", err),
			Error:   1906,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		postClientAddress.CliID)
	// changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", postClientAddress.Address)

	// changClientAddr := fmt.Sprintf("ifconfig-push %s %s\n",
	// 	cliAddress,
	// 	global.GlobalJWireGuardini.SubnetMask)

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
		cliAddress,
		global.GlobalJWireGuardini.SubnetMask,
		global.GlobalJWireGuardini.IPPrefix,
		global.GlobalJWireGuardini.NetworkMask,
		cliAddress)

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_addr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
			Error:   1907,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseAddrSuccess{
		Status:  true,
		Message: "客户端IP修改成功!",
		Address: cliAddress,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdateCliMap(w http.ResponseWriter, r *http.Request) {

	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[update_cli_map] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[update_cli_map] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[update_cli_map] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_map] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2001,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据
	postClientAddressMapping := PostCliAddrMap{}
	clientConfig := database.CliConfig{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &postClientAddressMapping); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		global.Log.Errorf("[update_cli_map] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2002,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[update_cli_map] json:[%+v]", postClientAddressMapping)

	if postClientAddressMapping.CliID == "" || postClientAddressMapping.CliMapping == "" || postClientAddressMapping.Address == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_cli_map] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2003,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查看客户端是否存在
	clientConfig.CliID.String = postClientAddressMapping.CliID
	err = clientConfig.GetCliConfigByCliID(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_map] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   2004,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		postClientAddressMapping.CliID)
	// changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", postClientAddress.Address)

	// changClientAddr := fmt.Sprintf("ifconfig-push %s %s\n",
	// 	cliAddress,
	// 	global.GlobalJWireGuardini.SubnetMask)

	cliAddress, _, err := global.ParseConfigFile(changClientFile)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_map] 解析 %s 文件出错, err:%v", changClientFile, err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析 %s 文件出错, err:%v", changClientFile, err),
			Error:   2005,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
		cliAddress,
		global.GlobalJWireGuardini.SubnetMask,
		global.GlobalJWireGuardini.IPPrefix,
		global.GlobalJWireGuardini.NetworkMask,
		cliAddress)

	// 判断是否有逗号分隔符
	if strings.Contains(postClientAddressMapping.CliMapping, ",") {
		// 按照逗号分割
		cidrList := strings.Split(postClientAddressMapping.CliMapping, ",")
		// 遍历每个 CIDR 段
		for _, cidr := range cidrList {
			ip, mask, network, err := global.ParseCIDR(cidr)
			if err != nil {
				// 如果参数为空，返回 JSON 错误响应
				global.Log.Errorf("[update_cli_map] 解析 %s 时出错, err:%v", cidr, err)
				responseError := ResponseError{
					Status:  false,
					Message: fmt.Sprintf("解析 %s 时出错, err:%v", cidr, err),
					Error:   2006,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseError)
				return
			}

			// 输出解析结果
			global.Log.Debugf("[update_cli_map] IP 地址: %s, 子网掩码: %s, 网络地址: %s", ip, mask, network)
			changClientAddr = fmt.Sprintf("%spush \"route %s %s %s\"\n", changClientAddr, network, mask, postClientAddressMapping.Address)
		}

	} else {
		ip, mask, network, err := global.ParseCIDR(postClientAddressMapping.CliMapping)
		if err != nil {
			// 如果参数为空，返回 JSON 错误响应
			global.Log.Errorf("[update_cli_map] 解析 %s 时出错, err:%v", postClientAddressMapping.CliMapping, err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("解析 %s 时出错, err:%v", postClientAddressMapping.CliMapping, err),
				Error:   2007,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}

		// 输出解析结果
		global.Log.Debugf("[update_cli_map] IP 地址: %s, 子网掩码: %s, 网络地址: %s", ip, mask, network)
		changClientAddr = fmt.Sprintf("%spush \"route %s %s %s\"\n", changClientAddr, network, mask, postClientAddressMapping.Address)
	}

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_cli_addr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
			Error:   2008,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "客户端映射成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func UpdataSubnetCliAddr(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugf("[update_subnet_cli_addr] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[update_subnet_cli_addr] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[update_subnet_cli_addr] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_subnet_cli_addr] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2101,
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
		global.Log.Errorf("[update_subnet_cli_addr] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2102,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	global.Log.Debugf("[update_subnet_cli_addr] json:[%+v]", portUpdateClientAddress)

	if portUpdateClientAddress.CliID == "" || portUpdateClientAddress.SerID == "" {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorln("[update_subnet_cli_addr] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2103,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	global.GlobalDB, err = database.MonitorDatabase(global.GlobalDB)
	if err != nil {
		global.Log.Errorf("[update_subnet_cli_addr] 数据库连接失败, err:%v", err)
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
	subnet := database.Subnet{}
	cliConfig := database.CliConfig{}
	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)
	cliConfig.CreateCliConfig(global.GlobalDB)

	// 获取子网网段
	subnet.SerID.String = portUpdateClientAddress.SerID
	err = subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_subnet_cli_addr] 子网不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网不存在, err:%v", err),
			Error:   2104,
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
		global.Log.Errorf("[update_subnet_cli_addr] 客户端不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("客户端不存在, err:%v", err),
			Error:   2106,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 获取客户端可用的IP地址
	ipPrefix := fmt.Sprintf("%s.%d",
		global.GlobalJWireGuardini.IPPrefix,
		subnet.SerNum.Int32)

	cliAddress, err := FindUnusedIP(global.GlobalOpenVPNPath.CcdPath, ipPrefix)
	if err != nil {
		global.Log.Errorf("[update_subnet_cli_addr] 无法获取到当前可用的客户端IP, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("无法获取到当前可用的客户端IP, err:%v", err),
			Error:   2107,
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
		global.Log.Errorf("[update_subnet_cli_addr] 在数据库中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在数据库中修改客户端IP地址失败, err:%v", err),
			Error:   2105,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	changClientFile := fmt.Sprintf("%s/%s",
		global.GlobalOpenVPNPath.CcdPath,
		portUpdateClientAddress.CliID)
	// changClientAddr := fmt.Sprintf("ifconfig-push %s 255.255.0.0", cliAddress)

	// changClientAddr := fmt.Sprintf("ifconfig-push %s %s\n",
	// 	cliAddress,
	// 	global.GlobalJWireGuardini.SubnetMask)

	changClientAddr := fmt.Sprintf("ifconfig-push %s %s\npush \"route %s.0.0 %s %s\"\n",
		cliAddress,
		global.GlobalJWireGuardini.SubnetMask,
		global.GlobalJWireGuardini.IPPrefix,
		global.GlobalJWireGuardini.NetworkMask,
		cliAddress)

	err = global.WriteToFile(changClientFile, changClientAddr)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		global.Log.Errorf("[update_subnet_cli_addr] 在文件中修改客户端IP地址失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("在文件中修改客户端IP地址失败, err:%v", err),
			Error:   2106,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseAddrSuccess{
		Status:  true,
		Message: "更新客户端子网成功!",
		Address: cliAddress,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
