package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/rc-svc/internal/engine"
	"github.com/shulian-paas/im/rc-svc/internal/model"
	"github.com/shulian-paas/im/rc-svc/internal/repo"
)

type RCService struct {
	mysqlRepo *repo.MySQLRepo
	cache     *repo.Cache
	dfa       *engine.DFAEngine
	tenantID  int64        // default tenant ID for global rules
	mu        sync.Mutex
}

func NewRCService(mysqlRepo *repo.MySQLRepo, cache *repo.Cache) *RCService {
	return &RCService{
		mysqlRepo: mysqlRepo,
		cache:     cache,
		dfa:       engine.New(),
		tenantID:  0, // tenant-agnostic defaults use 0
	}
}

// Init loads sensitive words into the DFA engine and seeds defaults.
func (s *RCService) Init(ctx context.Context) error {
	// load all active words across tenants
	for _, tid := range []int64{0} { // expand when multi-tenant is needed
		words, err := s.mysqlRepo.GetAllActiveWords(ctx, tid)
		if err != nil {
			return fmt.Errorf("load words for tenant %d: %w", tid, err)
		}
		entries := make([]engine.WordEntry, len(words))
		for i, w := range words {
			entries[i] = engine.WordEntry{
				Word:        w.Word,
				Strategy:    w.Strategy,
				Replacement: w.Replacement,
			}
		}
		if len(entries) > 0 {
			s.dfa.Build(entries)
		}
	}
	return nil
}

// RefreshWords reloads the DFA engine from the database.
func (s *RCService) RefreshWords(ctx context.Context, tenantID int64) error {
	words, err := s.mysqlRepo.GetAllActiveWords(ctx, tenantID)
	if err != nil {
		return err
	}
	entries := make([]engine.WordEntry, len(words))
	for i, w := range words {
		entries[i] = engine.WordEntry{
			Word:        w.Word,
			Strategy:    w.Strategy,
			Replacement: w.Replacement,
		}
	}
	s.dfa.Build(entries)
	return nil
}

// ---- Sensitive Word Check ----

// CheckSensitive runs the DFA engine against the text.
func (s *RCService) CheckSensitive(ctx context.Context, tenantID int64, content string) (*model.SensitiveCheckResp, error) {
	result := s.dfa.Check(content)
	if !result.HasMatch {
		return &model.SensitiveCheckResp{Passed: true}, nil
	}

	// Determine overall action: if any word has strategy=2 (block), block.
	blocked := false
	var hitWords []string
	for _, h := range result.Words {
		hitWords = append(hitWords, h.Word)
		if h.Strategy == engine.SensitiveStrategyBlock {
			blocked = true
		}
	}

	cleaned, _ := s.dfa.Replace(content)

	return &model.SensitiveCheckResp{
		Passed:   !blocked,
		HitWords: hitWords,
		Cleaned:  cleaned,
		Blocked:  blocked,
	}, nil
}

// ---- Rate Limit Check ----

func (s *RCService) CheckRateLimit(ctx context.Context, tenantID int64, targetID int64, targetType int8) (*model.RateLimitCheckResp, error) {
	rule, err := s.mysqlRepo.GetRateLimitRule(ctx, tenantID, targetType)
	if err != nil {
		return nil, fmt.Errorf("get rate limit rule: %w", err)
	}
	if rule == nil {
		// no rule configured — pass
		return &model.RateLimitCheckResp{Passed: true, Remaining: -1}, nil
	}

	key := repo.RateLimitKey(targetType, targetID)
	allowed, remaining, err := s.cache.CheckRateLimit(ctx, key, rule.MaxCount, rule.TimeWindowSeconds)
	if err != nil {
		return nil, fmt.Errorf("rate limit check: %w", err)
	}

	return &model.RateLimitCheckResp{
		Passed:    allowed,
		Remaining: remaining,
	}, nil
}

// ---- File Size Check ----

func (s *RCService) CheckFileLimit(ctx context.Context, tenantID int64, fileType string, fileSizeBytes int) (*model.FileLimitCheckResp, error) {
	limit, err := s.mysqlRepo.GetFileLimit(ctx, tenantID, fileType)
	if err != nil {
		return nil, fmt.Errorf("get file limit: %w", err)
	}
	if limit == nil {
		return &model.FileLimitCheckResp{Passed: true}, nil
	}

	maxBytes := limit.MaxSizeMB * 1024 * 1024
	return &model.FileLimitCheckResp{
		Passed:     fileSizeBytes <= maxBytes,
		MaxSizeMB:  limit.MaxSizeMB,
		Extensions: limit.AllowedExtensions,
	}, nil
}

// ---- Full Check Chain ----

func (s *RCService) CheckChain(ctx context.Context, tenantID int64, content string, senderID int64, senderType int8, fileType string, fileSize int) *model.CheckChainResp {
	resp := &model.CheckChainResp{Passed: true}

	// 1. Sensitive word check
	if content != "" {
		sensitiveResult, err := s.CheckSensitive(ctx, tenantID, content)
		if err != nil {
			log.Warn().Err(err).Msg("sensitive check failed, continuing")
		} else {
			resp.SensitiveCheck = sensitiveResult
			if sensitiveResult.Blocked {
				resp.Passed = false
			}
		}
	}

	// 2. Rate limit check
	if senderID > 0 {
		rateResult, err := s.CheckRateLimit(ctx, tenantID, senderID, senderType)
		if err != nil {
			log.Warn().Err(err).Msg("rate limit check failed, continuing")
		} else {
			resp.RateLimitCheck = rateResult
			if !rateResult.Passed {
				resp.Passed = false
			}
		}
	}

	// 3. File size check
	if fileType != "" && fileSize > 0 {
		fileResult, err := s.CheckFileLimit(ctx, tenantID, fileType, fileSize)
		if err != nil {
			log.Warn().Err(err).Msg("file limit check failed, continuing")
		} else {
			resp.FileLimitCheck = fileResult
			if !fileResult.Passed {
				resp.Passed = false
			}
		}
	}

	return resp
}

// ---- CRUD helpers ----

func (s *RCService) CreateWord(ctx context.Context, req *model.SensitiveWordCreateReq) error {
	word := &model.SensitiveWord{
		TenantID:    req.TenantID,
		Word:        req.Word,
		Strategy:    req.Strategy,
		Replacement: req.Replacement,
		Category:    req.Category,
		Status:      1,
	}
	if word.Strategy == 0 {
		word.Strategy = model.SensitiveStrategyReplace
	}
	if word.Replacement == "" {
		word.Replacement = "***"
	}

	if err := s.mysqlRepo.CreateSensitiveWord(ctx, word); err != nil {
		return err
	}

	// async refresh DFA
	go func() {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.RefreshWords(refreshCtx, req.TenantID); err != nil {
			log.Warn().Err(err).Msg("refresh DFA after create")
		}
	}()

	return nil
}

func (s *RCService) BatchImportWords(ctx context.Context, req *model.SensitiveWordBatchReq) (int, error) {
	words := make([]model.SensitiveWord, len(req.Words))
	for i, w := range req.Words {
		strategy := w.Strategy
		if strategy == 0 {
			strategy = model.SensitiveStrategyReplace
		}
		replacement := w.Replacement
		if replacement == "" {
			replacement = "***"
		}
		words[i] = model.SensitiveWord{
			TenantID:    req.TenantID,
			Word:        w.Word,
			Strategy:    strategy,
			Replacement: replacement,
			Category:    w.Category,
			Status:      1,
		}
	}
	if err := s.mysqlRepo.BatchCreateSensitiveWords(ctx, words); err != nil {
		return 0, err
	}

	go func() {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.RefreshWords(refreshCtx, req.TenantID); err != nil {
			log.Warn().Err(err).Msg("refresh DFA after batch import")
		}
	}()

	return len(words), nil
}

func (s *RCService) ListWords(ctx context.Context, tenantID int64, page, pageSize int) ([]model.SensitiveWord, int64, error) {
	return s.mysqlRepo.ListSensitiveWords(ctx, tenantID, page, pageSize)
}

func (s *RCService) UpdateWord(ctx context.Context, wordID int64, updates map[string]interface{}, tenantID int64) error {
	return s.mysqlRepo.UpdateSensitiveWord(ctx, wordID, updates)
}

func (s *RCService) DeleteWord(ctx context.Context, wordID int64, tenantID int64) error {
	if err := s.mysqlRepo.DeleteSensitiveWord(ctx, wordID); err != nil {
		return err
	}
	go func() {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.RefreshWords(refreshCtx, tenantID); err != nil {
			log.Warn().Err(err).Msg("refresh DFA after delete")
		}
	}()
	return nil
}

func (s *RCService) CreateRateLimitRule(ctx context.Context, req *model.RateLimitRuleCreateReq) error {
	rule := &model.FrequencyControlRule{
		TenantID:          req.TenantID,
		TargetType:        req.TargetType,
		MaxCount:          req.MaxCount,
		TimeWindowSeconds: req.TimeWindowSeconds,
		Action:            1,
		Status:            1,
	}
	return s.mysqlRepo.CreateRateLimitRule(ctx, rule)
}

func (s *RCService) ListRateLimitRules(ctx context.Context, tenantID int64) ([]model.FrequencyControlRule, error) {
	return s.mysqlRepo.ListRateLimitRules(ctx, tenantID)
}

func (s *RCService) UpdateRateLimitRule(ctx context.Context, ruleID int64, updates map[string]interface{}, tenantID int64) error {
	return s.mysqlRepo.UpdateRateLimitRule(ctx, ruleID, updates)
}

func (s *RCService) DeleteRateLimitRule(ctx context.Context, ruleID int64, tenantID int64) error {
	return s.mysqlRepo.DeleteRateLimitRule(ctx, ruleID)
}

func (s *RCService) ListFileLimits(ctx context.Context, tenantID int64) ([]model.FileLimitConfig, error) {
	return s.mysqlRepo.ListFileLimits(ctx, tenantID)
}

func (s *RCService) UpdateFileLimit(ctx context.Context, configID int64, updates map[string]interface{}, tenantID int64) error {
	return s.mysqlRepo.UpdateFileLimit(ctx, configID, updates)
}

func (s *RCService) DeleteFileLimit(ctx context.Context, configID int64, tenantID int64) error {
	return s.mysqlRepo.DeleteFileLimit(ctx, configID)
}

func (s *RCService) CreateFileLimit(ctx context.Context, req *model.FileLimitCreateReq) error {
	limit := &model.FileLimitConfig{
		TenantID:          req.TenantID,
		FileType:          req.FileType,
		MaxSizeMB:         req.MaxSizeMB,
		AllowedExtensions: req.AllowedExtensions,
		Status:            1,
	}
	return s.mysqlRepo.CreateFileLimit(ctx, limit)
}
