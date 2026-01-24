import asyncio
import pytest

import rag_indexer.fetcher as fetcher
from rag_indexer.fetcher import FetchError


class _FakeResp:
    def __init__(self, status: int, text: str = ""):
        self.status = status
        self._text = text

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb):
        return False

    async def text(self):
        await asyncio.sleep(0)  # allow context switch
        return self._text


class _FakeSession:
    def __init__(self, responses):
        self._responses = iter(responses)

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb):
        return False

    def get(self, *_args, **_kwargs):
        try:
            return next(self._responses)
        except StopIteration as exc:  # pragma: no cover - guard
            raise AssertionError("no more responses") from exc

    def post(self, *_args, **_kwargs):
        try:
            return next(self._responses)
        except StopIteration as exc:  # pragma: no cover
            raise AssertionError("no more responses") from exc


@pytest.mark.asyncio
async def test_rest_pagination_combines_bodies(monkeypatch):
    resps = [_FakeResp(200, "page1"), _FakeResp(200, "page2"), _FakeResp(200, "")]
    monkeypatch.setattr(fetcher.aiohttp, "ClientSession", lambda *args, **kwargs: _FakeSession(resps))
    text, etag = await fetcher._fetch_rest_paginated("https://api.example.com/items", {"rest_paginate": True})
    assert text == "page1\npage2"
    assert etag is None


@pytest.mark.asyncio
async def test_graphql_fetch(monkeypatch):
    resps = [_FakeResp(200, "{data}")]
    monkeypatch.setattr(fetcher.aiohttp, "ClientSession", lambda *args, **kwargs: _FakeSession(resps))
    text, etag = await fetcher._fetch_graphql(
        "https://api.example.com/graphql", {"graphql_query": "query { ping }", "graphql_variables": {"k": 1}}
    )
    assert text == "{data}"
    assert etag is None


@pytest.mark.asyncio
async def test_graphql_failure(monkeypatch):
    resps = [_FakeResp(500, "fail")]
    monkeypatch.setattr(fetcher.aiohttp, "ClientSession", lambda *args, **kwargs: _FakeSession(resps))
    with pytest.raises(FetchError):
        await fetcher._fetch_graphql("https://api.example.com/graphql", {"graphql_query": "query { ping }"})
