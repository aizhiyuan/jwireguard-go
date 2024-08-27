// webservice/subnet.go
package webservice

import (
	"encoding/json"
	"jwireguard/database"
	"jwireguard/global"
	"net/http"
)

func registerSubnetRoutes() {
	http.HandleFunc("/add_subnet", AddSubnet)
	http.HandleFunc("/edit_subnet", EditSubnet)
	http.HandleFunc("/del_subnet", DelSubnet)
}

func AddSubnet(w http.ResponseWriter, r *http.Request) {

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
	portSubnet := database.Subnet{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portSubnet); err != nil {
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

	if portSubnet.SerID == "" ||
		portSubnet.SerName == "" ||
		(portSubnet.CliNum <= 0 && portSubnet.CliNum >= 255) ||
		(portSubnet.SerNum <= 0 && portSubnet.SerNum >= 255) {
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
	portSubnet.CreateSubnet(global.GlobalDB)

	// 查询子网是否存在
	err := portSubnet.GetSubnetBySerId(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网已存在!",
			Error:   4,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加数据库
	err = portSubnet.InsertSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "添加子网失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "子网添加成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func EditSubnet(w http.ResponseWriter, r *http.Request) {
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
	portSubnet := database.Subnet{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &portSubnet); err != nil {
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

	if portSubnet.SerID == "" ||
		portSubnet.SerName == "" ||
		(portSubnet.CliNum <= 0 && portSubnet.CliNum >= 255) ||
		(portSubnet.SerNum < 0 && portSubnet.SerNum >= 255) {
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
	portSubnet.CreateSubnet(global.GlobalDB)

	// 查询子网是否存在
	err := portSubnet.GetSubnetBySerId(global.GlobalDB)
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

	// 添加数据库
	err = portSubnet.UpdateSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网更新失败!",
			Error:   5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "子网更新成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}

func DelSubnet(w http.ResponseWriter, r *http.Request) {
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

	// 创建数据库对象
	subnet := database.Subnet{}
	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)

	// 查看子网是否存在
	subnet.SerID = serId
	err := subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网不存在!",
			Error:   2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除子网
	err = subnet.DeleteSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		responseError := ResponseError{
			Status:  false,
			Message: "子网删除失败!",
			Error:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "子网删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
