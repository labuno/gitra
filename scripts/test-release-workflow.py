"""Exercise the release creation step with a fake GitHub CLI, without network access."""

import os
import re
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parent.parent


def creation_step():
    workflow = (ROOT / ".github/workflows/release.yml").read_text()
    prepare = workflow.split("name: Create the release if it does not exist yet", 1)[1]
    body = prepare.split("run: |\n", 1)[1]
    return textwrap.dedent(re.split(r"\n  [a-z][a-z-]*:\n", body, maxsplit=1)[0])


class ReleaseWorkflowTest(unittest.TestCase):
    def run_creation(self, tag, exists=False, has_notes=True):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            gh = directory / "gh"
            gh.write_text(
                '#!/bin/bash\n'
                'if [ "$2" = view ]; then exit "$GH_VIEW_STATUS"; fi\n'
                'printf "%s\\0" "$@" > "$GH_CALLS"\n'
            )
            gh.chmod(0o755)
            calls = directory / "calls"
            notes = directory / "docs/releases" / (tag + ".md")
            notes.parent.mkdir(parents=True)
            if has_notes:
                notes.write_text("## What's new\n\n- Version-specific changes.\n\n## 更新内容\n")
            env = dict(os.environ, TAG=tag, GH_CALLS=str(calls),
                       GH_VIEW_STATUS="0" if exists else "1",
                       PATH=str(directory) + os.pathsep + os.environ["PATH"])
            result = subprocess.run(["bash", "-c", creation_step()], cwd=directory,
                                    env=env, capture_output=True, text=True)
            if not has_notes and not exists:
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(calls.exists())
                self.assertIn("Missing release notes", result.stderr)
                return []
            self.assertEqual(result.returncode, 0, result.stderr)
            if not calls.exists():
                return []
            args = calls.read_bytes().decode().rstrip("\0").split("\0")
            self.assertEqual(args[args.index("--notes-file") + 1], "docs/releases/" + tag + ".md")
            return args

    def test_rc_is_prerelease_and_does_not_replace_latest(self):
        args = self.run_creation("v0.1.4-rc1")
        self.assertEqual(args[:3], ["release", "create", "v0.1.4-rc1"])
        self.assertIn("--prerelease", args)
        self.assertIn("--latest=false", args)

    def test_stable_release_preserves_normal_creation(self):
        args = self.run_creation("v0.1.4")
        self.assertEqual(args[:3], ["release", "create", "v0.1.4"])
        self.assertNotIn("--prerelease", args)
        self.assertNotIn("--latest=false", args)
        self.assertIn("--notes-file", args)

    def test_existing_release_is_not_created_again(self):
        self.assertEqual(self.run_creation("v0.1.4-rc1", exists=True), [])

    def test_missing_release_notes_stop_creation(self):
        self.run_creation("v0.1.4-rc1", has_notes=False)

    def test_only_macos_app_and_checksums_are_uploaded(self):
        workflow = (ROOT / ".github/workflows/release.yml").read_text()
        uploads = re.findall(r'gh release upload "\$TAG" (\S+)', workflow)
        self.assertEqual(uploads, ["dist/Gitra-macOS.zip", "assets/checksums.txt"])
        checksums = workflow.split("\n  checksums:\n", 1)[1]
        self.assertIn("needs: macos-app", checksums)


if __name__ == "__main__":
    unittest.main()
