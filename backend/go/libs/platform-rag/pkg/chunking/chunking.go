package chunking

import "strings"

// Options controls how text is chunked before embedding.
type Options struct {
	MaxChars            int  // target maximum characters per chunk
	Overlap             int  // number of chars to overlap between consecutive chunks
	NormalizeWhitespace bool // collapse repeated whitespace into single spaces
	TrimSpace           bool // trim leading/trailing whitespace in each chunk
	MinCharsForBoundary int  // minimum chars before allowing a boundary fallback
}

// Segment represents a slice of text with its original offsets.
type Segment struct {
	Text  string
	Start int
	End   int
}

const (
	defaultMaxChars            = 1200
	defaultOverlap             = 80
	defaultMinCharsForBoundary = 120
)

// ChunkText splits the input into segments respecting max length and overlap.
func ChunkText(input string, opt Options) []Segment {
	if input == "" {
		return nil
	}

	cfg := normalizeOptions(opt)
	text := input
	if cfg.NormalizeWhitespace {
		text = strings.Join(strings.Fields(text), " ")
	}
	if cfg.TrimSpace {
		text = strings.TrimSpace(text)
	}
	if text == "" {
		return nil
	}

	var segments []Segment
	start := 0
	for start < len(text) {
		end := start + cfg.MaxChars
		if end > len(text) {
			end = len(text)
		}

		// Try to cut at a natural boundary inside the window.
		cut := chooseBoundary(text, start, end, cfg.MinCharsForBoundary)

		chunk := text[start:cut]
		if cfg.TrimSpace {
			chunk = strings.TrimSpace(chunk)
		}
		segments = append(segments, Segment{Text: chunk, Start: start, End: cut})

		if cut == len(text) {
			break
		}

		next := cut - cfg.Overlap
		if next <= start {
			next = cut
		}
		start = next
	}

	return segments
}

func chooseBoundary(text string, start, end, minBoundary int) int {
	if end >= len(text) {
		return len(text)
	}
	window := text[start:end]
	if idx := lastBoundary(window, "\n.?!", minBoundary); idx >= 0 {
		return start + idx
	}
	if idx := lastBoundary(window, " ", minBoundary); idx >= 0 {
		return start + idx
	}
	return end
}

func lastBoundary(window string, chars string, minBoundary int) int {
	idx := strings.LastIndexAny(window, chars)
	if idx >= 0 && idx+1 >= minBoundary {
		return idx + 1
	}
	return -1
}

func normalizeOptions(opt Options) Options {
	if opt.MaxChars <= 0 {
		opt.MaxChars = defaultMaxChars
	}
	if opt.Overlap < 0 {
		opt.Overlap = 0
	}
	if opt.Overlap >= opt.MaxChars {
		opt.Overlap = opt.MaxChars / 4
	}
	if opt.MinCharsForBoundary <= 0 {
		opt.MinCharsForBoundary = defaultMinCharsForBoundary
	}
	return opt
}
