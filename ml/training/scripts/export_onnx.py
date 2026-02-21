"""Export trained MobileNetV3 checkpoint to ONNX.

Supports optional dynamic INT8 quantization via onnxruntime.
"""

from __future__ import annotations

import argparse
import pathlib


def build_model(checkpoint: dict):
    import torch
    from torch import nn
    from torchvision import models

    num_classes = int(checkpoint.get("num_classes", 8))
    model = models.mobilenet_v3_small(weights=None)
    in_features = model.classifier[3].in_features
    model.classifier[3] = nn.Linear(in_features, num_classes)

    state_dict = checkpoint.get("model_state_dict") or checkpoint.get("state_dict")
    if not state_dict:
        raise ValueError("checkpoint missing model_state_dict/state_dict")

    model.load_state_dict(state_dict, strict=True)
    model.eval()
    input_size = int(checkpoint.get("input_size", 224))
    return model, input_size


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--opset", type=int, default=17)
    parser.add_argument("--quantize-int8", action="store_true")
    parser.add_argument("--quantized-output", type=pathlib.Path, default=None)
    args = parser.parse_args()

    try:
        import torch
    except ImportError as exc:
        raise SystemExit(
            "Missing torch dependency. Run: pip install -r ml/training/requirements.txt"
        ) from exc

    checkpoint = torch.load(args.checkpoint, map_location="cpu")
    model, input_size = build_model(checkpoint)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    dummy = torch.randn(1, 3, input_size, input_size)

    torch.onnx.export(
        model,
        dummy,
        args.output,
        input_names=["input"],
        output_names=["logits"],
        dynamic_axes={"input": {0: "batch"}, "logits": {0: "batch"}},
        opset_version=args.opset,
    )

    quantized_path = None
    if args.quantize_int8:
        try:
            from onnxruntime.quantization import QuantType, quantize_dynamic
        except ImportError as exc:
            raise SystemExit(
                "Missing onnxruntime dependency for quantization. Install ml/training/requirements.txt"
            ) from exc

        quantized_path = args.quantized_output
        if quantized_path is None:
            quantized_path = args.output.with_name(args.output.stem + ".int8.onnx")

        quantize_dynamic(
            model_input=str(args.output),
            model_output=str(quantized_path),
            weight_type=QuantType.QInt8,
        )

    message = {"onnx": str(args.output)}
    if quantized_path is not None:
        message["onnx_int8"] = str(quantized_path)
    print(message)


if __name__ == "__main__":
    main()
