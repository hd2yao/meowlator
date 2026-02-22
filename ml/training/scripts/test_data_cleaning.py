import unittest

from data_cleaning import clean_feedback_rows


class DataCleaningTest(unittest.TestCase):
    def test_clean_feedback_rows(self):
        rows = [
            {
                "sample_id": "s1_old",
                "user_id": "u1",
                "image_key": "img-1.jpg",
                "phash": "p1",
                "is_correct": False,
                "true_label": "FEEDING",
                "model_label": "WANT_PLAY",
                "training_weight": 1.0,
                "reliability_score": 1.0,
                "created_at": 100,
            },
            {
                "sample_id": "s1_new",
                "user_id": "u1",
                "image_key": "img-1.jpg",
                "phash": "p1",
                "is_correct": False,
                "true_label": "SEEK_ATTENTION",
                "model_label": "WANT_PLAY",
                "training_weight": 1.0,
                "reliability_score": 0.9,
                "created_at": 200,
            },
            {
                "sample_id": "missing_true",
                "user_id": "u1",
                "image_key": "img-2.jpg",
                "is_correct": False,
                "training_weight": 1.0,
                "reliability_score": 1.0,
                "created_at": 300,
            },
        ]
        for idx in range(6):
            rows.append(
                {
                    "sample_id": f"susp-{idx}",
                    "user_id": "u_susp",
                    "image_key": f"susp-{idx}.jpg",
                    "is_correct": False,
                    "true_label": "DEFENSIVE_ALERT",
                    "training_weight": 1.0,
                    "reliability_score": 1.0,
                    "created_at": 400 + idx,
                }
            )

        cleaned, report = clean_feedback_rows(rows, suspicious_scale=0.5)
        self.assertEqual(report["dropped_missing_true_label"], 1)
        self.assertEqual(len(cleaned), 7)

        dedup_match = [r for r in cleaned if r["dedup_key"] == "p1"]
        self.assertEqual(len(dedup_match), 1)
        self.assertEqual(dedup_match[0]["sample_id"], "s1_new")
        self.assertEqual(dedup_match[0]["label"], "SEEK_ATTENTION")

        susp_rows = [r for r in cleaned if r["user_id"] == "u_susp"]
        self.assertTrue(all(abs(r["reliability_score"] - 0.5) < 1e-9 for r in susp_rows))
        self.assertIn("u_susp", report["suspicious_users"])


if __name__ == "__main__":
    unittest.main()
