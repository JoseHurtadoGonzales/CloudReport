"""
Chromium PDF worker (Playwright).

Renders HTML → PDF using a real headless Chromium, so we get:
  - JavaScript-rendered content (Chart.js, ECharts, etc.)
  - Web fonts
  - displayHeaderFooter with <span class="pageNumber"></span> etc.
  - Wait for selector / network idle
  - Scale, margins per side, page ranges, format / orientation
  - emulate media (screen/print)

Input  (job.data):
    {
        "html": "<rendered-html>",
        "data": { ... },                 # already resolved by the orchestrator
        "options": {                      # all optional
            "format": "A4",
            "landscape": false,
            "margin": { "top": "1cm", "right": "1cm", "bottom": "1cm", "left": "1cm" },
            "pageRanges": "1-5",
            "scale": 1.0,
            "printBackground": true,
            "preferCSSPageSize": false,
            "displayHeaderFooter": false,
            "headerTemplate": "<div ...></div>",
            "footerTemplate": "<div ...></div>",
            "emulateMediaType": "print",
            "waitForNetworkIdle": false,
            "waitForSelector": ".ready",
            "timeout": 30000
        }
    }
"""
from __future__ import annotations

import os
import sys

sys.path.insert(0, "/app/shared")

from protocol import Job, Reply, S3, consume  # noqa: E402

from playwright.sync_api import sync_playwright, TimeoutError as PWTimeout  # noqa: E402


s3 = S3()
_browser = None


def get_browser():
    """Lazy single Chromium instance, reused across jobs to avoid the ~1s
    startup cost on every render."""
    global _browser
    if _browser is None:
        pw = sync_playwright().start()
        _browser = pw.chromium.launch(
            args=[
                "--no-sandbox",
                "--disable-setuid-sandbox",
                "--disable-dev-shm-usage",
                "--font-render-hinting=none",
            ],
        )
    return _browser


def render(job: Job) -> Reply:
    data = job.data or {}
    html = data.get("html") or ""
    opts = data.get("options") or {}

    timeout_ms = int(opts.get("timeout", 30000))

    browser = get_browser()
    context = browser.new_context()
    page = context.new_page()
    try:
        # emulate media
        if opts.get("emulateMediaType"):
            page.emulate_media(media=str(opts["emulateMediaType"]))
        # set HTML
        page.set_content(html, wait_until="domcontentloaded", timeout=timeout_ms)

        # wait for selector
        if opts.get("waitForSelector"):
            page.wait_for_selector(
                str(opts["waitForSelector"]),
                state="visible",
                timeout=timeout_ms,
            )
        # wait for network idle (optional, redundant with set_content but explicit)
        if opts.get("waitForNetworkIdle"):
            page.wait_for_load_state("networkidle", timeout=timeout_ms)

        # ── build PDF options
        pdf_kwargs: dict = {
            "format":           opts.get("format", "A4"),
            "landscape":        bool(opts.get("landscape", False)),
            "scale":            float(opts.get("scale", 1.0)),
            "print_background": bool(opts.get("printBackground", True)),
            "prefer_css_page_size": bool(opts.get("preferCSSPageSize", False)),
        }
        m = opts.get("margin") or {}
        if isinstance(m, dict):
            pdf_kwargs["margin"] = {
                "top":    str(m.get("top",    "1cm")),
                "right":  str(m.get("right",  "1cm")),
                "bottom": str(m.get("bottom", "1cm")),
                "left":   str(m.get("left",   "1cm")),
            }
        if opts.get("pageRanges"):
            pdf_kwargs["page_ranges"] = str(opts["pageRanges"])
        if opts.get("displayHeaderFooter"):
            pdf_kwargs["display_header_footer"] = True
            pdf_kwargs["header_template"] = opts.get("headerTemplate") or "<div></div>"
            pdf_kwargs["footer_template"] = opts.get("footerTemplate") or "<div></div>"
        # outline (Chrome 105+)
        if opts.get("outline"):
            pdf_kwargs["tagged"] = True  # required for outline
        if opts.get("taggedPdf"):
            pdf_kwargs["tagged"] = True

        try:
            pdf_bytes = page.pdf(**pdf_kwargs)
        except PWTimeout as e:
            return Reply(id=job.id, status="error",
                         error=f"chromium timeout after {timeout_ms}ms: {e}")
    finally:
        try:
            context.close()
        except Exception:
            pass

    key = s3.put("outputs/chromium", pdf_bytes, "application/pdf")
    return Reply(id=job.id, status="success", outputBlob=key,
                 mimeType="application/pdf")


def main() -> None:
    stream = os.environ.get("WORKER_STREAM", "renders:chrome-pdf")
    group = os.environ.get("WORKER_GROUP", "chromium-workers")
    consume(stream, group, render)


if __name__ == "__main__":
    main()
