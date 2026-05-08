import json
import boto3

s3 = boto3.client("s3")


def _read_worker_result(bucket, key):
    obj = s3.get_object(Bucket=bucket, Key=key)
    return json.loads(obj["Body"].read().decode("utf-8"))


def handler(event, _context):
    current_results = event.get("results", [])
    all_results = event.get("allResults", [])
    failed_workers = []
    failed_workers_final = event.get("failedWorkersFinal", [])
    results_s3_bucket = event.get("resultsS3Bucket", "")

    normalized_results = []

    for item in current_results:
        status = str(item.get("status", "")).upper()
        if status == "TASK_SUCCEEDED":
            result_s3_key = item.get("resultS3Key", "")
            if not results_s3_bucket or not result_s3_key:
                normalized = {
                    "workerId": item.get("workerId"),
                    "payload": item.get("payload", {}),
                    "attempt": item.get("attempt", 0),
                    "status": "FAILED",
                    "errorType": "RETRYABLE",
                    "message": "missing resultsS3Bucket or resultS3Key",
                    "productTotal": 0,
                    "productSucceeded": 0,
                    "queuedCount": 0
                }
            else:
                try:
                    normalized = _read_worker_result(results_s3_bucket, result_s3_key)
                    normalized["attempt"] = item.get("attempt", 0)
                    normalized["resultS3Key"] = result_s3_key
                except Exception as exc:
                    normalized = {
                        "workerId": item.get("workerId"),
                        "payload": item.get("payload", {}),
                        "attempt": item.get("attempt", 0),
                        "status": "FAILED",
                        "errorType": "RETRYABLE",
                        "message": f"failed to read worker result from s3: {exc}",
                        "productTotal": 0,
                        "productSucceeded": 0,
                        "queuedCount": 0
                    }
        else:
            normalized = item

        normalized_results.append(normalized)
        status = str(normalized.get("status", "")).upper()
        if status != "SUCCEEDED":
            worker = {
                "workerId": normalized.get("workerId"),
                "payload": normalized.get("payload", {}),
                "errorType": normalized.get("errorType", "RETRYABLE"),
                "message": normalized.get("message", "")
            }
            if worker["errorType"] == "RETRYABLE":
                failed_workers.append(worker)
            else:
                failed_workers_final.append(worker)

    all_results = all_results + normalized_results

    retry_count = int(event.get("retryCount", 0))
    retry_limit = int(event.get("retryLimit", 0))
    if retry_count >= retry_limit:
        failed_workers_final = failed_workers_final + failed_workers
        failed_workers = []

    return {
        "jobId": event.get("jobId"),
        "text": event.get("text"),
        "retryLimit": retry_limit,
        "retryCount": retry_count,
        "workers": normalized_results,
        "failedWorkers": failed_workers,
        "failedWorkersFinal": failed_workers_final,
        "results": all_results,
        "allResults": all_results
    }
