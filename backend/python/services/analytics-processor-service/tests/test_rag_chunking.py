from rag_indexer.chunking import chunk_text


def test_chunk_text_basic():
    text = "Sentence one. Sentence two follows. Sentence three closes."
    segs = chunk_text(text, max_chars=35, overlap=5, min_boundary=10)
    assert len(segs) >= 2
    assert segs[0][0].endswith(".")


def test_chunk_text_respects_boundary_and_overlap():
    text = "A B C D E F G H I J K L M N O P"
    segs = chunk_text(text, max_chars=10, overlap=3, min_boundary=3)
    assert len(segs) >= 2
    assert segs[1][1] < segs[0][2]  # overlap exists
