package service

import (
	"context"
	"testing"

	"github.com/shulian-paas/im/rc-svc/internal/engine"
	"github.com/shulian-paas/im/rc-svc/internal/model"
)

func TestCheckSensitive_EmptyPasses(t *testing.T) {
	svc := RCService{dfa: engine.New()}
	resp, err := svc.CheckSensitive(context.Background(), 0, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Passed {
		t.Error("expected passed when no words loaded")
	}
}

func TestCheckSensitive_Hit(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "bad", Strategy: engine.SensitiveStrategyBlock},
	})
	svc := RCService{dfa: e}
	resp, err := svc.CheckSensitive(context.Background(), 0, "this is bad")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Passed {
		t.Error("expected blocked")
	}
	if !resp.Blocked {
		t.Error("expected blocked=true")
	}
	if len(resp.HitWords) == 0 || resp.HitWords[0] != "bad" {
		t.Errorf("expected hit 'bad', got %v", resp.HitWords)
	}
}

func TestCheckSensitive_Replace(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "bad", Strategy: engine.SensitiveStrategyReplace, Replacement: "***"},
	})
	svc := RCService{dfa: e}
	resp, err := svc.CheckSensitive(context.Background(), 0, "this is bad")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Passed {
		t.Error("expected not passed (blocked by replace + block mix, bad triggers blocked=false but all strategies are replace)")
	}
	// With only replace strategy, blocked should be false
	if resp.Blocked {
		t.Error("expected blocked=false for replace strategy")
	}
	if resp.Cleaned != "this is ***" {
		t.Errorf("expected 'this is ***', got %q", resp.Cleaned)
	}
}

func TestCheckSensitive_BlockTakesPriority(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "bad", Strategy: engine.SensitiveStrategyReplace, Replacement: "***"},
		{Word: "evil", Strategy: engine.SensitiveStrategyBlock},
	})
	svc := RCService{dfa: e}
	resp, err := svc.CheckSensitive(context.Background(), 0, "bad evil")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Passed {
		t.Error("expected blocked when any word has block strategy")
	}
	if !resp.Blocked {
		t.Error("expected blocked=true")
	}
}

func TestCheckSensitive_LogOnly(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "spam", Strategy: engine.SensitiveStrategyLog},
	})
	svc := RCService{dfa: e}
	resp, err := svc.CheckSensitive(context.Background(), 0, "spam message")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Passed {
		t.Error("expected passed for log-only strategy")
	}
	if resp.Blocked {
		t.Error("expected blocked=false for log strategy")
	}
}

func TestCheckChain_EmptyContent(t *testing.T) {
	svc := RCService{dfa: engine.New()}
	resp := svc.CheckChain(context.Background(), 0, "", 0, 0, "", 0)
	if !resp.Passed {
		t.Error("expected chain to pass with empty inputs")
	}
}

func TestCheckChain_SensitiveOnly(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "bad", Strategy: engine.SensitiveStrategyBlock},
	})
	svc := RCService{dfa: e}
	resp := svc.CheckChain(context.Background(), 0, "bad", 0, 0, "", 0)
	if resp.Passed {
		t.Error("expected chain to fail when sensitive check blocks")
	}
	if resp.SensitiveCheck == nil || !resp.SensitiveCheck.Blocked {
		t.Error("expected sensitive check to block")
	}
}

func TestCheckSensitive_MultipleHits(t *testing.T) {
	e := engine.New()
	e.Build([]engine.WordEntry{
		{Word: "bad", Strategy: engine.SensitiveStrategyReplace, Replacement: "***"},
		{Word: "evil", Strategy: engine.SensitiveStrategyBlock},
	})
	svc := RCService{dfa: e}
	resp, err := svc.CheckSensitive(context.Background(), 0, "bad and evil words")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.HitWords) < 2 {
		t.Errorf("expected at least 2 hit words, got %v", resp.HitWords)
	}
}

func TestCheckSensitive_EmptyContent(t *testing.T) {
	svc := RCService{dfa: engine.New()}
	resp, err := svc.CheckSensitive(context.Background(), 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Passed {
		t.Error("expected passed for empty content")
	}
}
