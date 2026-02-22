"""Build deterministic train/val/test splits from a manifest JSONL file."""

from __future__ import annotations

import argparse
import json
import pathlib
import random
from typing import Dict, Iterable, List


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


def split_manifest(rows: List[Dict], seed: int, train_ratio: float, val_ratio: float) -> Dict[str, List[Dict]]:
    rng = random.Random(seed)
    shuffled = list(rows)
    rng.shuffle(shuffled)

    n_total = len(shuffled)
    n_train = int(n_total * train_ratio)
    n_val = int(n_total * val_ratio)
    n_test = max(0, n_total - n_train - n_val)

    return {
        "train": shuffled[:n_train],
        "val": shuffled[n_train : n_train + n_val],
        "test": shuffled[n_train + n_val : n_train + n_val + n_test],
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=pathlib.Path, required=True)
    parser.add_argument("--output-dir", type=pathlib.Path, required=True)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--train-ratio", type=float, default=0.7)
    parser.add_argument("--val-ratio", type=float, default=0.15)
    args = parser.parse_args()

    rows = read_jsonl(args.input)
    split = split_manifest(rows, args.seed, args.train_ratio, args.val_ratio)
    for name, content in split.items():
        write_jsonl(args.output_dir / f"{name}.jsonl", content)
    report = {
        "seed": args.seed,
        "total": len(rows),
        "train": len(split["train"]),
        "val": len(split["val"]),
        "test": len(split["test"]),
    }
    with (args.output_dir / "split_report.json").open("w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=True, indent=2)
    print(json.dumps(report, ensure_ascii=True))


if __name__ == "__main__":
    main()
