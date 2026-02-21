import unittest

from active_learning import Candidate, select_candidates


class ActiveLearningTest(unittest.TestCase):
    def test_sampling_budget(self):
        pool = [
            Candidate(sample_id=f"s{i}", confidence=0.1 + i * 0.01, is_conflict=i % 2 == 0, is_novel=i % 3 == 0)
            for i in range(30)
        ]
        selected = select_candidates(pool, daily_budget=10)
        self.assertEqual(len(selected), 10)
        self.assertEqual(len({c.sample_id for c in selected}), 10)


if __name__ == "__main__":
    unittest.main()
