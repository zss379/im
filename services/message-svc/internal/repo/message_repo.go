package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shulian-paas/im/message-svc/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const messagesCollection = "messages"

// MessageRepo MongoDB 消息数据访问
type MessageRepo struct {
	db *mongo.Database
}

func NewMessageRepo(db *mongo.Database) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) coll() *mongo.Collection {
	return r.db.Collection(messagesCollection)
}

// Insert 持久化新消息
func (r *MessageRepo) Insert(ctx context.Context, msg *model.Message) error {
	msg.SendTime = time.Now()
	msg.Status = model.MsgStatusSent
	res, err := r.coll().InsertOne(ctx, msg)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	msg.ID = res.InsertedID
	return nil
}

// FindByMsgID 按业务 msg_id 查询
func (r *MessageRepo) FindByMsgID(ctx context.Context, msgID string) (*model.Message, error) {
	var msg model.Message
	err := r.coll().FindOne(ctx, bson.M{"msg_id": msgID}).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find by msg_id: %w", err)
	}
	return &msg, nil
}

// FindByClientMsgID 按客户端幂等 ID 查询
func (r *MessageRepo) FindByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.Message, error) {
	var msg model.Message
	err := r.coll().FindOne(ctx, bson.M{
		"conversation_id": conversationID,
		"client_msg_id":   clientMsgID,
	}).Decode(&msg)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find by client_msg_id: %w", err)
	}
	return &msg, nil
}

// FindByConversation 按会话游标翻页查历史消息
// cursor: base64 编码的 {t: lastSendTime, i: lastID}，首次传空
func (r *MessageRepo) FindByConversation(ctx context.Context, conversationID int64, cursor string, limit int, direction int) ([]model.Message, string, error) {
	filter := bson.M{"conversation_id": conversationID, "is_deleted": false}

	opts := options.Find().
		SetSort(bson.D{{Key: "send_time", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit + 1))

	if cursor != "" {
		decoded, err := decodeCursor(cursor)
		if err == nil {
			lastTime := time.UnixMilli(decoded.LastSendTime)
			lastOID, oidErr := primitive.ObjectIDFromHex(decoded.LastID)
			if direction == 0 {
				// 向前翻（更早消息）
				filter["$or"] = []bson.M{
					{"send_time": bson.M{"$lt": lastTime}},
					{"send_time": lastTime, "_id": bson.M{"$lt": lastOID}},
				}
			} else {
				// 向后翻（更新消息）
				_ = oidErr
				filter["$or"] = []bson.M{
					{"send_time": bson.M{"$gt": lastTime}},
					{"send_time": lastTime, "_id": bson.M{"$gt": lastOID}},
				}
				opts.SetSort(bson.D{{Key: "send_time", Value: 1}, {Key: "_id", Value: 1}})
			}
		}
	}

	cursorOpts := opts
	cur, err := r.coll().Find(ctx, filter, cursorOpts)
	if err != nil {
		return nil, "", fmt.Errorf("find by conversation: %w", err)
	}
	defer cur.Close(ctx)

	var messages []model.Message
	if err := cur.All(ctx, &messages); err != nil {
		return nil, "", fmt.Errorf("decode messages: %w", err)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	if direction == 1 && len(messages) > 0 {
		// 反向翻页后，把结果反转回时间倒序
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		last := messages[len(messages)-1]
		nextCursor = encodeCursor(last.SendTime, last.ID)
	}

	return messages, nextCursor, nil
}

// UpdateStatus 更新消息状态（撤回等）
func (r *MessageRepo) UpdateStatus(ctx context.Context, msgID string, status int8, recallTime *time.Time) error {
	update := bson.M{"$set": bson.M{"status": status, "edit_time": time.Now()}}
	if recallTime != nil {
		update["$set"].(bson.M)["recall_time"] = recallTime
	}
	_, err := r.coll().UpdateOne(ctx, bson.M{"msg_id": msgID}, update)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// Search 搜索消息
func (r *MessageRepo) Search(ctx context.Context, req *model.SearchReq) ([]model.Message, int64, error) {
	filter := bson.M{"is_deleted": false}

	if req.Q != "" {
		filter["content.text"] = bson.M{"$regex": req.Q, "$options": "i"}
	}
	if req.ConversationID > 0 {
		filter["conversation_id"] = req.ConversationID
	}
	if req.SenderID > 0 {
		filter["sender_id"] = req.SenderID
	}
	if req.MsgType > 0 {
		filter["msg_type"] = req.MsgType
	}
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err == nil {
			filter["send_time"] = bson.M{"$gte": t}
		}
	}
	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err == nil {
			if existing, ok := filter["send_time"].(bson.M); ok {
				existing["$lte"] = t
			} else {
				filter["send_time"] = bson.M{"$lte": t}
			}
		}
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count search: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "send_time", Value: -1}}).
		SetSkip(int64((req.Page - 1) * req.PageSize)).
		SetLimit(int64(req.PageSize))

	cur, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("search messages: %w", err)
	}
	defer cur.Close(ctx)

	var messages []model.Message
	if err := cur.All(ctx, &messages); err != nil {
		return nil, 0, fmt.Errorf("decode search results: %w", err)
	}

	return messages, total, nil
}

// encodeCursor 编码游标：base64(time + _id)
func encodeCursor(t time.Time, id interface{}) string {
	oid, ok := id.(primitive.ObjectID)
	if !ok {
		return ""
	}
	c := model.Cursor{
		LastSendTime: t.UnixMilli(),
		LastID:       oid.Hex(),
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// decodeCursor 解码游标
func decodeCursor(cursor string) (*model.Cursor, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}
	var c model.Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
