import json
import tempfile
import unittest
from pathlib import Path

from check_corpus_overlap import find_code_overlaps, find_overlaps


class CorpusOverlapTests(unittest.TestCase):
    def test_flags_copied_paragraph_and_accepts_original_paragraph(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "repo"
            corpus = Path(directory) / "corpus"
            lesson = root / "internal/leetgrinder/lessons/day-01.json"
            lesson.parent.mkdir(parents=True)
            corpus.mkdir()
            copied = "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen"
            (corpus / "research.md").write_text(copied)
            lesson.write_text(json.dumps({"paragraph": "A separate explanation with independent wording."}))
            self.assertEqual(find_overlaps(root, corpus), [])
            lesson.write_text(json.dumps({"paragraph": copied}))
            self.assertEqual(
                find_overlaps(root, corpus),
                [(Path("internal/leetgrinder/lessons/day-01.json"), Path("research.md"))],
            )

    def test_flags_copied_code_block(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / "repo"
            corpus = Path(directory) / "corpus"
            program = root / "internal/leetgrinder/examples/day-01/demo/main.py"
            program.parent.mkdir(parents=True)
            corpus.mkdir()
            source = "first = 1\nsecond = first + 2\nthird = second * 3\nfourth = third - 4\nprint(fourth)"
            (corpus / "research.md").write_text("```python\n" + source + "\n```\n")
            program.write_text("print('original')\n")
            self.assertEqual(find_code_overlaps(root, corpus), [])
            program.write_text(source + "\n")
            self.assertEqual(
                find_code_overlaps(root, corpus),
                [(Path("internal/leetgrinder/examples/day-01/demo/main.py"), Path("research.md"))],
            )


if __name__ == "__main__":
    unittest.main()
