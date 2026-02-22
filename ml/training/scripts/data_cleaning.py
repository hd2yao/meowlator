"""Feedback data cleaning for v0.4 training pipeline.

Steps:
1. Validate label consistency.
2. Deduplicate by perceptual hash / image key / sample id.
3. Down-weight suspicious users.
4. Write cleaned JSONL + summary report.
"""

from __future__ import annotations

import argparse
import json
import pathlib
from collections import Counter, defaultdict
from typing import Dict, Iterable, List, Tuple

from constants import INTENT_LABELS


def read_jsonl(path: pathlib.Path) -> List[Dict]:
    rows: List[Dict] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
    return rows


def write_jsonl(path: pathlib.Path, rows: Iterable[Dict]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=True) + "\n")


def resolve_label(row: Dict) -> str:
    is_correct = bool(row.get("is_correct", False))
    true_label = row.get("true_label")
    if is_correct:
        return str(true_label or row.get("model_label") or row.get("label") or "")
    return str(true_label or "")


def dedup_key(row: Dict) -> str:
    for key in ("phash", "image_hash", "image_key", "sample_id"):
        value = str(row.get(key, "")).strip()
        if value:
            return value
    return ""


def mark_suspicious_users(rows: Iterable[Dict]) -> Dict[str, bool]:
    per_user_labels: Dict[str, Counter] = defaultdict(Counter)
    per_user_total: Counter = Counter()

    for row in rows:
        user_id = str(row.get("user_id", "")).strip()
        if not user_id:
            continue
        label = resolve_label(row)
        if label not in INTENT_LABELS:
            continue
        per_user_total[user_id] += 1
        per_user_labels[user_id][label] += 1

    suspicious: Dict[str, bool] = {}
    for user_id, total in per_user_total.items():
        top_label_count = per_user_labels[user_id].most_common(1)[0][1]
        dominant_ratio = top_label_count / max(total, 1)
        suspicious[user_id] = total >= 6 and dominant_ratio >= 0.95
    return suspicious


def clean_feedback_rows(rows: List[Dict], suspicious_scale: float = 0.5) -> Tuple[List[Dict], Dict]:
    suspicious_flags = mark_suspicious_users(rows)

    report = {
        "input_rows": len(rows),
        "kept_rows": 0,
        "dropped_invalid_label": 0,
        "dropped_missing_true_label": 0,
        "dropped_missing_key": 0,
        "dropped_duplicate": 0,
        "suspicious_users": sorted([uid for uid, v in suspicious_flags.items() if v]),
    }

    dedup_map: Dict[str, Dict] = {}
    for row in rows:
        is_correct = bool(row.get("is_correct", False))
        label = resolve_label(row)
        if not is_correct and not str(row.get("true_label", "")).strip():
            report["dropped_missing_true_label"] += 1
            continue
        if label not in INTENT_LABELS:
            report["dropped_invalid_label"] += 1
            continue

        key = dedup_key(row)
        if not key:
            report["dropped_missing_key"] += 1
            continue

        created_at = int(row.get("created_at", 0) or 0)
        candidate = {
            "sample_id": str(row.get("sample_id", "")).strip(),
            "user_id": str(row.get("user_id", "")).strip(),
            "image_key": str(row.get("image_key", "")).strip(),
            "scene_tag": str(row.get("scene_tag", "UNKNOWN")).strip() or "UNKNOWN",
            "label": label,
            "is_correct": is_correct,
            "training_weight": float(row.get("training_weight", 0.2)),
            "reliability_score": float(row.get("reliability_score", 1.0)),
            "created_at": created_at,
            "dedup_key": key,
        }

        user_id = candidate["user_id"]
        if user_id and suspicious_flags.get(user_id, False):
            candidate["reliability_score"] = round(candidate["reliability_score"] * suspicious_scale, 6)

        current = dedup_map.get(key)
        if current is not None:
            report["dropped_duplicate"] += 1
            if candidate["created_at"] >= current["created_at"]:
                dedup_map[key] = candidate
            continue
        dedup_map[key] = candidate

    cleaned = list(dedup_map.values())
    cleaned.sort(key=lambda x: (x["created_at"], x["sample_id"]))
    report["kept_rows"] = len(cleaned)
    report["suspicious_user_count"] = len(report["suspicious_users"])
    return cleaned, report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=pathlib.Path, required=True, help="raw feedback JSONL")
    parser.add_argument("--output", type=pathlib.Path, required=True, help="cleaned feedback JSONL")
    parser.add_argument("--report", type=pathlib.Path, required=True, help="cleaning report JSON")
    parser.add_argument("--suspicious-scale", type=float, default=0.5)
    args = parser.parse_args()

    rows = read_jsonl(args.input)
    cleaned, report = clean_feedback_rows(rows, suspicious_scale=args.suspicious_scale)
    write_jsonl(args.output, cleaned)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    with args.report.open("w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=True, indent=2)
    print(json.dumps({"cleaned_rows": len(cleaned), "report": str(args.report)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
