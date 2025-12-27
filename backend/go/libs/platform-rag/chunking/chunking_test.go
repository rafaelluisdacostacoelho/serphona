package chunking

import (
	"strings"
	"testing"
)

func TestChunkTextRespectsMaxAndOverlap(t *testing.T) {
	text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat."
	segs := ChunkText(text, Options{MaxChars: 80, Overlap: 10, NormalizeWhitespace: true, TrimSpace: true})

	if len(segs) < 2 {
		t.Fatalf("expected multiple segments, got %d", len(segs))
	}

	for i, s := range segs {
		if len(s.Text) > 80 {
			t.Fatalf("segment %d exceeds max: %d", i, len(s.Text))
		}
	}

	if segs[1].Start >= segs[0].End {
		t.Fatalf("expected overlap between first segments")
	}
}

func TestChunkTextDefaultsAndBoundaries(t *testing.T) {
	text := strings.Repeat("Sentence boundary. ", 200) // ~4k chars
	segs := ChunkText(text, DefaultOptions())

	if len(segs) == 0 {
		t.Fatalf("expected segments")
	}

	for i, s := range segs {
		if len(s.Text) > defaultMaxChars {
			t.Fatalf("segment %d exceeds default max: %d", i, len(s.Text))
		}
		if i > 0 && s.Start >= segs[i-1].End {
			t.Fatalf("expected overlap at segment %d", i)
		}
	}
}

func TestChunkTextLanguageAwareBoundary(t *testing.T) {
	text := "你好世界。再见世界。"
	opts := DefaultOptions()
	opts.MaxChars = 15
	opts.Overlap = 0
	opts.LanguageCode = "zh"
	segs := ChunkText(text, opts)
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments for zh punctuation, got %d", len(segs))
	}
	if !strings.HasSuffix(segs[0].Text, "。") {
		t.Fatalf("expected first segment to end with zh period, got %q", segs[0].Text)
	}
}

func TestChunkTextMaxSegmentsCap(t *testing.T) {
	text := strings.Repeat("0123456789", 2000) // ~20k chars
	opts := DefaultOptions()
	opts.MaxChars = 50
	opts.Overlap = 0
	opts.MaxSegments = 10
	segs := ChunkText(text, opts)
	if len(segs) != 10 {
		t.Fatalf("expected 10 segments cap, got %d", len(segs))
	}
}

func TestChunkTextUsesNaturalBoundary(t *testing.T) {
	text := "Sentence one is here. Sentence two follows. Sentence three closes."
	segs := ChunkText(text, Options{MaxChars: 35, Overlap: 0, NormalizeWhitespace: true, TrimSpace: true, MinCharsForBoundary: 10})

	if len(segs) < 2 {
		t.Fatalf("expected at least 2 segments, got %d", len(segs))
	}

	if segs[0].Text[len(segs[0].Text)-1] != '.' {
		t.Fatalf("expected first segment to end at period: %q", segs[0].Text)
	}
}

func TestChunkTextHandlesLargeOverlap(t *testing.T) {
	text := "hi world"
	opts := Options{MaxChars: 5, Overlap: 5, NormalizeWhitespace: false, TrimSpace: false, MinCharsForBoundary: 1}
	segs := ChunkText(text, opts)
	if len(segs) < 2 {
		t.Fatalf("expected multiple segments with large overlap, got %d", len(segs))
	}
	advancedByCut := false
	for i := 1; i < len(segs); i++ {
		if segs[i].Start == segs[i-1].End {
			advancedByCut = true
			break
		}
	}
	if !advancedByCut {
		t.Fatalf("expected overlap branch to advance by cut, got segments %+v", segs)
	}
}

func TestChunkTextHandlesEmpty(t *testing.T) {
	if segs := ChunkText("", Options{}); segs != nil {
		t.Fatalf("expected nil for empty input")
	}

	if segs := ChunkText("   ", Options{NormalizeWhitespace: true, TrimSpace: true}); segs != nil {
		t.Fatalf("expected nil for whitespace-only input")
	}
}

func TestNormalizeOptionsAdjustsBounds(t *testing.T) {
	opts := normalizeOptions(Options{MaxChars: 0, Overlap: -5, MinCharsForBoundary: 0})
	if opts.MaxChars != defaultMaxChars || opts.Overlap != 0 || opts.MinCharsForBoundary != defaultMinCharsForBoundary {
		t.Fatalf("expected defaults applied, got %+v", opts)
	}

	opts = normalizeOptions(Options{MaxChars: 100, Overlap: 200, MinCharsForBoundary: 10})
	if opts.Overlap >= opts.MaxChars {
		t.Fatalf("overlap should have been reduced below max, got %d", opts.Overlap)
	}
}

func TestChooseBoundaryFallbacks(t *testing.T) {
	if cut := chooseBoundary("abc defghi", 0, 7, 0, ""); cut != 4 {
		t.Fatalf("expected space fallback boundary at 4, got %d", cut)
	}
	if cut := chooseBoundary("abcdefghij", 0, 5, 10, ""); cut != 5 {
		t.Fatalf("expected hard end fallback at 5, got %d", cut)
	}
}

func BenchmarkChunkTextSmall(b *testing.B) {
	text := "The quick brown fox jumps over the lazy dog."
	for n := 0; n < b.N; n++ {
		_ = ChunkText(text, DefaultOptions())
	}
}

func BenchmarkChunkTextLarge(b *testing.B) {
	text := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 1000)
	for n := 0; n < b.N; n++ {
		_ = ChunkText(text, DefaultOptions())
	}
}
