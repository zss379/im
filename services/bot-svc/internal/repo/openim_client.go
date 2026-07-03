package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shulian-paas/im/bot-svc/internal/config"
)

// OpenIMClient 封装 OpenIM Server API 调用
type OpenIMClient struct {
	cfg     *config.OpenIMConfig
	client  *http.Client
}

func NewOpenIMClient(cfg *config.OpenIMConfig) *OpenIMClient {
	return &OpenIMClient{
		cfg:    cfg,
		client: &http.Client{},
	}
}

// CreateUser 在 OpenIM 中创建机器人用户
func (c *OpenIMClient) CreateUser(userID int64, nickname, avatarURL string) error {
	payload := map[string]any{
		"userID":   fmt.Sprintf("bot_%d", userID),
		"nickname": nickname,
		"faceURL":  avatarURL,
		"ex":       fmt.Sprintf(`{"bot_id":%d,"type":"bot"}`, userID),
	}
	return c.post("/user/create", payload)
}

// DeactivateUser 停用 OpenIM 用户
func (c *OpenIMClient) DeactivateUser(userID int64) error {
	payload := map[string]any{
		"userID": fmt.Sprintf("bot_%d", userID),
	}
	return c.post("/user/account_check", payload)
}

// DeleteUser 删除 OpenIM 用户
func (c *OpenIMClient) DeleteUser(userID int64) error {
	payload := map[string]any{
		"userIDs": []string{fmt.Sprintf("bot_%d", userID)},
	}
	return c.post("/user/delete", payload)
}

func (c *OpenIMClient) post(path string, payload any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.cfg.APIEndpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("secret", c.cfg.Secret)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("openim api error: status=%d path=%s", resp.StatusCode, path)
	}
	return nil
}
