"""
WeasyPrint worker: HTML -> PDF.

Input  (job.data):
    {
        "html":    "<html>...</html>",
        "data":    {...},                # already-resolved data (unused here)
        "options": { "pageSize": "A4", "presentationalHints": true, ... }
    }
Output:
    Uploads the PDF to SeaweedFS and returns the S3 key in reply.outputBlob.
"""
from __future__ import annotations

import io
import os
import sys

# Allow importing the shared module by file path.
sys.path.insert(0, "/app/shared")

from protocol import Job, Reply, S3, consume  # noqa: E402

import weasyprint  # noqa: E402


s3 = S3()


def render(job: Job) -> Reply:
    data = job.data or {}
    html = data.get("html", "")
    options = data.get("options") or {}

    base_url = options.get("baseUrl") or os.environ.get("BASE_URL", "")
    presentational_hints = bool(options.get("presentationalHints", True))

    doc = weasyprint.HTML(string=html, base_url=base_url or None)
    css = []
    extra_css = options.get("css")
    if isinstance(extra_css, str) and extra_css:
        css.append(weasyprint.CSS(string=extra_css))

    buf = io.BytesIO()
    doc.write_pdf(target=buf, stylesheets=css or None,
                  presentational_hints=presentational_hints)
    pdf_bytes = buf.getvalue()

    key = s3.put("outputs/weasyprint", pdf_bytes, "application/pdf")
    return Reply(id=job.id, status="success", outputBlob=key,
                 mimeType="application/pdf")


def main() -> None:
    stream = os.environ.get("WORKER_STREAM", "renders:weasyprint")
    group = os.environ.get("WORKER_GROUP", "weasyprint-workers")
    consume(stream, group, render)


if __name__ == "__main__":
    main()
