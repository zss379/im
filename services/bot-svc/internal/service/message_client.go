package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/model"
)

// MessageClient 封装调用 message-svc 发送消息
type MessageClient struct {
	msgSvcEndpoint string
	httpClient     *http.Client
}

func NewMessageClient(endpoint string) *MessageClient {
	return &MessageClient{
		msgSvcEndpoint: endpoint,
		httpClient:     &http.Client{},
	}
}

// SendReply 发送机器人回复消息到会话，走标准消息通道
func (c *MessageClient) SendReply(botID int64, reply string, replyType string, conv model.ConversationContext) error {
	payload := map[string]any{
		"sender_bot_id":   botID,
		"msg_type":        1, // text
		"content":         reply,
		"conversation_id": conv.ConvID,
		"conv_type":       conv.ConvType,
	}

	if conv.ConvType == 2 {
		payload["group_id"] = conv.GroupID
	}

	body, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		c.msgSvcEndpoint+"/messages",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("call message-svc: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("message-svc returned status=%d", resp.StatusCode)
	}

	log.Debug().Int64("bot_id", botID).Str("conv", conv.ConvID).
		Str("reply", reply).Msg("bot reply sent")
	return nil
}

// SendToken 发送 SSE 流式 token
func (c *MessageClient) SendToken(botID int64, msgID string, token string) {
	payload := map[string]any{
		"bot_id":   botID,
		"msg_id":   msgID,
		"token":    token,
	}
	body, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		c.msgSvcEndpoint+"/messages/sse",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Warn().Err(err).Int64("bot_id", botID).Msg("send SSE token failed")
		return
	}
	resp.Body.Close()
}
