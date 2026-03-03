package model

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

// HistoryEntry represents a single prompt from ~/.claude/history.jsonl.
type HistoryEntry struct {
	Timestamp time.Time
	Project   string
	SessionID string
	Prompt    string
}

// WordCount holds a word and its frequency count.
type WordCount struct {
	Word  string
	Count int
}

// HistoryStats holds aggregated statistics from history entries.
type HistoryStats struct {
	TotalPrompts  int
	ActiveDays    map[string]bool // "2006-01-02" -> true
	PromptsByDate map[string]int  // "2006-01-02" -> count
	HourCounts    map[int]int     // hour (0-23) -> count
	Heatmap        [7][24]int // day-of-week (Mon=0..Sun=6) x hour
	TopWords       []WordCount
	AvgPromptWords float64 // average words per prompt
	P95PromptWords int     // 95th percentile words per prompt
}

// stopWords is the set of common words to exclude from top words analysis.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
	"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "can": true, "may": true, "might": true, "shall": true,
	"to": true, "of": true, "in": true, "for": true, "on": true, "with": true,
	"at": true, "by": true, "from": true, "as": true, "into": true, "through": true,
	"about": true, "it": true, "i": true, "me": true, "my": true, "we": true,
	"our": true, "you": true, "your": true, "this": true, "that": true, "and": true,
	"or": true, "but": true, "not": true, "no": true, "if": true, "then": true,
	"so": true, "up": true, "out": true, "just": true, "also": true, "how": true,
	"what": true, "when": true, "where": true, "which": true, "who": true, "why": true,
	"all": true, "each": true, "some": true, "any": true, "let": true, "please": true,
	"make": true, "use": true, "add": true, "get": true, "set": true, "new": true,
	"like": true, "want": true, "need": true, "file": true, "code": true, "using": true,
	"here": true, "don": true, "there": true,
}

// ComputeHistoryStats aggregates a slice of history entries into stats.
func ComputeHistoryStats(entries []HistoryEntry) HistoryStats {
	stats := HistoryStats{
		ActiveDays:    make(map[string]bool),
		PromptsByDate: make(map[string]int),
		HourCounts:    make(map[int]int),
	}
	wordFreqs := make(map[string]int)
	var promptLengths []int
	for _, e := range entries {
		stats.TotalPrompts++
		date := e.Timestamp.Format("2006-01-02")
		stats.ActiveDays[date] = true
		stats.PromptsByDate[date]++
		hour := e.Timestamp.Hour()
		stats.HourCounts[hour]++
		weekday := e.Timestamp.Weekday()
		day := int(weekday) - 1
		if day < 0 {
			day = 6
		}
		stats.Heatmap[day][hour]++

		// Tokenize prompt for word frequency and length stats
		if e.Prompt != "" {
			allWords := strings.Fields(e.Prompt)
			promptLengths = append(promptLengths, len(allWords))
			for _, raw := range allWords {
				w := strings.TrimFunc(strings.ToLower(raw), func(r rune) bool {
					return !unicode.IsLetter(r)
				})
				if len(w) >= 3 && !stopWords[w] {
					wordFreqs[w]++
				}
			}
		}
	}

	// Compute prompt length stats
	if len(promptLengths) > 0 {
		total := 0
		for _, l := range promptLengths {
			total += l
		}
		stats.AvgPromptWords = math.Round(float64(total)/float64(len(promptLengths))*10) / 10

		sort.Ints(promptLengths)
		p95Idx := int(math.Ceil(0.95*float64(len(promptLengths)))) - 1
		if p95Idx >= len(promptLengths) {
			p95Idx = len(promptLengths) - 1
		}
		if p95Idx < 0 {
			p95Idx = 0
		}
		stats.P95PromptWords = promptLengths[p95Idx]
	}

	// Build top 5 words
	if len(wordFreqs) > 0 {
		type wc struct {
			word  string
			count int
		}
		var sorted []wc
		for w, c := range wordFreqs {
			sorted = append(sorted, wc{w, c})
		}
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].count != sorted[j].count {
				return sorted[i].count > sorted[j].count
			}
			return sorted[i].word < sorted[j].word
		})
		n := 5
		if len(sorted) < n {
			n = len(sorted)
		}
		stats.TopWords = make([]WordCount, n)
		for i := 0; i < n; i++ {
			stats.TopWords[i] = WordCount{Word: sorted[i].word, Count: sorted[i].count}
		}
	}

	return stats
}
