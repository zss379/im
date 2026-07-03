package engine

import (
	"testing"
)

func TestDFAEngine(t *testing.T) {
	engine := New()
	engine.Build([]WordEntry{
		{Word: "bad", Strategy: SensitiveStrategyReplace, Replacement: "***"},
		{Word: "evil", Strategy: SensitiveStrategyBlock},
		{Word: "spam", Strategy: SensitiveStrategyLog},
	})

	t.Run("Check no match", func(t *testing.T) {
		result := engine.Check("hello world")
		if result.HasMatch {
			t.Error("expected no match")
		}
	})

	t.Run("Check replace match", func(t *testing.T) {
		result := engine.Check("this is bad")
		if !result.HasMatch {
			t.Fatal("expected match")
		}
		if len(result.Words) != 1 || result.Words[0].Word != "bad" {
			t.Errorf("unexpected match: %+v", result.Words)
		}
		if result.Words[0].Strategy != SensitiveStrategyReplace {
			t.Errorf("expected replace strategy, got %d", result.Words[0].Strategy)
		}
	})

	t.Run("Check block match", func(t *testing.T) {
		result := engine.Check("evil person")
		if !result.HasMatch {
			t.Fatal("expected match")
		}
		if len(result.Words) != 1 || result.Words[0].Word != "evil" {
			t.Errorf("unexpected match: %+v", result.Words)
		}
		if result.Words[0].Strategy != SensitiveStrategyBlock {
			t.Errorf("expected block strategy, got %d", result.Words[0].Strategy)
		}
	})

	t.Run("Check case insensitive", func(t *testing.T) {
		result := engine.Check("THIS IS BAD")
		if !result.HasMatch {
			t.Error("expected case-insensitive match")
		}
	})

	t.Run("Check multiple hits", func(t *testing.T) {
		result := engine.Check("bad evil spam")
		if !result.HasMatch {
			t.Fatal("expected match")
		}
		if len(result.Words) < 2 {
			t.Errorf("expected at least 2 matches, got %d", len(result.Words))
		}
	})

	t.Run("Replace strategy", func(t *testing.T) {
		cleaned, blocked := engine.Replace("this is bad")
		if blocked {
			t.Error("expected not blocked")
		}
		if cleaned != "this is ***" {
			t.Errorf("unexpected cleaned: %q", cleaned)
		}
	})

	t.Run("Block strategy returns blocked", func(t *testing.T) {
		_, blocked := engine.Replace("evil person")
		if !blocked {
			t.Error("expected blocked")
		}
	})

	t.Run("Log strategy no modification", func(t *testing.T) {
		cleaned, blocked := engine.Replace("spam message")
		if blocked {
			t.Error("expected not blocked")
		}
		if cleaned != "spam message" {
			t.Errorf("expected no change, got %q", cleaned)
		}
	})

	t.Run("No match returns original", func(t *testing.T) {
		cleaned, blocked := engine.Replace("clean text")
		if blocked {
			t.Error("expected not blocked")
		}
		if cleaned != "clean text" {
			t.Errorf("expected original, got %q", cleaned)
		}
	})

	t.Run("Empty text", func(t *testing.T) {
		result := engine.Check("")
		if result.HasMatch {
			t.Error("expected no match for empty text")
		}
		cleaned, blocked := engine.Replace("")
		if blocked || cleaned != "" {
			t.Error("unexpected result for empty text")
		}
	})

	t.Run("Empty entries are skipped", func(t *testing.T) {
		e2 := New()
		e2.Build([]WordEntry{{Word: "", Strategy: SensitiveStrategyReplace, Replacement: "x"}})
		if e2.WordCount() != 0 {
			t.Error("empty word should be skipped")
		}
	})

	t.Run("Longest match wins", func(t *testing.T) {
		e2 := New()
		e2.Build([]WordEntry{
			{Word: "abc", Strategy: SensitiveStrategyReplace, Replacement: "x"},
			{Word: "abcdef", Strategy: SensitiveStrategyReplace, Replacement: "y"},
		})
		result := e2.Check("abcdef")
		if !result.HasMatch || len(result.Words) == 0 {
			t.Fatal("expected match")
		}
		// longest match should be the last one that matched
		if result.Words[0].Word != "abcdef" {
			t.Errorf("expected longest match 'abcdef', got %q", result.Words[0].Word)
		}
	})

	t.Run("WordCount", func(t *testing.T) {
		if engine.WordCount() != 3 {
			t.Errorf("expected 3 words, got %d", engine.WordCount())
		}
	})

	t.Run("Rebuild replaces trie", func(t *testing.T) {
		engine.Build([]WordEntry{{Word: "new", Strategy: SensitiveStrategyReplace, Replacement: "x"}})
		if engine.WordCount() != 1 {
			t.Errorf("expected 1 word after rebuild, got %d", engine.WordCount())
		}
		result := engine.Check("bad")
		if result.HasMatch {
			t.Error("expected no match for old word after rebuild")
		}
		result = engine.Check("new")
		if !result.HasMatch {
			t.Error("expected match for new word after rebuild")
		}
	})
}
