"""
Helper: convert an Office document (.docx, .xlsx, .pptx) to PDF using a
headless LibreOffice (`soffice --headless --convert-to pdf`).

Runs in the worker's container — Dockerfile must `apt install libreoffice-core
libreoffice-writer libreoffice-calc libreoffice-impress`.
"""
from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import uuid


def convert_to_pdf(src_bytes: bytes, src_ext: str) -> bytes:
    """Convert binary office content to PDF and return the bytes.
    `src_ext` is "docx" / "xlsx" / "pptx" (without dot)."""
    if shutil.which("soffice") is None:
        raise RuntimeError("LibreOffice (soffice) not installed in worker image")

    tmp = tempfile.mkdtemp(prefix="cr-soffice-")
    src_path = os.path.join(tmp, f"in-{uuid.uuid4().hex}.{src_ext}")
    out_dir = os.path.join(tmp, "out")
    os.makedirs(out_dir, exist_ok=True)

    try:
        with open(src_path, "wb") as f:
            f.write(src_bytes)
        # Each soffice invocation is independent — no shared profile.
        profile = f"file://{tmp}/profile"
        result = subprocess.run(
            [
                "soffice",
                "--headless",
                "--nologo",
                "--nofirststartwizard",
                f"-env:UserInstallation={profile}",
                "--convert-to", "pdf",
                "--outdir", out_dir,
                src_path,
            ],
            capture_output=True,
            timeout=90,
        )
        if result.returncode != 0:
            raise RuntimeError(
                f"soffice failed: {result.stderr.decode('utf-8', errors='replace')[:400]}"
            )
        # The output filename mirrors the input filename with .pdf extension.
        base = os.path.splitext(os.path.basename(src_path))[0]
        out_path = os.path.join(out_dir, f"{base}.pdf")
        if not os.path.exists(out_path):
            raise RuntimeError(f"soffice produced no output at {out_path}")
        with open(out_path, "rb") as f:
            return f.read()
    finally:
        shutil.rmtree(tmp, ignore_errors=True)
