import unittest

from gate_model_release import evaluate_gate


class GateModelReleaseTest(unittest.TestCase):
    def test_gate_pass(self):
        baseline = {"final_eval_top1": 0.6, "final_eval_top3": 0.82, "ece": 0.08}
        candidate = {"final_eval_top1": 0.62, "final_eval_top3": 0.84, "ece": 0.07}
        report = evaluate_gate(baseline, candidate, max_top1_drop=0.01, max_top3_drop=0.01, max_ece_increase=0.02)
        self.assertTrue(report["pass"])

    def test_gate_fail(self):
        baseline = {"final_eval_top1": 0.6, "final_eval_top3": 0.82, "ece": 0.08}
        candidate = {"final_eval_top1": 0.55, "final_eval_top3": 0.79, "ece": 0.12}
        report = evaluate_gate(baseline, candidate, max_top1_drop=0.01, max_top3_drop=0.01, max_ece_increase=0.02)
        self.assertFalse(report["pass"])


if __name__ == "__main__":
    unittest.main()
