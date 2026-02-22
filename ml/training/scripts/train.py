"""Train a MobileNetV3 vision baseline on Oxford-IIIT Pet.

Notes:
- Oxford-IIIT Pet does not include cat intent labels.
- This MVP stage uses a deterministic pseudo-label mapping from category id -> 8 intent buckets
  to train a transferable visual base model and produce deployable artifacts.
- User feedback fine-tuning still happens in later stages.
"""

from __future__ import annotations

import argparse
import json
import pathlib
import random
import warnings
from dataclasses import dataclass
from typing import Dict, Iterable, List, Optional, Tuple

from constants import INTENT_LABELS


def load_feedback(path: pathlib.Path | None) -> List[Dict]:
    if path is None or not path.exists():
        return []
    with path.open("r", encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def weighted_count(records: Iterable[Dict]) -> float:
    total = 0.0
    for row in records:
        total += float(row.get("training_weight", 0.2)) * float(row.get("reliability_score", 1.0))
    return total


def pseudo_intent_from_category(category: int, intent_count: int) -> int:
    normalized = max(0, int(category) - 1)
    return normalized % intent_count


@dataclass
class EpochMetrics:
    loss: float
    top1: float
    top3: float


def set_seed(seed: int) -> None:
    import torch

    random.seed(seed)
    torch.manual_seed(seed)
    if torch.cuda.is_available():
        torch.cuda.manual_seed_all(seed)
    try:
        import numpy as np

        np.random.seed(seed)
    except ImportError:
        pass


def train_one_epoch(model, loader, criterion, optimizer, device, num_classes: int) -> EpochMetrics:
    import torch

    model.train()
    running_loss = 0.0
    correct_top1 = 0
    correct_top3 = 0
    total = 0

    for images, labels in loader:
        images = images.to(device)
        labels = labels.to(device)

        optimizer.zero_grad(set_to_none=True)
        logits = model(images)
        loss = criterion(logits, labels)
        loss.backward()
        optimizer.step()

        batch_size = labels.size(0)
        running_loss += loss.item() * batch_size
        total += batch_size

        top1 = logits.argmax(dim=1)
        correct_top1 += (top1 == labels).sum().item()

        k = min(3, num_classes)
        topk = logits.topk(k=k, dim=1).indices
        correct_top3 += topk.eq(labels.view(-1, 1)).any(dim=1).sum().item()

    return EpochMetrics(
        loss=running_loss / max(total, 1),
        top1=correct_top1 / max(total, 1),
        top3=correct_top3 / max(total, 1),
    )


def evaluate(model, loader, criterion, device, num_classes: int) -> Tuple[EpochMetrics, List[List[int]], Dict]:
    import torch

    model.eval()
    running_loss = 0.0
    correct_top1 = 0
    correct_top3 = 0
    total = 0
    confusion = [[0 for _ in range(num_classes)] for _ in range(num_classes)]
    confidence_pairs: List[Tuple[float, int]] = []

    with torch.no_grad():
        for images, labels in loader:
            images = images.to(device)
            labels = labels.to(device)
            logits = model(images)
            loss = criterion(logits, labels)

            batch_size = labels.size(0)
            running_loss += loss.item() * batch_size
            total += batch_size

            top1 = logits.argmax(dim=1)
            correct_top1 += (top1 == labels).sum().item()
            label_values = labels.detach().cpu().tolist()
            pred_values = top1.detach().cpu().tolist()
            probs = torch.softmax(logits, dim=1)
            conf_values = probs.max(dim=1).values.detach().cpu().tolist()
            for truth, pred in zip(label_values, pred_values):
                confusion[int(truth)][int(pred)] += 1
            for conf, truth, pred in zip(conf_values, label_values, pred_values):
                confidence_pairs.append((float(conf), 1 if int(truth) == int(pred) else 0))

            k = min(3, num_classes)
            topk = logits.topk(k=k, dim=1).indices
            correct_top3 += topk.eq(labels.view(-1, 1)).any(dim=1).sum().item()

    metrics = EpochMetrics(
        loss=running_loss / max(total, 1),
        top1=correct_top1 / max(total, 1),
        top3=correct_top3 / max(total, 1),
    )
    return metrics, confusion, build_calibration(confidence_pairs)


def build_calibration(pairs: List[Tuple[float, int]], bins: int = 10) -> Dict:
    if bins <= 0:
        bins = 10
    if not pairs:
        return {"ece": 0.0, "bins": []}

    buckets = [
        {
            "idx": idx,
            "start": idx / bins,
            "end": (idx + 1) / bins,
            "count": 0,
            "correct": 0,
            "conf_sum": 0.0,
        }
        for idx in range(bins)
    ]
    for conf, correct in pairs:
        pos = int(conf * bins)
        if pos >= bins:
            pos = bins - 1
        bucket = buckets[pos]
        bucket["count"] += 1
        bucket["correct"] += correct
        bucket["conf_sum"] += conf

    total = float(len(pairs))
    ece = 0.0
    output_bins = []
    for bucket in buckets:
        count = int(bucket["count"])
        if count == 0:
            continue
        avg_conf = bucket["conf_sum"] / float(count)
        accuracy = float(bucket["correct"]) / float(count)
        gap = abs(avg_conf - accuracy)
        ece += (float(count) / total) * gap
        output_bins.append(
            {
                "idx": bucket["idx"],
                "start": round(bucket["start"], 4),
                "end": round(bucket["end"], 4),
                "count": count,
                "avg_conf": round(avg_conf, 6),
                "accuracy": round(accuracy, 6),
                "gap": round(gap, 6),
            }
        )

    return {"ece": round(ece, 6), "bins": output_bins}


def build_datasets(dataset_root: pathlib.Path, input_size: int, download: bool):
    import torch
    from torch.utils.data import Dataset
    from torchvision import datasets, transforms

    class IntentMappedDataset(Dataset):
        def __init__(self, base_dataset, intent_count: int):
            self.base_dataset = base_dataset
            self.intent_count = intent_count

        def __len__(self):
            return len(self.base_dataset)

        def __getitem__(self, index):
            image, category = self.base_dataset[index]
            if isinstance(category, (list, tuple)):
                category = category[0]
            label = pseudo_intent_from_category(int(category), self.intent_count)
            return image, torch.tensor(label, dtype=torch.long)

    train_transform = transforms.Compose(
        [
            transforms.Resize((input_size, input_size)),
            transforms.RandomHorizontalFlip(p=0.5),
            transforms.ColorJitter(brightness=0.2, contrast=0.2),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )
    eval_transform = transforms.Compose(
        [
            transforms.Resize((input_size, input_size)),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )

    base_train = datasets.OxfordIIITPet(
        root=str(dataset_root),
        split="trainval",
        target_types="category",
        transform=train_transform,
        download=download,
    )
    base_test = datasets.OxfordIIITPet(
        root=str(dataset_root),
        split="test",
        target_types="category",
        transform=eval_transform,
        download=download,
    )

    return (
        IntentMappedDataset(base_train, len(INTENT_LABELS)),
        IntentMappedDataset(base_test, len(INTENT_LABELS)),
    )


def build_fake_datasets(input_size: int):
    import torch
    from torchvision import datasets, transforms

    transform = transforms.Compose(
        [
            transforms.Resize((input_size, input_size)),
            transforms.ToTensor(),
        ]
    )

    class FakeIntentDataset(torch.utils.data.Dataset):
        def __init__(self, size: int):
            self.base = datasets.FakeData(
                size=size,
                image_size=(3, input_size, input_size),
                num_classes=len(INTENT_LABELS),
                transform=transform,
            )

        def __len__(self):
            return len(self.base)

        def __getitem__(self, idx):
            image, label = self.base[idx]
            return image, torch.tensor(int(label), dtype=torch.long)

    return FakeIntentDataset(size=512), FakeIntentDataset(size=128)


def compute_intent_priors(dataset) -> Dict[str, float]:
    counts = [0 for _ in INTENT_LABELS]
    for _, label in dataset:
        counts[int(label)] += 1
    total = sum(counts)
    if total == 0:
        return {label: round(1.0 / len(INTENT_LABELS), 6) for label in INTENT_LABELS}
    return {label: round(counts[idx] / total, 6) for idx, label in enumerate(INTENT_LABELS)}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", choices=["oxford", "fake"], default="oxford")
    parser.add_argument("--dataset-root", type=pathlib.Path, required=True)
    parser.add_argument("--output-dir", type=pathlib.Path, required=True)
    parser.add_argument("--model-version", default="mobilenetv3-small-v2")
    parser.add_argument("--epochs", type=int, default=3)
    parser.add_argument("--batch-size", type=int, default=32)
    parser.add_argument("--lr", type=float, default=3e-4)
    parser.add_argument("--input-size", type=int, default=224)
    parser.add_argument("--num-workers", type=int, default=2)
    parser.add_argument("--device", default="auto", choices=["auto", "cpu", "cuda"])
    parser.add_argument("--download", action="store_true")
    parser.add_argument("--feedback", type=pathlib.Path, default=None)
    parser.add_argument("--resume-checkpoint", type=pathlib.Path, default=None)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--pretrained", action=argparse.BooleanOptionalAction, default=True)
    args = parser.parse_args()

    try:
        import torch
        from torch import nn
        from torch.utils.data import DataLoader
        from torchvision import models
    except ImportError as exc:
        raise SystemExit(
            "Missing training dependencies. Run: pip install -r ml/training/requirements.txt"
        ) from exc

    set_seed(args.seed)

    if args.device == "auto":
        device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    else:
        device = torch.device(args.device)

    if args.dataset == "fake":
        train_ds, test_ds = build_fake_datasets(input_size=args.input_size)
        dataset_name = "fake_intent_smoke"
    else:
        train_ds, test_ds = build_datasets(
            dataset_root=args.dataset_root,
            input_size=args.input_size,
            download=args.download,
        )
        dataset_name = "oxford_iiit_pet_pseudo_intent"

    num_workers = max(0, int(args.num_workers))
    if num_workers > 0:
        warnings.warn(
            "Current dataset wrappers are not multiprocess-pickle safe in this environment; "
            "forcing num_workers=0 for stable training."
        )
        num_workers = 0

    loader_generator = torch.Generator()
    loader_generator.manual_seed(args.seed)

    train_loader = DataLoader(
        train_ds,
        batch_size=args.batch_size,
        shuffle=True,
        num_workers=num_workers,
        generator=loader_generator,
    )
    test_loader = DataLoader(
        test_ds,
        batch_size=args.batch_size,
        shuffle=False,
        num_workers=num_workers,
    )

    weights = models.MobileNet_V3_Small_Weights.IMAGENET1K_V1 if args.pretrained else None
    model = models.mobilenet_v3_small(weights=weights)
    in_features = model.classifier[3].in_features
    model.classifier[3] = nn.Linear(in_features, len(INTENT_LABELS))
    resumed_from: Optional[str] = None
    history: List[Dict] = []

    if args.resume_checkpoint is not None:
        if not args.resume_checkpoint.exists():
            raise SystemExit(f"resume checkpoint not found: {args.resume_checkpoint}")
        resume_ckpt = torch.load(args.resume_checkpoint, map_location="cpu")
        resume_state_dict = resume_ckpt.get("model_state_dict") or resume_ckpt.get("state_dict")
        if resume_state_dict is None:
            raise SystemExit("resume checkpoint missing model_state_dict/state_dict")
        missing_keys, unexpected_keys = model.load_state_dict(resume_state_dict, strict=False)
        if missing_keys or unexpected_keys:
            warnings.warn(
                f"resume checkpoint loaded with key mismatch; missing={missing_keys}, unexpected={unexpected_keys}"
            )
        resumed_from = str(args.resume_checkpoint)
        raw_history = resume_ckpt.get("history", [])
        if isinstance(raw_history, list):
            history = list(raw_history)

    model.to(device)

    criterion = nn.CrossEntropyLoss()
    optimizer = torch.optim.AdamW(model.parameters(), lr=args.lr)

    best_eval_top1 = 0.0
    for row in history:
        best_eval_top1 = max(best_eval_top1, float(row.get("eval_top1", 0.0)))

    latest_confusion = [[0 for _ in INTENT_LABELS] for _ in INTENT_LABELS]
    latest_calibration = {"ece": 0.0, "bins": []}
    start_epoch = len(history) + 1
    end_epoch = start_epoch + args.epochs
    for epoch in range(start_epoch, end_epoch):
        train_metrics = train_one_epoch(model, train_loader, criterion, optimizer, device, len(INTENT_LABELS))
        eval_metrics, latest_confusion, latest_calibration = evaluate(model, test_loader, criterion, device, len(INTENT_LABELS))

        history.append(
            {
                "epoch": epoch,
                "train_loss": round(train_metrics.loss, 6),
                "train_top1": round(train_metrics.top1, 6),
                "train_top3": round(train_metrics.top3, 6),
                "eval_loss": round(eval_metrics.loss, 6),
                "eval_top1": round(eval_metrics.top1, 6),
                "eval_top3": round(eval_metrics.top3, 6),
            }
        )
        best_eval_top1 = max(best_eval_top1, eval_metrics.top1)

    feedback_records = load_feedback(args.feedback)
    effective_samples = weighted_count(feedback_records)
    priors = compute_intent_priors(train_ds)

    args.output_dir.mkdir(parents=True, exist_ok=True)
    checkpoint_path = args.output_dir / f"{args.model_version}.pt"
    metrics_path = args.output_dir / "metrics.json"
    priors_path = args.output_dir / "intent_priors.json"
    confusion_path = args.output_dir / "confusion_matrix.json"
    calibration_path = args.output_dir / "calibration.json"

    torch.save(
        {
            "model_version": args.model_version,
            "model_state_dict": model.state_dict(),
            "num_classes": len(INTENT_LABELS),
            "input_size": args.input_size,
            "intent_labels": INTENT_LABELS,
            "intent_priors": priors,
            "history": history,
            "seed": args.seed,
            "resumed_from": resumed_from,
        },
        checkpoint_path,
    )

    metrics = {
        "model_version": args.model_version,
        "dataset": dataset_name,
        "epochs": args.epochs,
        "batch_size": args.batch_size,
        "best_eval_top1": round(best_eval_top1, 6),
        "final_eval_top1": round(history[-1]["eval_top1"], 6) if history else 0.0,
        "final_eval_top3": round(history[-1]["eval_top3"], 6) if history else 0.0,
        "records": len(feedback_records),
        "effective_samples": round(effective_samples, 3),
        "checkpoint": str(checkpoint_path),
        "resumed_from": resumed_from,
        "seed": args.seed,
        "intent_priors": priors,
        "ece": latest_calibration.get("ece", 0.0),
        "history": history,
    }

    with metrics_path.open("w", encoding="utf-8") as f:
        json.dump(metrics, f, ensure_ascii=True, indent=2)

    with priors_path.open("w", encoding="utf-8") as f:
        json.dump(
            {
                "model_version": args.model_version,
                "intent_priors": priors,
                "source": dataset_name,
            },
            f,
            ensure_ascii=True,
            indent=2,
        )

    with confusion_path.open("w", encoding="utf-8") as f:
        json.dump(
            {
                "model_version": args.model_version,
                "labels": INTENT_LABELS,
                "matrix": latest_confusion,
            },
            f,
            ensure_ascii=True,
            indent=2,
        )

    with calibration_path.open("w", encoding="utf-8") as f:
        json.dump(
            {
                "model_version": args.model_version,
                "ece": latest_calibration.get("ece", 0.0),
                "bins": latest_calibration.get("bins", []),
            },
            f,
            ensure_ascii=True,
            indent=2,
        )

    print(
        json.dumps(
            {
                "checkpoint": str(checkpoint_path),
                "metrics": str(metrics_path),
                "confusion_matrix": str(confusion_path),
                "calibration": str(calibration_path),
            },
            ensure_ascii=True,
        )
    )


if __name__ == "__main__":
    main()
