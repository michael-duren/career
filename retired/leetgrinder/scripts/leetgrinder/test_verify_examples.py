"""Behavioral tests for the four-language example verifier."""

import copy
import json
import tempfile
import unittest
from pathlib import Path

import verify_examples as verifier


SOURCES = {
    "python": ("main.py", 'import json,sys\nsys.stdin.read()\nprint(json.dumps({"event":"start","variables":[{"name":"n","value":"1"}]}))\nprint(json.dumps({"event":"done","variables":[{"name":"n","value":"2"}]}))\nprint(json.dumps({"result":"2"}))\n'),
    "cpp": ("main.cpp", '#include <iostream>\nint main(){std::cout << "{\\"event\\":\\"start\\",\\"variables\\":[{\\"name\\":\\"n\\",\\"value\\":\\"1\\"}]}\\n{\\"event\\":\\"done\\",\\"variables\\":[{\\"name\\":\\"n\\",\\"value\\":\\"2\\"}]}\\n{\\"result\\":\\"2\\"}\\n";}\n'),
    "java": ("Main.java", 'class Main { public static void main(String[] args) { System.out.println("{\\"event\\":\\"start\\",\\"variables\\":[{\\"name\\":\\"n\\",\\"value\\":\\"1\\"}]}"); System.out.println("{\\"event\\":\\"done\\",\\"variables\\":[{\\"name\\":\\"n\\",\\"value\\":\\"2\\"}]}"); System.out.println("{\\"result\\":\\"2\\"}"); }}\n'),
    "go": ("main.go", 'package main\nimport "fmt"\nfunc main(){fmt.Println(`{"event":"start","variables":[{"name":"n","value":"1"}]}`);fmt.Println(`{"event":"done","variables":[{"name":"n","value":"2"}]}`);fmt.Println(`{"result":"2"}`)}\n'),
}


def example_data():
    scene = {"width": 100, "height": 80, "description": "n changes", "nodes": [], "edges": [], "labels": []}
    def frame(event, value):
        return {"event": event, "lines": {lang: [1] for lang in SOURCES}, "explanation": event,
                "variables": [{"name": "n", "value": value}], "scene": copy.deepcopy(scene)}
    return {"id": "day-01-tiny", "title": "Tiny", "invariant": "n increments",
            "variants": [{"language": lang, "file": name, "algorithmStart": 1, "algorithmEnd": 1}
                         for lang, (name, _) in SOURCES.items()],
            "cases": [{"id": "normal", "label": "Normal", "input": "1\n", "expectedOutput": "2",
                       "frames": [frame("start", "1"), frame("done", "2")]},
                      {"id": "edge", "label": "Edge", "input": "0\n", "expectedOutput": "2",
                       "frames": [frame("start", "1"), frame("done", "2")]}]}


class VerifierTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name) / "examples" / "day-01" / "tiny"
        self.directory.mkdir(parents=True)
        self.data = example_data()
        self.write()

    def write(self):
        (self.directory / "example.json").write_text(json.dumps(self.data), encoding="utf-8")
        for name, source in SOURCES.values():
            (self.directory / name).write_text(source, encoding="utf-8")

    def verify(self):
        return verifier.verify_example(self.directory)

    def test_all_four_languages_pass_both_cases_and_record_commands(self):
        result = self.verify()
        self.assertTrue(result["passed"], result)
        self.assertEqual(8, len(result["runs"]))
        self.assertEqual({"cpp", "python", "java", "go"}, {run["language"] for run in result["runs"]})
        for run in result["runs"]:
            self.assertTrue(run["passed"], run)
            self.assertTrue(run["command"])
            self.assertTrue(run["toolchainVersion"])

    def test_go_text_source_builds_with_original_line_mappings(self):
        source = self.directory / "main.go"
        text_source = self.directory / "main.go.txt"
        source.rename(text_source)
        go_variant = next(v for v in self.data["variants"] if v["language"] == "go")
        go_variant.update(file="main.go.txt", algorithmStart=3, algorithmEnd=3)
        for case in self.data["cases"]:
            for frame in case["frames"]:
                frame["lines"]["go"] = [3]
        (self.directory / "example.json").write_text(json.dumps(self.data), encoding="utf-8")

        result = self.verify()

        self.assertTrue(result["passed"], result)
        self.assertEqual(text_source.read_text(), SOURCES["go"][1])
        self.assertFalse(source.exists())
        for run in result["runs"]:
            if run["language"] == "go":
                self.assertEqual("main.go", Path(run["compileCommand"][-1]).name)

    def test_wrong_result_fails_case_and_language(self):
        path = self.directory / "main.py"
        path.write_text(path.read_text().replace('"result":"2"', '"result":"3"'))
        result = self.verify()
        self.assertFalse(result["passed"])
        run = next(run for run in result["runs"] if run["case"] == "normal" and run["language"] == "python")
        self.assertIn("result", " ".join(run["errors"]))

    def test_wrong_intermediate_variable_fails(self):
        path = self.directory / "main.py"
        path.write_text(path.read_text().replace('"value":"1"', '"value":"9"'))
        result = self.verify()
        self.assertIn("variables", " ".join(error for run in result["runs"] if run["language"] == "python" for error in run["errors"]))

    def test_extra_and_missing_event_fail(self):
        path = self.directory / "main.py"
        original = path.read_text()
        for replacement in (original.replace('print(json.dumps({"result":"2"}))', 'print(json.dumps({"event":"extra","variables":[]}))\nprint(json.dumps({"result":"2"}))'),
                            original.replace('print(json.dumps({"event":"done","variables":[{"name":"n","value":"2"}]}))\n', '')):
            path.write_text(replacement)
            result = self.verify()
            self.assertFalse(result["passed"])
            self.assertTrue(result["runs"][1]["errors"])

    def test_nonzero_exit_invalid_json_and_timeout_fail(self):
        path = self.directory / "main.py"
        for source, expected in [('raise SystemExit(7)\n', 'exit'), ('print("not-json")\n', 'JSON'),
                                 ('import time\ntime.sleep(3)\n', 'timeout')]:
            path.write_text(source)
            result = verifier.verify_example(self.directory, run_timeout=0.2)
            self.assertFalse(result["passed"])
            self.assertIn(expected, " ".join(error for run in result["runs"] if run["language"] == "python" for error in run["errors"]))

    def test_missing_compiler_is_a_failure(self):
        result = verifier.verify_example(self.directory, tool_commands={"g++": "missing-gxx-test-command"})
        self.assertFalse(result["passed"])
        self.assertIn("missing-gxx-test-command", str(result["errors"]))

    def test_schema_rejects_unsafe_path_missing_language_and_bad_scene(self):
        for mutation, expected in [
            (lambda d: d["variants"][0].update(file="../outside.cpp"), "file"),
            (lambda d: d.update(id="another-example"), "ID"),
            (lambda d: d["variants"].pop(), "languages"),
            (lambda d: d["cases"][0]["frames"][0]["lines"].pop("go"), "lines"),
            (lambda d: d["cases"][0]["frames"][0]["scene"]["edges"].append({"id": "e", "from": "absent", "to": "absent", "label": "", "directed": True, "role": "active"}), "edge"),
            (lambda d: d["cases"][0]["frames"][0]["variables"].append({"name": "n", "value": "3"}), "duplicate"),
        ]:
            with self.subTest(expected=expected):
                data = example_data()
                mutation(data)
                self.data = data
                self.write()
                result = self.verify()
                self.assertFalse(result["passed"])
                self.assertIn(expected, " ".join(result["errors"]))

    def test_symlinked_metadata_is_rejected(self):
        metadata = self.directory / "example.json"
        target = Path(self.temp.name) / "external.json"
        metadata.rename(target)
        metadata.symlink_to(target)
        result = self.verify()
        self.assertFalse(result["passed"])
        self.assertIn("unsafe", " ".join(result["errors"]))

    def test_cli_fails_for_missing_selected_day_and_writes_report(self):
        report_path = Path(self.temp.name) / "report.json"
        code = verifier.main(["--days", "2", "--root", str(Path(self.temp.name) / "examples"), "--report", str(report_path)])
        self.assertEqual(1, code)
        report = json.loads(report_path.read_text())
        self.assertFalse(report["passed"])
        self.assertIn("day-02", str(report["errors"]))

    def test_cli_passes_selected_day_and_reports_every_run(self):
        report_path = Path(self.temp.name) / "report.json"
        code = verifier.main(["--days", "1", "--root", str(Path(self.temp.name) / "examples"), "--report", str(report_path)])
        self.assertEqual(0, code)
        report = json.loads(report_path.read_text())
        self.assertTrue(report["passed"])
        self.assertEqual(8, len(report["examples"][0]["runs"]))

    def test_day_with_unconfigured_example_directory_fails(self):
        (self.directory.parent / "forgotten").mkdir()
        report = verifier.verify_days(self.directory.parent.parent, [1])
        self.assertFalse(report["passed"])
        self.assertIn("forgotten", " ".join(report["errors"]))


if __name__ == "__main__":
    unittest.main()
