import unittest

from generate_active_learning_tasks import PoolRow, select_with_reasons


class GenerateActiveLearningTasksTest(unittest.TestCase):
    def test_select_with_reasons_ratio(self):
        pool = []
        for i in range(10):
            pool.append(PoolRow(sample_id=f"low-{i}", confidence=0.1 + i * 0.01, is_conflict=False, is_novel=False))
            pool.append(PoolRow(sample_id=f"conflict-{i}", confidence=0.8, is_conflict=True, is_novel=False))
            pool.append(PoolRow(sample_id=f"novel-{i}", confidence=0.9, is_conflict=False, is_novel=True))

        selected, report = select_with_reasons(pool, daily_budget=10)
        self.assertEqual(len(selected), 10)
        self.assertEqual(len({row["sample_id"] for row in selected}), 10)

        counts = report["reason_counts"]
        self.assertEqual(counts.get("LOW_CONFIDENCE", 0), 4)
        self.assertEqual(counts.get("PREDICTION_CONFLICT", 0), 4)
        self.assertEqual(counts.get("NOVEL_SCENE", 0), 2)


if __name__ == "__main__":
    unittest.main()
