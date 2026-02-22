"""Generate daily active-learning tasks with 40/40/20 strategy."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import pathlib
from dataclasses import dataclass
from typing import Dict, Iterable, List, Tuple


@dataclass
class PoolRow:
    sample_id: str
    confidence: float
    is_conflict: bool
    is_novel: bool


def read_pool(path: pathlib.Path) -> List[PoolRow]:
    rows: List[PoolRow] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            item = json.loads(line)
            sample_id = str(item.get("sample_id", "")).strip()
            if not sample_id:
                continue
            rows.append(
                PoolRow(
                    sample_id=sample_id,
                    confidence=float(item.get("confidence", 1.0)),
                    is_conflict=bool(item.get("is_conflict", False)),
                    is_novel=bool(item.get("is_novel", False)),
                )
            )
    return rows


def write_json(path: pathlib.Path, payload: Dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        json.dump(payload, f, ensure_ascii=True, indent=2)


def select_with_reasons(pool: Iterable[PoolRow], daily_budget: int) -> Tuple[List[Dict], Dict]:
    items = list(pool)
    low_conf = sorted(items, key=lambda x: x.confidence)
    conflict = [r for r in items if r.is_conflict]
    novel = [r for r in items if r.is_novel]

    n_low = int(daily_budget * 0.4)
    n_conflict = int(daily_budget * 0.4)
    n_novel = daily_budget - n_low - n_conflict

    picked: List[Dict] = []
    seen = set()

    def take(rows: List[PoolRow], count: int, reason: str) -> None:
        for row in rows:
            if count <= 0 or len(picked) >= daily_budget:
                break
            if row.sample_id in seen:
                continue
            picked.append(
                {
                    "sample_id": row.sample_id,
                    "confidence": row.confidence,
                    "is_conflict": row.is_conflict,
                    "is_novel": row.is_novel,
                    "reason": reason,
                }
            )
            seen.add(row.sample_id)
            count -= 1

    take(low_conf, n_low, "LOW_CONFIDENCE")
    take(conflict, n_conflict, "PREDICTION_CONFLICT")
    take(novel, n_novel, "NOVEL_SCENE")

    if len(picked) < daily_budget:
        take(low_conf, daily_budget - len(picked), "LOW_CONFIDENCE_BACKFILL")

    reason_counts: Dict[str, int] = {}
    for row in picked:
        reason = row["reason"]
        reason_counts[reason] = reason_counts.get(reason, 0) + 1

    report = {
        "daily_budget": daily_budget,
        "selected": len(picked),
        "ratio_target": {"LOW_CONFIDENCE": 0.4, "PREDICTION_CONFLICT": 0.4, "NOVEL_SCENE": 0.2},
        "reason_counts": reason_counts,
    }
    return picked, report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pool", type=pathlib.Path, required=True, help="candidate pool JSONL")
    parser.add_argument("--daily-budget", type=int, default=100)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--date", default=dt.date.today().isoformat())
    args = parser.parse_args()

    pool_rows = read_pool(args.pool)
    selected, report = select_with_reasons(pool_rows, args.daily_budget)

    payload = {
        "task_date": args.date,
        "daily_budget": args.daily_budget,
        "tasks": [
            {
                "task_id": f"al_{args.date}_{idx + 1:04d}",
                **item,
            }
            for idx, item in enumerate(selected)
        ],
        "report": report,
    }
    write_json(args.output, payload)
    print(json.dumps({"task_count": len(selected), "output": str(args.output)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
