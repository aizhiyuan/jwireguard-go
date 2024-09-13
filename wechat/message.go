package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// 定义结构体用于发送消息
type WeChatMessage struct {
	Touser  string `json:"touser"`
	AgentID int    `json:"agentid"`
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	Safe int `json:"safe"`
}

// 获取Access Token
func GetAccessToken(corpID, secret string) (string, error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", corpID, secret)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

// 发送消息
func SendWeChatMessage(accessToken string, message WeChatMessage) error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", accessToken)

	// 将消息结构体转为JSON
	messageData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 发送HTTP POST请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(messageData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析响应
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("企业微信API错误: %s", result.ErrMsg)
	}

	return nil
}
