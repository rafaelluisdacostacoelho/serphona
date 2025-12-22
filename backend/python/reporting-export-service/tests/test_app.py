from fastapi.testclient import TestClient

from reporting_export.main import app

client = TestClient(app)


def test_health_endpoint():
    response = client.get("/health")
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "healthy"
    assert body["service"] == "reporting-export-service"


def test_export_lifecycle_returns_placeholder_download():
    payload = {
        "report_type": "calls",
        "tenant_id": "tenant-1",
        "start_date": "2024-01-01",
        "end_date": "2024-01-31",
        "format": "csv",
    }

    created = client.post("/api/v1/exports", json=payload)
    assert created.status_code == 200
    job = created.json()
    assert job["status"] == "pending"
    assert job["job_id"].startswith("exp_")

    status = client.get(f"/api/v1/exports/{job['job_id']}")
    assert status.status_code == 200
    status_body = status.json()
    assert status_body["download_url"].endswith(f"{job['job_id']}/download")

    download = client.get(f"/api/v1/exports/{job['job_id']}/download")
    assert download.status_code == 200
    assert "text/csv" in download.headers.get("content-type", "")
    assert b"id,timestamp,value" in download.content


def test_deliveries_are_accepted_without_external_io():
    payload = {
        "job_id": "exp_fake",
        "destination_type": "s3",
        "destination_config": {"bucket": "tests"},
    }
    response = client.post("/api/v1/deliveries", json=payload)
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "pending"
    assert body["destination_type"] == "s3"
