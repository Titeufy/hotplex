package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	feishuAPIBase    = "https://open.feishu.cn"
	feishuTokenAPI   = "/open-apis/auth/v3/app_access_token/internal"
	feishuMessageAPI = "/open-apis/im/v1/messages"
)

// Client wraps the Feishu Open API
type Client struct {
	appID     string
	appSecret string
	logger    *slog.Logger
	httpClient *http.Client
}

// NewClient creates a new Feishu API client
func NewClient(appID, appSecret string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		logger:    logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TokenResponse represents the app access token response
type TokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	AppAccessToken    string `json:"app_access_token"`
	Expire            int    `json:"expire"`
}

// GetAppToken fetches a new app access token
func (c *Client) GetAppToken() (string, int, error) {
	ctx := context.Background()
	return c.GetAppTokenWithContext(ctx)
}

// GetAppTokenWithContext fetches a new app access token with context
func (c *Client) GetAppTokenWithContext(ctx context.Context) (string, int, error) {
	url := feishuAPIBase + feishuTokenAPI
	
	reqBody := map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, ErrTokenFetchFailed
	}
	defer func() { _ = resp.Body.Close() }()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, err
	}
	
	if tokenResp.Code != 0 {
		c.logger.Error("Feishu token API error", "code", tokenResp.Code, "msg", tokenResp.Msg)
		return "", 0, &APIError{Code: tokenResp.Code, Msg: tokenResp.Msg}
	}
	
	return tokenResp.AppAccessToken, tokenResp.Expire, nil
}

// SendMessageRequest represents a message send request
type SendMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

// SendMessageResponse represents a message send response
type SendMessageResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	Data      *MessageData `json:"data"`
}

// MessageData represents message data in response
type MessageData struct {
	MessageID string `json:"message_id"`
}

// SendTextMessage sends a text message
func (c *Client) SendTextMessage(ctx context.Context, token, chatID, text string) (string, error) {
	url := feishuAPIBase + feishuMessageAPI + "?receive_id_type=chat_id"
	
	content := map[string]string{
		"text": text,
	}
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	
	reqBody := SendMessageRequest{
		ReceiveID: chatID,
		MsgType:   "text",
		Content:   string(contentBytes),
	}
	
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", ErrMessageSendFailed
	}
	defer func() { _ = resp.Body.Close() }()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var msgResp SendMessageResponse
	if err := json.Unmarshal(body, &msgResp); err != nil {
		return "", err
	}
	
	if msgResp.Code != 0 {
		c.logger.Error("Feishu message API error", "code", msgResp.Code, "msg", msgResp.Msg)
		return "", &APIError{Code: msgResp.Code, Msg: msgResp.Msg}
	}
	
	return msgResp.Data.MessageID, nil
}
