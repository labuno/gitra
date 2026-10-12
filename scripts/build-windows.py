"""Build the Windows x64 portable archive using Go and Python's standard library."""
import argparse
import os
from pathlib import Path
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parent.parent
PACKAGE = "Gitra-Windows-x64"


def compile_binary(version, output):
    env = dict(os.environ, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
    flags = "-s -w -X github.com/zhanhd/gitra/internal/version.Version=" + version
    subprocess.run(["go", "build", "-trimpath", "-ldflags", flags, "-o", str(output), "./cmd/gitra"], cwd=ROOT, env=env, check=True)


def build(version, output_dir):
    output_dir.mkdir(parents=True, exist_ok=True)
    archive = output_dir / (PACKAGE + ".zip")
    with tempfile.TemporaryDirectory(prefix="gitra-windows-build-") as temporary:
        stage = Path(temporary)
        compile_binary(version, stage / "gitra.exe")
        for name in ["Gitra.cmd", "README.txt", "README.zh-CN.txt"]:
            contents = (ROOT / "scripts/windows" / name).read_text(encoding="utf-8")
            (stage / name).write_bytes(contents.replace("\n", "\r\n").encode("utf-8"))
        with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
            for file in sorted(stage.iterdir()):
                bundle.write(file, PACKAGE + "/" + file.name)
    return archive


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", default=os.environ.get("VERSION", "dev"))
    parser.add_argument("--output-dir", type=Path, default=ROOT / "dist")
    args = parser.parse_args()
    print(build(args.version, args.output_dir))
