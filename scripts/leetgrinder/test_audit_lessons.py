"""The embedded asset audit catches accidental private research file copies."""
import tempfile
import unittest
from pathlib import Path

from audit_lessons import embedded_asset_errors


class EmbeddedAssetTests(unittest.TestCase):
    def test_rejects_article_copy_in_embedded_examples(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            example = root / "internal/leetgrinder/examples/day-01/demo"
            example.mkdir(parents=True)
            (example / "main.py").write_text("print(1)\n")
            self.assertEqual(embedded_asset_errors(root), [])
            article = example / "source-article.md"
            article.write_text("research material\n")
            self.assertEqual(
                embedded_asset_errors(root),
                ["unexpected embedded example asset: internal/leetgrinder/examples/day-01/demo/source-article.md"],
            )


if __name__ == "__main__":
    unittest.main()
