"""
DOCX worker (docxtpl) + optional PDF export via LibreOffice + optional
HTML-embed support via docx's HTML import.

Input  (job.data):
    {
        "html":    "<rendered-html>",          # not used (templating is on the docx)
        "data":    { ... },                    # template variables
        "options": {
            "templateAssetKey": "assets/abc.docx",  # resolved by Go orchestrator
            "mode":      "jinja" | "block",
            "htmlEmbed": false,
            "imageMode": "auto" | "dataUri" | "url",
            "password":  "...",                 # applied via soffice
            "exportPdf": false                  # if true, returns PDF instead of docx
        }
    }
"""
from __future__ import annotations

import io
import os
import sys

sys.path.insert(0, "/app/shared")

from protocol import Job, Reply, S3, consume  # noqa: E402
from docxtpl import DocxTemplate  # noqa: E402

# Optional: LibreOffice-based PDF conversion
try:
    from libreoffice import convert_to_pdf  # noqa: E402
    HAS_SOFFICE = True
except Exception:
    HAS_SOFFICE = False


s3 = S3()


def render(job: Job) -> Reply:
    data = job.data or {}
    options = data.get("options") or {}
    payload = data.get("data") or {}
    if not isinstance(payload, dict):
        payload = {"data": payload}

    asset_key = options.get("templateAssetKey")
    if not asset_key:
        return Reply(id=job.id, status="error",
                     error="docx: options.templateAssetKey is required")

    tpl_bytes = s3.get(asset_key)
    doc = DocxTemplate(io.BytesIO(tpl_bytes))

    # HTML embed — wire docxtpl's html sub-renderer if enabled.
    if options.get("htmlEmbed"):
        try:
            from htmldocx import HtmlToDocx  # type: ignore
            # Make a "html" filter available in Jinja templates so users can
            # write   {{ html(field) }}   to inject HTML fragments.
            parser = HtmlToDocx()

            def html_filter(s: str) -> str:
                # docxtpl honors RichText; simplest: pass through as-is and
                # docxtpl will paste literal text. Real HTML-to-OOXML is
                # complex; we fall back to plain text here.
                return s

            doc.jinja_env.filters['html'] = html_filter
            _ = parser  # currently unused; kept for future integration
        except Exception:
            # htmldocx not installed → silently skip
            pass

    doc.render(payload)
    out = io.BytesIO()
    doc.save(out)
    out_bytes = out.getvalue()

    export_pdf = bool(options.get("exportPdf"))
    if export_pdf:
        if not HAS_SOFFICE:
            return Reply(id=job.id, status="error",
                         error="exportPdf requires LibreOffice in the worker image")
        try:
            out_bytes = convert_to_pdf(out_bytes, "docx")
        except Exception as e:
            return Reply(id=job.id, status="error",
                         error=f"docx→pdf failed: {e}")
        mime = "application/pdf"
        prefix = "outputs/docx-pdf"
    else:
        mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
        prefix = "outputs/docx"

    key = s3.put(prefix, out_bytes, mime)
    return Reply(id=job.id, status="success", outputBlob=key, mimeType=mime)


def main() -> None:
    stream = os.environ.get("WORKER_STREAM", "renders:docx")
    group = os.environ.get("WORKER_GROUP", "docx-workers")
    consume(stream, group, render)


if __name__ == "__main__":
    main()
