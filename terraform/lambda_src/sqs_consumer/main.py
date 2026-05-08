import json
import logging

logger = logging.getLogger()
logger.setLevel(logging.INFO)


def handler(event, _context):
    failures = []
    for record in event.get("Records", []):
        message_id = record.get("messageId")
        body = record.get("body", "")
        try:
            logger.info("processing message_id=%s body=%s", message_id, body)
            payload = json.loads(body) if body else {}
            if payload.get("forceFail"):
                raise ValueError("forced failure")
        except Exception:
            logger.exception("failed message_id=%s", message_id)
            failures.append({"itemIdentifier": message_id})

    return {"batchItemFailures": failures}
