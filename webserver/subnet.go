// webservice/subnet.go
package webservice

import (
	"encoding/json"
	"fmt"
	"jwireguard/database"
	"jwireguard/global"
	"log"
	"net"
	"net/http"
)

func registerSubnetRoutes() {
	http.HandleFunc("/add_subnet", ValidateSessionMiddleware(AddSubnet))
	http.HandleFunc("/edit_subnet", ValidateSessionMiddleware(EditSubnet))
	http.HandleFunc("/del_subnet", ValidateSessionMiddleware(DelSubnet))
}

func AddSubnet(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[add_subnet] userID:", XUserID)
	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[add_subnet] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[add_subnet] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[add_subnet] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2201,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据
	exportPortSubnet := database.ExportedSubnet{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &exportPortSubnet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[add_subnet] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2202,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	log.Printf("[add_subnet] json:[%+v]", exportPortSubnet)
	// 将字符串类型转为SQL类型
	portSubnet := exportPortSubnet.ConvertToSubnet()

	if portSubnet.SerName.String == "" ||
		(portSubnet.CliNum.Int32 <= 0 && portSubnet.CliNum.Int32 >= 255) ||
		(portSubnet.SerNum.Int32 <= 0 && portSubnet.SerNum.Int32 >= 255) {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[add_subnet] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2203,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 生成用户ID
	if portSubnet.SerID.String == "" {
		portSubnet.SerID.String = global.GenerateMD5(portSubnet.SerName.String)
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 初始化数据库
	portSubnet.CreateSubnet(global.GlobalDB)

	// 查询子网是否存在
	err = portSubnet.GetSubnetBySerId(global.GlobalDB)
	if err == nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_subnet] 子网已存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网已存在, err:%v", err),
			Error:   2204,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加数据库
	err = portSubnet.InsertSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[add_subnet] 添加子网失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("添加子网失败, err:%v", err),
			Error:   2205,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 配置Iptables
	rules := fmt.Sprintf("-s %s.%d.0/24 -d %s.0.0/16 -j ACCEPT",
		global.GlobalJWireGuardini.IPPrefix,
		portSubnet.SerNum.Int32,
		global.GlobalJWireGuardini.IPPrefix)

	if !global.CheckIptablesRule(rules) {
		err := global.AddIptablesRule(rules)
		if err != nil {
			log.Printf("[add_subnet] 路由配置错误 '%s': %v", rules, err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("路由配置错误 '%s': %v", rules, err),
				Error:   2206,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
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
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[edit_subnet] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[edit_subnet] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[edit_subnet] client [%s:%s]", ip, port)
	// 确保请求方法是POST
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[edit_subnet] 请求类型不是Post")
		responseError := ResponseError{
			Status:  false,
			Message: "请求类型不是Post",
			Error:   2301,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 创建一个PostCliConfig实例来存储解析后的数据
	exportPortSubnet := database.ExportedSubnet{}

	// 使用封装的parseJSONBody函数解析请求体
	if err := parseJSONBody(r, &exportPortSubnet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("[edit_subnet] 解析JSON请求参数错误, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("解析JSON请求参数错误, err:%v", err),
			Error:   2302,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}
	log.Printf("[edit_subnet] json:[%+v]", exportPortSubnet)
	portSubnet := exportPortSubnet.ConvertToSubnet()

	if portSubnet.SerID.String == "" ||
		(portSubnet.CliNum.Int32 <= 0 && portSubnet.CliNum.Int32 >= 255) ||
		(portSubnet.SerNum.Int32 <= 0 && portSubnet.SerNum.Int32 >= 255) {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[edit_subnet] 请求参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "请求参数为空",
			Error:   2303,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 初始化数据库
	portSubnet.CreateSubnet(global.GlobalDB)

	portSubnetbak := portSubnet

	// 查询子网是否存在
	err = portSubnetbak.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_subnet] 子网不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网不存在, err:%v", err),
			Error:   2304,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 添加数据库
	err = portSubnet.UpdateSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[edit_subnet] 子网更新失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网更新失败, err:%v", err),
			Error:   2305,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	log.Println("旧的：", portSubnetbak.SerNum.Int32)
	log.Println("新的：", portSubnet.SerNum.Int32)
	if portSubnetbak.SerNum.Int32 != portSubnet.SerNum.Int32 {
		// 配置Iptables
		delRules := fmt.Sprintf("-s %s.%d.0/24 -d %s.0.0/16 -j ACCEPT",
			global.GlobalJWireGuardini.IPPrefix,
			portSubnetbak.SerNum.Int32,
			global.GlobalJWireGuardini.IPPrefix)

		if global.CheckIptablesRule(delRules) {
			err := global.DeleteIptablesRule(delRules)
			if err != nil {
				log.Printf("[edit_subnet] 路由删除错误 '%s': %v", delRules, err)
				responseError := ResponseError{
					Status:  false,
					Message: fmt.Sprintf("路由删除错误 '%s': %v", delRules, err),
					Error:   2306,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseError)
				return
			}
		}

		// 配置Iptables
		addRules := fmt.Sprintf("-s %s.%d.0/24 -d %s.0.0/16 -j ACCEPT",
			global.GlobalJWireGuardini.IPPrefix,
			portSubnet.SerNum.Int32,
			global.GlobalJWireGuardini.IPPrefix)

		if !global.CheckIptablesRule(addRules) {
			err := global.AddIptablesRule(addRules)
			if err != nil {
				log.Printf("[edit_subnet] 路由配置错误 '%s': %v", addRules, err)
				responseError := ResponseError{
					Status:  false,
					Message: fmt.Sprintf("路由配置错误 '%s': %v", addRules, err),
					Error:   2307,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(responseError)
				return
			}
		}
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
	XUserID := r.Header.Get("X-User-ID")
	log.Println("[del_subnet] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Printf("[del_subnet] Error parsing IP address code %d", http.StatusInternalServerError)
		return
	}
	log.Printf("[del_subnet] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	serId := query.Get("ser_id")
	log.Printf("[del_subnet] ser_id:[%s]", serId)
	// 判断参数是否为空
	if serId == "" {
		// 如果参数为空，返回 JSON 错误响应
		log.Println("[del_subnet] 参数为空")
		responseError := ResponseError{
			Status:  false,
			Message: "参数为空",
			Error:   2401,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 查询连接状态
	database.MonitorDatabase(global.GlobalDB)

	// 创建数据库对象
	subnet := database.Subnet{}
	// 初始化数据库
	subnet.CreateSubnet(global.GlobalDB)

	// 查看子网是否存在
	subnet.SerID.String = serId
	err = subnet.GetSubnetBySerId(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[del_subnet] 子网不存在, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网不存在, err:%v", err),
			Error:   2402,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除子网
	err = subnet.DeleteSubnet(global.GlobalDB)
	if err != nil {
		// 如果参数为空，返回 JSON 错误响应
		log.Printf("[del_subnet] 子网删除失败, err:%v", err)
		responseError := ResponseError{
			Status:  false,
			Message: fmt.Sprintf("子网删除失败, err:%v", err),
			Error:   2403,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseError)
		return
	}

	// 删除Iptables
	rules := fmt.Sprintf("-s %s.%d.0/24 -d %s.0.0/16 -j ACCEPT",
		global.GlobalJWireGuardini.IPPrefix,
		subnet.SerNum.Int32,
		global.GlobalJWireGuardini.IPPrefix)
	if global.CheckIptablesRule(rules) {
		err := global.DeleteIptablesRule(rules)
		if err != nil {
			log.Printf("[del_subnet] 路由删除错误 '%s': %v", rules, err)
			responseError := ResponseError{
				Status:  false,
				Message: fmt.Sprintf("路由删除错误 '%s': %v", rules, err),
				Error:   2404,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(responseError)
			return
		}
	}

	// 返回结果
	responseSuccess := ResponseSuccess{
		Status:  true,
		Message: "子网删除成功!",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseSuccess)
}
