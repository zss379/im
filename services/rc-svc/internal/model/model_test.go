package model

import (
	"encoding/json"
	"testing"
)

func TestSensitiveCheckResp_JSON(t *testing.T) {
	resp := &SensitiveCheckResp{
		Passed:   false,
		HitWords: []string{"bad", "evil"},
		Cleaned:  "*** word",
		Blocked:  true,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded SensitiveCheckResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Passed != resp.Passed || decoded.Blocked != resp.Blocked {
		t.Error("round-trip mismatch")
	}
	if len(decoded.HitWords) != 2 {
		t.Error("expected 2 hit words")
	}
}

func TestRateLimitCheckResp_JSON(t *testing.T) {
	resp := &RateLimitCheckResp{Passed: true, Remaining: 5}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded RateLimitCheckResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Passed != true || decoded.Remaining != 5 {
		t.Error("round-trip mismatch")
	}
}

func TestFileLimitCheckResp_JSON(t *testing.T) {
	resp := &FileLimitCheckResp{Passed: false, MaxSizeMB: 10, Extensions: "jpg,png"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded FileLimitCheckResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Passed != false || decoded.MaxSizeMB != 10 || decoded.Extensions != "jpg,png" {
		t.Error("round-trip mismatch")
	}
}

func TestCheckChainResp_JSON(t *testing.T) {
	resp := &CheckChainResp{
		Passed: true,
		SensitiveCheck: &SensitiveCheckResp{
			Passed:  true,
			Cleaned: "clean text",
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded CheckChainResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Passed != true {
		t.Error("passed mismatch")
	}
	if decoded.SensitiveCheck == nil || !decoded.SensitiveCheck.Passed {
		t.Error("sensitive check mismatch")
	}
}

func TestCheckChainResp_NilChecks(t *testing.T) {
	resp := &CheckChainResp{Passed: true}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded CheckChainResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SensitiveCheck != nil {
		t.Error("expected nil sensitive check")
	}
}

func TestSensitiveWordCreateReq_Validation(t *testing.T) {
	req := &SensitiveWordCreateReq{Word: "bad", Strategy: SensitiveStrategyBlock}
	if req.Word != "bad" || req.Strategy != 2 {
		t.Error("field mismatch")
	}
}

func TestSensitiveWordBatchReq(t *testing.T) {
	req := &SensitiveWordBatchReq{
		Words: []SensitiveWordInput{
			{Word: "word1", Strategy: 1},
			{Word: "word2", Strategy: 2},
		},
	}
	if len(req.Words) != 2 {
		t.Error("expected 2 words")
	}
}

func TestRateLimitRuleCreateReq(t *testing.T) {
	req := &RateLimitRuleCreateReq{
		TargetType:        1,
		MaxCount:          5,
		TimeWindowSeconds: 1,
	}
	if req.TargetType != 1 || req.MaxCount != 5 || req.TimeWindowSeconds != 1 {
		t.Error("field mismatch")
	}
}

func TestFileLimitCreateReq(t *testing.T) {
	req := &FileLimitCreateReq{
		FileType:          "image",
		MaxSizeMB:         10,
		AllowedExtensions: "jpg,png",
	}
	if req.FileType != "image" || req.MaxSizeMB != 10 {
		t.Error("field mismatch")
	}
}

func TestFileLimitCheckReq(t *testing.T) {
	req := &FileLimitCheckReq{FileType: "video", FileSize: 104857600}
	if req.FileType != "video" || req.FileSize != 104857600 {
		t.Error("field mismatch")
	}
}
