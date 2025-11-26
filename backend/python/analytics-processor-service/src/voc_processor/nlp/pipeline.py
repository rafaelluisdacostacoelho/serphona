import json
from typing import List, Dict, Any
from datetime import datetime
from hashlib import sha256


class SimpleCache:
    def __init__(self):
        self._cache = {}

    def get(self, key):
        return self._cache.get(key)

    def set(self, key, value):
        self._cache[key] = value


cache = SimpleCache()


def rule_based_sentiment(text: str) -> float:
    pos = ["good", "great", "thank", "satisfied"]
    neg = ["bad", "angry", "hate", "not happy", "frustrat"]
    lc = text.lower()
    score = sum(1 for w in pos if w in lc) - sum(1 for w in neg if w in lc)
    return float(score)


def call_model_batch_sync(texts: List[str]) -> List[float]:
    # Placeholder synchronous inference (fast heuristic). Replace with model call or remote inference.
    return [rule_based_sentiment(t) for t in texts]


async def annotate_nlp(events: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    texts = []
    idxs = []
    for i, e in enumerate(events):
        text = (e.get("transcript") or "")[:20000]
        if not text:
            e["sentiment"] = None
            e["topic"] = None
            e["processed_at"] = datetime.utcnow()
            e["payload"] = e.get("payload", e)
            continue
        key = sha256(text.encode()).hexdigest()
        cached = cache.get(key)
        if cached:
            e["sentiment"] = cached.get("sentiment")
            e["topic"] = cached.get("topic")
            e["processed_at"] = datetime.utcnow()
            e["payload"] = e.get("payload", e)
        else:
            # quick rule to avoid model calls
            score = rule_based_sentiment(text)
            if abs(score) >= 1.0:
                e["sentiment"] = score
                e["topic"] = None
                cache.set(key, {"sentiment": score, "topic": None})
                e["processed_at"] = datetime.utcnow()
                e["payload"] = e.get("payload", e)
            else:
                texts.append(text)
                idxs.append((i, key))

    if texts:
        # synchronous placeholder; for production, implement a batched async model call
        model_scores = call_model_batch_sync(texts)
        for (i, key), score in zip(idxs, model_scores):
            events[i]["sentiment"] = score
            events[i]["topic"] = None
            events[i]["processed_at"] = datetime.utcnow()
            events[i]["payload"] = events[i].get("payload", events[i])
            cache.set(key, {"sentiment": score, "topic": None})

    # ensure payload is set for storage
    for e in events:
        if "payload" not in e:
            e["payload"] = e
    return events
