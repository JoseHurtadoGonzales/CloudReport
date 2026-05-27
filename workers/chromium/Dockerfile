# Use Microsoft's playwright image which already bundles Chromium + all OS
# deps (fonts, libs). Saves a ~300MB install step on every build.
FROM mcr.microsoft.com/playwright/python:v1.47.0-jammy

WORKDIR /app
COPY chromium/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY chromium/app /app/app
COPY shared /app/shared

ENV PYTHONUNBUFFERED=1
CMD ["python", "/app/app/main.py"]
