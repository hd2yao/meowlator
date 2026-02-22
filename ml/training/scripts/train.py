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
import warnings
from dataclasses import dataclass
from typing import Dict, Iterable, List, Tuple

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


def evaluate(model, loader, criterion, device, num_classes: int) -> EpochMetrics:
    import torch

    model.eval()
    running_loss = 0.0
    correct_top1 = 0
    correct_top3 = 0
    total = 0

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

            k = min(3, num_classes)
            topk = logits.topk(k=k, dim=1).indices
            correct_top3 += topk.eq(labels.view(-1, 1)).any(dim=1).sum().item()

    return EpochMetrics(
        loss=running_loss / max(total, 1),
        top1=correct_top1 / max(total, 1),
        top3=correct_top3 / max(total, 1),
    )


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

    train_loader = DataLoader(
        train_ds,
        batch_size=args.batch_size,
        shuffle=True,
        num_workers=num_workers,
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
    model.to(device)

    criterion = nn.CrossEntropyLoss()
    optimizer = torch.optim.AdamW(model.parameters(), lr=args.lr)

    best_eval_top1 = 0.0
    history = []

    for epoch in range(1, args.epochs + 1):
        train_metrics = train_one_epoch(model, train_loader, criterion, optimizer, device, len(INTENT_LABELS))
        eval_metrics = evaluate(model, test_loader, criterion, device, len(INTENT_LABELS))

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

    torch.save(
        {
            "model_version": args.model_version,
            "model_state_dict": model.state_dict(),
            "num_classes": len(INTENT_LABELS),
            "input_size": args.input_size,
            "intent_labels": INTENT_LABELS,
            "intent_priors": priors,
            "history": history,
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
        "intent_priors": priors,
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

    print(json.dumps({"checkpoint": str(checkpoint_path), "metrics": str(metrics_path)}, ensure_ascii=True))


if __name__ == "__main__":
    main()
