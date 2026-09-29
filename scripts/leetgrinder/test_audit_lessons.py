import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

path = Path(__file__).with_name("audit_lessons.py")
spec = importlib.util.spec_from_file_location("audit_lessons", path)
audit = importlib.util.module_from_spec(spec)
spec.loader.exec_module(audit)

class AuditTests(unittest.TestCase):
    def test_missing_day_is_reported(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            report = audit.audit(root, days=[1])
            self.assertFalse(report["passed"])
            self.assertIn("day 01 lesson missing", report["days"][0]["errors"])

    def test_missing_language_and_case_are_reported(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            lessons=root/'internal/leetgrinder/lessons';lessons.mkdir(parents=True)
            examples=root/'internal/leetgrinder/examples/day-01/demo';examples.mkdir(parents=True)
            (lessons/'day-01.json').write_text(json.dumps({"day":1,"objectives":["A"],"sections":[{"id":"intro","heading":"Worked example","paragraphs":["A"],"exampleIds":["day-01-demo"],"checks":[{"prompt":"Q","answer":"A"}],"diagram":{"width":10,"height":10,"description":"D"}}]}))
            (examples/'example.json').write_text(json.dumps({"id":"day-01-demo","variants":[{"language":"python","file":"main.py"}],"cases":[{"id":"normal","frames":[{},{}]}]}))
            report=audit.audit(root,days=[1])
            self.assertFalse(report['passed'])
            self.assertTrue(any('four languages' in error for error in report['days'][0]['errors']))
            self.assertTrue(any('normal and edge cases' in error for error in report['days'][0]['errors']))

if __name__=='__main__':unittest.main()
