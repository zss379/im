package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/shulian-paas/im/bot-svc/internal/model"
)

type BotRepo struct {
	db *sqlx.DB
}

func NewBotRepo(db *sqlx.DB) *BotRepo {
	return &BotRepo{db: db}
}

func (r *BotRepo) GetByID(botID int64) (*model.Bot, error) {
	var bot model.Bot
	err := r.db.Get(&bot, "SELECT * FROM bot WHERE bot_id = ?", botID)
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *BotRepo) GetActiveBotIDs() ([]int64, error) {
	var ids []int64
	err := r.db.Select(&ids, "SELECT bot_id FROM bot WHERE status = 1")
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *BotRepo) Create(bot *model.Bot) error {
	result, err := r.db.Exec(`
		INSERT INTO bot (tenant_id, bot_type, bot_name, avatar_url, description,
		                 webhook_url, api_key, response_mode, callback_url, ip_whitelist, creator_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bot.TenantID, bot.BotType, bot.BotName, bot.AvatarURL, bot.Description,
		bot.WebhookURL, bot.APIKey, bot.ResponseMode, bot.CallbackURL, bot.IPWhitelist, bot.CreatorID)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	bot.BotID = id
	return nil
}

func (r *BotRepo) Update(bot *model.Bot) error {
	_, err := r.db.Exec(`
		UPDATE bot SET bot_name=?, avatar_url=?, description=?, webhook_url=?,
		               api_key=?, response_mode=?, callback_url=?, ip_whitelist=?, status=?
		WHERE bot_id=?`,
		bot.BotName, bot.AvatarURL, bot.Description, bot.WebhookURL,
		bot.APIKey, bot.ResponseMode, bot.CallbackURL, bot.IPWhitelist, bot.Status, bot.BotID)
	return err
}

func (r *BotRepo) ToggleStatus(botID int64, status int8) error {
	_, err := r.db.Exec("UPDATE bot SET status=? WHERE bot_id=?", status, botID)
	return err
}

func (r *BotRepo) Delete(botID int64) error {
	_, err := r.db.Exec("DELETE FROM bot WHERE bot_id=?", botID)
	return err
}

func (r *BotRepo) SetOpenIMUserID(botID int64, openimUserID int64) error {
	_, err := r.db.Exec("UPDATE bot SET openim_user_id=? WHERE bot_id=?", openimUserID, botID)
	return err
}

// ListByTenant 按租户分页查询机器人
func (r *BotRepo) ListByTenant(tenantID int64, botType *int8, page, pageSize int) ([]model.Bot, int, error) {
	var bots []model.Bot
	var total int

	where := "WHERE tenant_id=?"
	args := []any{tenantID}
	if botType != nil {
		where += " AND bot_type=?"
		args = append(args, *botType)
	}

	err := r.db.Get(&total, "SELECT COUNT(*) FROM bot "+where, args...)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = r.db.Select(&bots, "SELECT * FROM bot "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	return bots, total, nil
}
