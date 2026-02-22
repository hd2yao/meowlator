"""Generate threshold impact report for edge/cloud decision."""

from __future__ import annotations

import argparse
import json
import pathlib
from typing import Dict, List


def read_jsonl(path: pathlib.Path) -> List[Dict]:
    rows: List[Dict] = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def decide(confidence: float, device_capable: bool, edge_accept: float, cloud_fallback: float) -> Dict[str, bool]:
    if not device_capable:
        return {"use_edge": False, "need_cloud": True, "force_feedback": True}
    if confidence >= edge_accept:
        return {"use_edge": True, "need_cloud": False, "force_feedback": False}
    if confidence < cloud_fallback:
        return {"use_edge": False, "need_cloud": True, "force_feedback": True}
    return {"use_edge": False, "need_cloud": True, "force_feedback": False}


def evaluate(rows: List[Dict], edge_accept: float, cloud_fallback: float) -> Dict:
    total = len(rows)
    if total == 0:
        return {
            "edge_accept": edge_accept,
            "cloud_fallback": cloud_fallback,
            "total": 0,
            "edge_ratio": 0.0,
            "cloud_ratio": 0.0,
            "feedback_ratio": 0.0,
        }
    use_edge = 0
    need_cloud = 0
    need_feedback = 0
    for row in rows:
        decision = decide(
            confidence=float(row.get("edge_confidence", 0.0)),
            device_capable=bool(row.get("device_capable", True)),
            edge_accept=edge_accept,
            cloud_fallback=cloud_fallback,
        )
        if decision["use_edge"]:
            use_edge += 1
        if decision["need_cloud"]:
            need_cloud += 1
        if decision["force_feedback"]:
            need_feedback += 1
    return {
        "edge_accept": edge_accept,
        "cloud_fallback": cloud_fallback,
        "total": total,
        "edge_ratio": round(use_edge / total, 6),
        "cloud_ratio": round(need_cloud / total, 6),
        "feedback_ratio": round(need_feedback / total, 6),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--edge-accept-list", default="0.65,0.70,0.75")
    parser.add_argument("--cloud-fallback-list", default="0.40,0.45,0.50")
    args = parser.parse_args()

    rows = read_jsonl(args.input)
    edge_accept_values = [float(v) for v in args.edge_accept_list.split(",") if v.strip()]
    cloud_fallback_values = [float(v) for v in args.cloud_fallback_list.split(",") if v.strip()]

    reports = []
    for edge_accept in edge_accept_values:
        for cloud_fallback in cloud_fallback_values:
            reports.append(evaluate(rows, edge_accept=edge_accept, cloud_fallback=cloud_fallback))

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        json.dump({"reports": reports}, f, ensure_ascii=True, indent=2)
    print(json.dumps({"reports": len(reports), "output": str(args.output)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
