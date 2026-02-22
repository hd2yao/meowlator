import unittest

from build_training_manifest import build_manifest


class BuildTrainingManifestTest(unittest.TestCase):
    def test_build_manifest(self):
        public_rows = [
            {"sample_id": "p1", "image_path": "/data/p1.jpg", "label": "FEEDING"},
            {"sample_id": "bad", "image_path": "/data/bad.jpg", "label": "NOT_A_LABEL"},
        ]
        feedback_rows = [
            {
                "sample_id": "f1",
                "image_key": "samples/u1/f1.jpg",
                "label": "SEEK_ATTENTION",
                "training_weight": 0.6,
                "reliability_score": 0.8,
            }
        ]

        manifest, report = build_manifest(public_rows, feedback_rows, seed=123, min_weight=0.05)
        self.assertEqual(report["manifest_rows"], 2)
        self.assertEqual(report["dropped_public"], 1)
        self.assertEqual(report["dropped_feedback"], 0)

        by_id = {row["sample_id"]: row for row in manifest}
        self.assertAlmostEqual(by_id["p1"]["weight"], 1.0)
        self.assertAlmostEqual(by_id["f1"]["weight"], 0.48)
        self.assertEqual(by_id["f1"]["source"], "FEEDBACK")


if __name__ == "__main__":
    unittest.main()
