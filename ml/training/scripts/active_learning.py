"""Active learning sampler for daily retraining.

Sampling ratio:
- 40% low confidence
- 40% prediction-user conflict
- 20% novel scene/breed
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable, List


@dataclass
class Candidate:
    sample_id: str
    confidence: float
    is_conflict: bool
    is_novel: bool


def select_candidates(pool: Iterable[Candidate], daily_budget: int) -> List[Candidate]:
    pool_list = list(pool)
    if daily_budget <= 0:
        return []

    low_conf = sorted(pool_list, key=lambda x: x.confidence)
    conflict = [c for c in pool_list if c.is_conflict]
    novel = [c for c in pool_list if c.is_novel]

    n_low = int(daily_budget * 0.4)
    n_conflict = int(daily_budget * 0.4)
    n_novel = daily_budget - n_low - n_conflict

    selected: List[Candidate] = []
    selected_ids = set()

    def take(items: List[Candidate], n: int) -> None:
        for item in items:
            if n <= 0 or len(selected) >= daily_budget:
                return
            if item.sample_id in selected_ids:
                continue
            selected.append(item)
            selected_ids.add(item.sample_id)
            n -= 1

    take(low_conf, n_low)
    take(conflict, n_conflict)
    take(novel, n_novel)

    if len(selected) < daily_budget:
        for item in low_conf:
            if item.sample_id in selected_ids:
                continue
            selected.append(item)
            selected_ids.add(item.sample_id)
            if len(selected) >= daily_budget:
                break

    return selected
