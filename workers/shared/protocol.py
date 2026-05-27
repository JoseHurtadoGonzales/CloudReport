"""
Shared protocol between the Go orchestrator and Python workers.

Jobs flow through Redis Streams:

  - request stream:  "renders:<recipe>"  (XADD by Go, XREADGROUP by worker)
  - reply  stream:   "renders:replies:<jobId>"  (XADD by worker, XREAD by Go)

Each XADD carries a single field "payload" whose value is the JSON below.
"""
from __future__ import annotations

import json
import os
import time
import uuid
from dataclasses import dataclass, field, asdict
from typing import Any, Optional

import boto3
import redis
from botocore.config import Config as BotoConfig


# ---------- Job / Reply models ----------

@dataclass
class Job:
    id: str
    recipe: str
    data: Optional[dict] = None    # arbitrary recipe input (incl. html, data, options)
    options: Optional[dict] = None
    replyTo: str = ""
    deadlineMs: int = 0
    # office templates pass the binary template via S3 key:
    inputBlob: Optional[str] = None

    @classmethod
    def parse(cls, raw: str) -> "Job":
        obj = json.loads(raw)
        # The Go side wraps the recipe-specific payload inside `data` which is
        # a JSON-encoded {html, data, options} blob. Decode it here for
        # convenience.
        d = obj.get("data")
        if isinstance(d, str):
            try:
                obj["data"] = json.loads(d)
            except Exception:
                pass
        elif isinstance(d, (bytes, bytearray)):
            try:
                obj["data"] = json.loads(d.decode("utf-8"))
            except Exception:
                pass
        return cls(
            id=obj.get("id", ""),
            recipe=obj.get("recipe", ""),
            data=obj.get("data"),
            options=obj.get("options"),
            replyTo=obj.get("replyTo", ""),
            deadlineMs=obj.get("deadlineMs", 0),
            inputBlob=obj.get("inputBlob"),
        )


@dataclass
class Reply:
    id: str
    status: str  # "success" | "error"
    outputBlob: str = ""
    mimeType: str = ""
    durationMs: int = 0
    error: str = ""
    logs: str = ""

    def to_redis_value(self) -> dict:
        return {"payload": json.dumps(asdict(self))}


# ---------- S3 (SeaweedFS) helper ----------

class S3:
    def __init__(self) -> None:
        self.bucket = os.environ.get("S3_BUCKET", "cloudreport")
        endpoint = os.environ.get("WEED_S3", "http://seaweedfs-s3:8333")
        self.client = boto3.client(
            "s3",
            endpoint_url=endpoint,
            aws_access_key_id=os.environ.get("S3_ACCESS_KEY", "cloudreport"),
            aws_secret_access_key=os.environ.get("S3_SECRET_KEY", "cloudreport123"),
            region_name=os.environ.get("S3_REGION", "us-east-1"),
            config=BotoConfig(s3={"addressing_style": "path"}, signature_version="s3v4"),
        )
        self._ensure_bucket()

    def _ensure_bucket(self) -> None:
        try:
            self.client.head_bucket(Bucket=self.bucket)
        except Exception:
            try:
                self.client.create_bucket(Bucket=self.bucket)
            except Exception:
                pass

    def get(self, key: str) -> bytes:
        obj = self.client.get_object(Bucket=self.bucket, Key=key)
        return obj["Body"].read()

    def put(self, prefix: str, body: bytes, content_type: str) -> str:
        key = f"{prefix}/{int(time.time())}-{uuid.uuid4().hex[:12]}"
        self.client.put_object(
            Bucket=self.bucket, Key=key, Body=body, ContentType=content_type,
        )
        return key


# ---------- Redis client + consumer loop ----------

def redis_client() -> "redis.Redis":
    url = os.environ.get("REDIS_URL", "redis://redis:6379/0")
    return redis.from_url(url, decode_responses=True)


def consume(
    stream: str,
    group: str,
    handler,
    consumer: str = "",
    block_ms: int = 5000,
) -> None:
    """Block in a consumer loop, calling `handler(job: Job) -> Reply` for each
    incoming job. Replies are XADDed onto job.replyTo and acked.
    """
    r = redis_client()
    consumer = consumer or f"c-{uuid.uuid4().hex[:8]}"

    # Ensure stream + group exist.
    try:
        r.xgroup_create(name=stream, groupname=group, id="0", mkstream=True)
    except redis.ResponseError as e:
        if "BUSYGROUP" not in str(e):
            raise

    print(f"[worker] listening stream={stream} group={group} consumer={consumer}", flush=True)
    while True:
        try:
            resp = r.xreadgroup(
                groupname=group, consumername=consumer,
                streams={stream: ">"}, count=1, block=block_ms,
            )
        except Exception as e:
            print(f"[worker] xreadgroup error: {e}", flush=True)
            time.sleep(1)
            continue
        if not resp:
            continue
        for _stream, messages in resp:
            for msg_id, fields in messages:
                start = time.time()
                payload = fields.get("payload", "{}")
                try:
                    job = Job.parse(payload)
                    reply = handler(job)
                    reply.durationMs = int((time.time() - start) * 1000)
                    if reply.status == "":
                        reply.status = "success"
                except Exception as e:
                    reply = Reply(
                        id=getattr(locals().get("job"), "id", ""),
                        status="error", error=f"{type(e).__name__}: {e}",
                        durationMs=int((time.time() - start) * 1000),
                    )
                if not reply.id and "job" in locals():
                    reply.id = job.id
                try:
                    r.xadd(job.replyTo, reply.to_redis_value())
                except Exception as e:
                    print(f"[worker] reply xadd error: {e}", flush=True)
                r.xack(stream, group, msg_id)
                r.xdel(stream, msg_id)
                print(f"[worker] handled job={reply.id} status={reply.status} dur={reply.durationMs}ms", flush=True)
