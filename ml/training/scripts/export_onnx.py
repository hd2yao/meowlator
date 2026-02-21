"""ONNX export scaffold.

Real export should wire actual model checkpoint + quantization pipeline.
"""

from __future__ import annotations

import argparse
import pathlib


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        "# placeholder for mobilenetv3-small-int8 onnx artifact\n"
        f"# source checkpoint: {args.checkpoint}\n",
        encoding="utf-8",
    )
    print(f"exported placeholder artifact to {args.output}")


if __name__ == "__main__":
    main()
