import unittest

from build_eval_splits import split_manifest


class BuildEvalSplitsTest(unittest.TestCase):
    def test_split_manifest(self):
        rows = [{"sample_id": f"s{i}", "label": "FEEDING"} for i in range(100)]
        split = split_manifest(rows, seed=42, train_ratio=0.7, val_ratio=0.15)
        self.assertEqual(len(split["train"]), 70)
        self.assertEqual(len(split["val"]), 15)
        self.assertEqual(len(split["test"]), 15)
        merged = split["train"] + split["val"] + split["test"]
        self.assertEqual(len({r["sample_id"] for r in merged}), 100)


if __name__ == "__main__":
    unittest.main()
