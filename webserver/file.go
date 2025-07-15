package webservice

import (
	"jwireguard/global"
	"net"
	"net/http"
)

type FileInfo struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	FileMd5  string `json:"file_md5"`
	FileVer  string `json:"file_ver"`
}

type ResponseFileInfo struct {
	Status       bool     `json:"status"`
	Message      string   `json:"message"`
	FileInfoData FileInfo `json:"data"`
}

// 注册用户路由
func registerFileRoutes() {
	http.HandleFunc("/get_update_info", ValidateSessionMiddleware(GetFileInfo))
}

// LogoutUser 处理登出请求
func GetFileInfo(w http.ResponseWriter, r *http.Request) {
	XUserID := r.Header.Get("X-User-ID")
	global.Log.Debugln("[get_update_info] userID:", XUserID)

	addr := r.RemoteAddr
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		global.Log.Errorf("[get_update_info] 解析 IP 地址代码时出错 %d", http.StatusInternalServerError)
		return
	}
	global.Log.Debugf("[get_update_info] client [%s:%s]", ip, port)
	// 解析 URL 参数
	query := r.URL.Query()
	update_type := query.Get("update_type")
	global.Log.Debugf("[get_update_info] type:[%s]", update_type)


	
}	
