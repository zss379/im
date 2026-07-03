package engine

import (
	"strings"
	"sync"
)

// DFA sensitive word matching engine.
// Builds a trie (prefix tree) for O(n) matching.

type (
	// DFAEngine is a goroutine-safe DFA matcher for sensitive words.
	DFAEngine struct {
		mu   sync.RWMutex
		root *node
	}

	node struct {
		children map[rune]*node
		isEnd    bool
		strategy int8
		replacement string
	}
)

func New() *DFAEngine {
	return &DFAEngine{root: &node{children: make(map[rune]*node)}}
}

// Build rebuilds the trie from a word list. Each entry is:
//   word, strategy (1=replace,2=block,3=log), replacement string
// This is called on startup and whenever the word list is updated.
func (e *DFAEngine) Build(words []WordEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.root = &node{children: make(map[rune]*node)}
	for _, w := range words {
		if w.Word == "" {
			continue
		}
		e.insert(w.Word, w.Strategy, w.Replacement)
	}
}

func (e *DFAEngine) insert(word string, strategy int8, replacement string) {
	cur := e.root
	for _, r := range []rune(strings.ToLower(word)) {
		next, ok := cur.children[r]
		if !ok {
			next = &node{children: make(map[rune]*node)}
			cur.children[r] = next
		}
		cur = next
	}
	cur.isEnd = true
	cur.strategy = strategy
	cur.replacement = replacement
}

// WordEntry represents a word with its strategy configuration.
type WordEntry struct {
	Word        string
	Strategy    int8
	Replacement string
}

// MatchResult contains information about matched sensitive words.
type MatchResult struct {
	HasMatch bool
	Words    []HitWord
}

// HitWord represents a single matched sensitive word.
type HitWord struct {
	Word        string
	Strategy    int8
	Replacement string
	Start       int // rune start index in original text
	End         int // rune end index in original text
}

// Check scans text and returns all matched sensitive words.
func (e *DFAEngine) Check(text string) *MatchResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	runes := []rune(text)
	var hits []HitWord
	length := len(runes)

	for i := 0; i < length; i++ {
		cur := e.root
		matchedIdx := -1
		matchedEnd := -1
		var matchedWord string

		for j := i; j < length; j++ {
			next, ok := cur.children[toLowerRune(runes[j])]
			if !ok {
				break
			}
			cur = next
			if cur.isEnd {
				matchedIdx = i
				matchedEnd = j
				matchedWord = string(runes[i : j+1])
			}
		}

		if matchedIdx >= 0 {
			hits = append(hits, HitWord{
				Word:        matchedWord,
				Strategy:    cur.strategy,
				Replacement: cur.replacement,
				Start:       matchedIdx,
				End:         matchedEnd,
			})
			// skip to end of match for overlapping detection
			// use longest match
		}
	}

	return &MatchResult{
		HasMatch: len(hits) > 0,
		Words:    hits,
	}
}

// Replace replaces matched words according to their strategy.
// For strategy=1 (replace): replaces with configured replacement.
// For strategy=2 (block): returns the text and indicates it should be blocked.
// For strategy=3 (log): no modification.
func (e *DFAEngine) Replace(text string) (cleaned string, blocked bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	runes := []rune(text)
	length := len(runes)
	replacements := make(map[int]string) // start index → replacement text
	blocked = false

	for i := 0; i < length; i++ {
		cur := e.root
		matchedEnd := -1
		var matchedRunes int
		var strategy int8
		var replacement string

		for j := i; j < length; j++ {
			next, ok := cur.children[toLowerRune(runes[j])]
			if !ok {
				break
			}
			cur = next
			if cur.isEnd {
				matchedEnd = j
				matchedRunes = j - i + 1
				strategy = cur.strategy
				replacement = cur.replacement
			}
		}

		if matchedEnd >= 0 {
			switch strategy {
			case SensitiveStrategyBlock:
				blocked = true
				return "", true
			case SensitiveStrategyReplace:
				if replacement == "" {
					replacement = "***"
				}
				replacements[i] = replacement
				i = matchedEnd
			case SensitiveStrategyLog:
				// no modification, just log
				i = matchedEnd
			}
		}
	}

	if len(replacements) == 0 {
		return text, false
	}

	var buf strings.Builder
	i := 0
	for i < length {
		if r, ok := replacements[i]; ok {
			buf.WriteString(r)
			// skip the matched runes
			for j := i + 1; j < length; j++ {
				if _, found := replacements[j]; found {
					break
				}
				// check if this position is within a matched range
				if _, inNext := replacements[j+1]; inNext {
					continue
				}
			}
			// simple approach: just skip ahead
			// find how long this match was
			skip := 1
			for k := i + 1; k < length; k++ {
				if _, isStart := replacements[k]; isStart {
					break
				}
				skip++
			}
			i += skip
		} else {
			buf.WriteRune(runes[i])
			i++
		}
	}

	return buf.String(), false
}

func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

// Strategy constants (duplicated locally to avoid import cycle)
const (
	SensitiveStrategyReplace = 1
	SensitiveStrategyBlock   = 2
	SensitiveStrategyLog     = 3
)

// WordCount returns the number of words in the trie.
func (e *DFAEngine) WordCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.countNodes(e.root, false)
}

func (e *DFAEngine) countNodes(n *node, counted bool) int {
	count := 0
	if n.isEnd && !counted {
		count++
		counted = true
	}
	for _, child := range n.children {
		count += e.countNodes(child, counted)
	}
	return count
}
