import unittest

from evaluate_intent_metrics import evaluate


class EvaluateIntentMetricsTest(unittest.TestCase):
    def test_evaluate(self):
        rows = [
            {"label": "FEEDING", "pred_label": "FEEDING", "confidence": 0.8, "top3_labels": ["FEEDING", "WANT_PLAY"]},
            {"label": "WANT_PLAY", "pred_label": "SEEK_ATTENTION", "confidence": 0.4, "top3_labels": ["SEEK_ATTENTION", "WANT_PLAY"]},
            {"label": "WANT_PLAY", "pred_label": "WANT_PLAY", "confidence": 0.42, "top3_labels": ["WANT_PLAY", "FEEDING"]},
        ]
        report = evaluate(rows, low_conf_threshold=0.45)
        self.assertAlmostEqual(report["top1"], 2 / 3, places=6)
        self.assertAlmostEqual(report["top3"], 1.0, places=6)
        self.assertEqual(report["low_conf_total"], 2)
        self.assertIn("WANT_PLAY", report["per_class"])


if __name__ == "__main__":
    unittest.main()
