"""Evaluate intent metrics from prediction records."""

from __future__ import annotations

import argparse
import json
import pathlib
from collections import Counter, defaultdict
from typing import Dict, List


def read_jsonl(path: pathlib.Path) -> List[Dict]:
    rows: List[Dict] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def evaluate(rows: List[Dict], low_conf_threshold: float) -> Dict:
    labels = sorted({str(row.get("label")) for row in rows if row.get("label")})
    tp = Counter()
    fp = Counter()
    fn = Counter()
    total = 0
    top1_correct = 0
    top3_correct = 0
    low_conf_total = 0
    low_conf_correct = 0

    for row in rows:
        truth = str(row.get("label", ""))
        pred = str(row.get("pred_label", ""))
        confidence = float(row.get("confidence", 0.0))
        top3 = [str(v) for v in row.get("top3_labels", [])]
        if not truth:
            continue
        total += 1
        if pred == truth:
            top1_correct += 1
            tp[truth] += 1
        else:
            fp[pred] += 1
            fn[truth] += 1
        if truth in top3:
            top3_correct += 1
        if confidence < low_conf_threshold:
            low_conf_total += 1
            if pred == truth:
                low_conf_correct += 1

    class_metrics = defaultdict(dict)
    for label in labels:
        precision_den = tp[label] + fp[label]
        recall_den = tp[label] + fn[label]
        precision = tp[label] / precision_den if precision_den else 0.0
        recall = tp[label] / recall_den if recall_den else 0.0
        if precision + recall == 0:
            f1 = 0.0
        else:
            f1 = 2 * precision * recall / (precision + recall)
        class_metrics[label] = {
            "precision": round(precision, 6),
            "recall": round(recall, 6),
            "f1": round(f1, 6),
            "support": tp[label] + fn[label],
        }

    return {
        "total": total,
        "top1": round(top1_correct / total, 6) if total else 0.0,
        "top3": round(top3_correct / total, 6) if total else 0.0,
        "low_conf_threshold": low_conf_threshold,
        "low_conf_total": low_conf_total,
        "low_conf_top1": round(low_conf_correct / low_conf_total, 6) if low_conf_total else 0.0,
        "per_class": dict(class_metrics),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--low-conf-threshold", type=float, default=0.45)
    args = parser.parse_args()

    report = evaluate(read_jsonl(args.input), low_conf_threshold=args.low_conf_threshold)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=True, indent=2)
    print(json.dumps({"top1": report["top1"], "top3": report["top3"], "output": str(args.output)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
