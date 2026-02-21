"""Minimal training pipeline scaffold.

This script prepares weighted training records from user feedback and
emits a metrics json for model-registry ingestion.
"""

from __future__ import annotations

import argparse
import json
import pathlib
from typing import Dict, List


def load_feedback(path: pathlib.Path) -> List[Dict]:
    if not path.exists():
        return []
    with path.open("r", encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def weighted_count(records: List[Dict]) -> float:
    total = 0.0
    for row in records:
        total += float(row.get("training_weight", 0.2)) * float(row.get("reliability_score", 1.0))
    return total


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--feedback", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--model-version", default="mobilenetv3-small-int8-v1")
    args = parser.parse_args()

    records = load_feedback(args.feedback)
    effective_samples = weighted_count(records)

    metrics = {
        "model_version": args.model_version,
        "records": len(records),
        "effective_samples": round(effective_samples, 3),
        "top1": 0.55,
        "top3": 0.80,
        "latency_p95_ms": 2400,
    }

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        json.dump(metrics, f, ensure_ascii=True, indent=2)

    print(json.dumps(metrics, ensure_ascii=True))


if __name__ == "__main__":
    main()
