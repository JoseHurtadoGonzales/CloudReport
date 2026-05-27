"""
PPTX worker (python-pptx + Jinja2 placeholders).

Strategy: load the .pptx template, walk every shape, and run Jinja2 on each
text-frame's runs. This is conservative — it doesn't add or remove slides,
just substitutes `{{ vars }}` in existing text. For more advanced templating
we'd want a dedicated lib (pptx-template), but this covers the common case.

Input  (job.data):
    {
        "data":    {...},
        "options": { "templateAssetKey": "assets/abc.pptx" }
    }
"""
from __future__ import annotations

import io
import os
import sys

sys.path.insert(0, "/app/shared")

from protocol import Job, Reply, S3, consume  # noqa: E402
from pptx import Presentation  # noqa: E402
from jinja2 import Environment  # noqa: E402

try:
    from libreoffice import convert_to_pdf  # noqa: E402
    HAS_SOFFICE = True
except Exception:
    HAS_SOFFICE = False


s3 = S3()
jinja_env = Environment(autoescape=False)


def _render_runs(shape, data):
    if not shape.has_text_frame:
        return
    for paragraph in shape.text_frame.paragraphs:
        # We render the paragraph text as a single template, then replace
        # the runs while preserving the first run's formatting.
        original = "".join(run.text for run in paragraph.runs)
        if "{{" not in original and "{%" not in original:
            continue
        try:
            rendered = jinja_env.from_string(original).render(**data)
        except Exception:
            continue
        if paragraph.runs:
            paragraph.runs[0].text = rendered
            for run in paragraph.runs[1:]:
                run.text = ""


def render(job: Job) -> Reply:
    data = job.data or {}
    options = data.get("options") or {}
    payload = data.get("data") or {}
    if not isinstance(payload, dict):
        payload = {"data": payload}

    asset_key = options.get("templateAssetKey")
    if not asset_key:
        return Reply(id=job.id, status="error",
                     error="pptx: options.templateAssetKey is required")

    tpl_bytes = s3.get(asset_key)
    prs = Presentation(io.BytesIO(tpl_bytes))

    for slide in prs.slides:
        for shape in slide.shapes:
            _render_runs(shape, payload)
            if shape.has_table:
                for row in shape.table.rows:
                    for cell in row.cells:
                        _render_runs(cell, payload)

    out = io.BytesIO()
    prs.save(out)
    out_bytes = out.getvalue()

    export_pdf = bool(options.get("exportPdf", False))
    if export_pdf:
        if not HAS_SOFFICE:
            return Reply(id=job.id, status="error",
                         error="exportPdf requires LibreOffice in the worker image")
        try:
            out_bytes = convert_to_pdf(out_bytes, "pptx")
        except Exception as e:
            return Reply(id=job.id, status="error",
                         error=f"pptx→pdf failed: {e}")
        mime = "application/pdf"
        prefix = "outputs/pptx-pdf"
    else:
        mime = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
        prefix = "outputs/pptx"

    key = s3.put(prefix, out_bytes, mime)
    return Reply(id=job.id, status="success", outputBlob=key, mimeType=mime)


def main() -> None:
    stream = os.environ.get("WORKER_STREAM", "renders:pptx")
    group = os.environ.get("WORKER_GROUP", "pptx-workers")
    consume(stream, group, render)


if __name__ == "__main__":
    main()
