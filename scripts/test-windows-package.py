"""Run the extracted Windows executable and launcher on a real Windows host."""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile
import zipfile


def run(command, **kwargs):
    return subprocess.run(command, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=30, **kwargs)


def verify(archive, version):
    with tempfile.TemporaryDirectory(prefix="gitra 中文 package ") as temporary:
        root = Path(temporary)
        with zipfile.ZipFile(archive) as bundle:
            bundle.extractall(root)
        app = root / "Gitra-Windows-x64"
        cmd = str(Path(os.environ["SystemRoot"]) / "System32/cmd.exe")
        env = dict(os.environ, GITRA_CONFIG_DIR=str(root / "config"))
        binary = run([str(app / "gitra.exe"), "--version"], env=env)
        assert binary.returncode == 0, binary.stderr
        assert binary.stdout.strip() == "gitra " + version, binary.stdout
        # The outer quote pair is required by cmd /s /c for a quoted script path.
        command = f'"{cmd}" /d /s /c ""{app / "Gitra.cmd"}" --version"'
        launched = run(command, env=env, input="\n")
        assert launched.returncode == 0, launched.stdout + launched.stderr
        assert "gitra " + version in launched.stdout, launched.stdout
        missing = run(command, env=dict(env, PATH=str(root)), input="\n")
        assert missing.returncode == 1, missing.stdout + missing.stderr
        assert "Install Git for Windows" in missing.stdout, missing.stdout
        command = f'"{cmd}" /d /s /c ""{app / "Gitra.cmd"}" invalid-command"'
        failed = run(command, env=env, input="\n")
        assert failed.returncode != 0, failed.stdout + failed.stderr
        print("PASS: extracted version, Unicode/space launcher, missing Git and failed command")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--archive", type=Path, default=Path("dist/Gitra-Windows-x64.zip"))
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    verify(args.archive, args.version)
