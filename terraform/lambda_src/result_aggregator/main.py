def handler(event, _context):
    current_results = event.get("results", [])
    all_results = event.get("allResults", [])
    failed_workers = []
    failed_workers_final = event.get("failedWorkersFinal", [])

    all_results = all_results + current_results

    for item in current_results:
        status = str(item.get("status", "")).upper()
        if status != "SUCCEEDED":
            worker = {
                "workerId": item.get("workerId"),
                "payload": item.get("payload", {}),
                "errorType": item.get("errorType", "RETRYABLE"),
                "message": item.get("message", "")
            }
            if worker["errorType"] == "RETRYABLE":
                failed_workers.append(worker)
            else:
                failed_workers_final.append(worker)

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
        "workers": current_results,
        "failedWorkers": failed_workers,
        "failedWorkersFinal": failed_workers_final,
        "results": all_results,
        "allResults": all_results
    }
