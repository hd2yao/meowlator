"""Gate model release by comparing candidate metrics against baseline."""

from __future__ import annotations

import argparse
import json
import pathlib
from typing import Dict, List


def read_json(path: pathlib.Path) -> Dict:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def evaluate_gate(baseline: Dict, candidate: Dict, max_top1_drop: float, max_top3_drop: float, max_ece_increase: float) -> Dict:
    b_top1 = float(baseline.get("final_eval_top1", 0.0))
    b_top3 = float(baseline.get("final_eval_top3", 0.0))
    b_ece = float(baseline.get("ece", 0.0))
    c_top1 = float(candidate.get("final_eval_top1", 0.0))
    c_top3 = float(candidate.get("final_eval_top3", 0.0))
    c_ece = float(candidate.get("ece", 0.0))

    reasons: List[str] = []
    if c_top1 < b_top1 - max_top1_drop:
        reasons.append(f"top1 regression too high: baseline={b_top1:.4f}, candidate={c_top1:.4f}")
    if c_top3 < b_top3 - max_top3_drop:
        reasons.append(f"top3 regression too high: baseline={b_top3:.4f}, candidate={c_top3:.4f}")
    if c_ece > b_ece + max_ece_increase:
        reasons.append(f"ece increase too high: baseline={b_ece:.4f}, candidate={c_ece:.4f}")

    improved = c_top1 > b_top1 or c_top3 > b_top3 or c_ece < b_ece
    if not improved:
        reasons.append("candidate has no observable improvement")

    return {
        "pass": len(reasons) == 0,
        "reasons": reasons,
        "baseline": {"top1": b_top1, "top3": b_top3, "ece": b_ece},
        "candidate": {"top1": c_top1, "top3": c_top3, "ece": c_ece},
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--baseline", type=pathlib.Path, required=True)
    parser.add_argument("--candidate", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--max-top1-drop", type=float, default=0.01)
    parser.add_argument("--max-top3-drop", type=float, default=0.01)
    parser.add_argument("--max-ece-increase", type=float, default=0.02)
    args = parser.parse_args()

    report = evaluate_gate(
        baseline=read_json(args.baseline),
        candidate=read_json(args.candidate),
        max_top1_drop=args.max_top1_drop,
        max_top3_drop=args.max_top3_drop,
        max_ece_increase=args.max_ece_increase,
    )
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=True, indent=2)
    print(json.dumps({"pass": report["pass"], "output": str(args.output)}, ensure_ascii=True))
    if not report["pass"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
