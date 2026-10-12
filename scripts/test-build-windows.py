"""Verify the portable package contract without requiring a Windows host."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import zipfile

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("build_windows", ROOT / "scripts/build-windows.py")
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class WindowsPackageTest(unittest.TestCase):
    def test_package_contains_versioned_executable_launcher_and_instructions(self):
        def compile_binary(command, **kwargs):
            self.assertEqual(kwargs["env"]["GOOS"], "windows")
            self.assertEqual(kwargs["env"]["GOARCH"], "amd64")
            self.assertEqual(kwargs["env"]["CGO_ENABLED"], "0")
            self.assertIn("Version=v0.1.4-test", command[command.index("-ldflags") + 1])
            Path(command[command.index("-o") + 1]).write_bytes(b"MZ-test-binary")

        with tempfile.TemporaryDirectory() as temporary, patch.object(builder.subprocess, "run", side_effect=compile_binary):
            archive = builder.build("v0.1.4-test", Path(temporary))
            self.assertEqual(archive.name, "Gitra-Windows-x64.zip")
            with zipfile.ZipFile(archive) as bundle:
                prefix = "Gitra-Windows-x64/"
                self.assertEqual(set(bundle.namelist()), {prefix + name for name in ["gitra.exe", "Gitra.cmd", "README.txt", "README.zh-CN.txt"]})
                self.assertEqual(bundle.read(prefix + "gitra.exe"), b"MZ-test-binary")
                launcher = bundle.read(prefix + "Gitra.cmd")
                self.assertIn(b"\r\n", launcher)
                self.assertNotIn(b"\n", launcher.replace(b"\r\n", b""))
                for name in ["README.txt", "README.zh-CN.txt"]:
                    self.assertIn("Git", bundle.read(prefix + name).decode("utf-8"))

    def test_compiler_failure_does_not_publish_archive(self):
        with tempfile.TemporaryDirectory() as temporary, patch.object(builder.subprocess, "run", side_effect=subprocess.CalledProcessError(1, "go")):
            with self.assertRaises(subprocess.CalledProcessError):
                builder.build("test", Path(temporary))
            self.assertFalse((Path(temporary) / "Gitra-Windows-x64.zip").exists())


if __name__ == "__main__":
    unittest.main()
