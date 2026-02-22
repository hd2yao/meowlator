"""Build unified training manifest from public data and cleaned feedback."""

from __future__ import annotations

import argparse
import json
import pathlib
import random
from typing import Dict, Iterable, List, Tuple

from constants import INTENT_LABELS


def read_jsonl(path: pathlib.Path | None) -> List[Dict]:
    if path is None or not path.exists():
        return []
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


def validate_label(label: str) -> bool:
    return label in INTENT_LABELS


def build_manifest(
    public_rows: List[Dict],
    feedback_rows: List[Dict],
    seed: int = 42,
    min_weight: float = 0.05,
) -> Tuple[List[Dict], Dict]:
    manifest: List[Dict] = []
    dropped_public = 0
    dropped_feedback = 0

    for row in public_rows:
        label = str(row.get("label", "")).strip()
        path = str(row.get("image_path") or row.get("image_key") or "").strip()
        if not validate_label(label) or not path:
            dropped_public += 1
            continue
        manifest.append(
            {
                "sample_id": str(row.get("sample_id", path)),
                "image_path": path,
                "label": label,
                "weight": 1.0,
                "source": "PUBLIC",
                "scene_tag": str(row.get("scene_tag", "UNKNOWN")),
            }
        )

    for row in feedback_rows:
        label = str(row.get("label", "")).strip()
        path = str(row.get("image_key") or row.get("image_path") or "").strip()
        if not validate_label(label) or not path:
            dropped_feedback += 1
            continue
        weight = float(row.get("training_weight", 0.2)) * float(row.get("reliability_score", 1.0))
        weight = max(min_weight, round(weight, 6))
        manifest.append(
            {
                "sample_id": str(row.get("sample_id", path)),
                "image_path": path,
                "label": label,
                "weight": weight,
                "source": "FEEDBACK",
                "scene_tag": str(row.get("scene_tag", "UNKNOWN")),
            }
        )

    rng = random.Random(seed)
    rng.shuffle(manifest)
    report = {
        "public_input": len(public_rows),
        "feedback_input": len(feedback_rows),
        "manifest_rows": len(manifest),
        "dropped_public": dropped_public,
        "dropped_feedback": dropped_feedback,
    }
    return manifest, report


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--public-manifest", type=pathlib.Path, default=None)
    parser.add_argument("--feedback", type=pathlib.Path, default=None)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--report", type=pathlib.Path, required=True)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--min-weight", type=float, default=0.05)
    args = parser.parse_args()

    public_rows = read_jsonl(args.public_manifest)
    feedback_rows = read_jsonl(args.feedback)
    manifest, report = build_manifest(
        public_rows=public_rows,
        feedback_rows=feedback_rows,
        seed=args.seed,
        min_weight=args.min_weight,
    )

    write_jsonl(args.output, manifest)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    with args.report.open("w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=True, indent=2)

    print(json.dumps({"manifest_rows": len(manifest), "report": str(args.report)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
