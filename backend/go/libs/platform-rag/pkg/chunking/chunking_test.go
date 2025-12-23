package chunking

import "testing"

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

func TestChunkTextHandlesEmpty(t *testing.T) {
	if segs := ChunkText("", Options{}); segs != nil {
		t.Fatalf("expected nil for empty input")
	}

	if segs := ChunkText("   ", Options{NormalizeWhitespace: true, TrimSpace: true}); segs != nil {
		t.Fatalf("expected nil for whitespace-only input")
	}
}
