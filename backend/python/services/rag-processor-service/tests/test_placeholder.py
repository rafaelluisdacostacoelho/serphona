from rag_processor import main


def test_main_imports():
    # Smoke test to ensure the scaffold is importable
    assert callable(main.main)
