from typing import List, Tuple


def chunk_text(
    text: str,
    max_chars: int = 1200,
    overlap: int = 80,
    min_boundary: int = 120,
    normalize_whitespace: bool = True,
    trim_space: bool = True,
) -> List[Tuple[str, int, int]]:
    if not text:
        return []

    if normalize_whitespace:
        text = " ".join(text.split())
    if trim_space:
        text = text.strip()
    if not text:
        return []

    segments = []
    start = 0
    n = len(text)
    while start < n:
        end = min(start + max_chars, n)
        cut = _choose_boundary(text, start, end, min_boundary)
        chunk = text[start:cut]
        if trim_space:
            chunk = chunk.strip()
        segments.append((chunk, start, cut))
        if cut == n:
            break
        next_start = cut - overlap
        if next_start <= start:
            next_start = cut
        start = next_start
    return segments


def _choose_boundary(text: str, start: int, end: int, min_boundary: int) -> int:
    if end >= len(text):
        return len(text)
    window = text[start:end]
    for chars in ("\n.?!", " "):
        idx = -1
        for ch in chars:
            idx = max(idx, window.rfind(ch))
        if idx >= 0 and idx + 1 >= min_boundary:
            return start + idx + 1
    return end
