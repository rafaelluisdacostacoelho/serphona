package pagination

import "testing"

func TestNormalizeAndMeta(t *testing.T) {
	p := Params{Page: -1, PageSize: 500}
	n := p.Normalize()
	if n.Page != 1 || n.PageSize != 100 {
		t.Fatalf("normalize failed, got page=%d size=%d", n.Page, n.PageSize)
	}
	meta := ComputeMeta(250, Params{Page: 2, PageSize: 20})
	if meta.TotalPages != 13 {
		t.Fatalf("expected total pages 13, got %d", meta.TotalPages)
	}
	tmp := Params{Page: meta.Page, PageSize: meta.PageSize}
	if off := tmp.Offset(); off != 20 {
		t.Fatalf("expected offset 20, got %d", off)
	}
}
