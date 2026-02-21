#!/usr/bin/env python3
"""Append an implementation node record to docs/implementation_nodes.md."""

from __future__ import annotations

import argparse
import datetime as dt
import pathlib


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--node-id", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--area", required=True)
    parser.add_argument("--functional-node", required=True)
    parser.add_argument("--verification", required=True)
    parser.add_argument("--commit", default="TBD")
    parser.add_argument(
        "--file",
        default="docs/implementation_nodes.md",
        help="target markdown table file",
    )
    args = parser.parse_args()

    today = dt.date.today().isoformat()
    row = (
        f"| {args.node_id} | {today} | {args.version} | {args.area} | "
        f"{args.functional_node} | {args.verification} | `{args.commit}` |\n"
    )

    path = pathlib.Path(args.file)
    if not path.exists():
        raise SystemExit(f"target file does not exist: {path}")

    content = path.read_text(encoding="utf-8")
    if not content.endswith("\n"):
        content += "\n"
    content += row
    path.write_text(content, encoding="utf-8")
    print(f"appended node {args.node_id} to {path}")


if __name__ == "__main__":
    main()
