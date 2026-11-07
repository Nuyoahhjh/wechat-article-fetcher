import subprocess, time, requests, json, sys, os
from pathlib import Path

os.chdir(Path(__file__).resolve().parent)
env = dict(os.environ)
env["DB_PATH"] = ":memory:"
env["HTTP_ADDR"] = ":18080"

proc = subprocess.Popen(["./server_test.exe"], env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
time.sleep(3)

# Login
resp = requests.post("http://127.0.0.1:18080/api/v1/auth/login",
                    json={"username": "admin", "password": "admin123!"},
                    headers={"Content-Type": "application/json"}, timeout=5)
token = resp.json()["data"]["access_token"]
auth = {"Authorization": f"Bearer {token}"}
print("Login: OK")

# Test API endpoints
endpoints = [
    ("GET", "/api/v1/agents", None),
    ("GET", "/api/v1/tasks", None),
    ("GET", "/api/v1/articles", None),
    ("GET", "/api/v1/dashboard/stats", None),
    ("GET", "/api/v1/dashboard/summary", None),
    ("POST", "/api/v1/tasks", {"account_name": "TestAccount", "max_articles": 5}),
]

for method, path, body in endpoints:
    try:
        if method == "GET":
            resp = requests.get(f"http://127.0.0.1:18080{path}", headers=auth, timeout=5)
        else:
            resp = requests.post(f"http://127.0.0.1:18080{path}", headers=auth, json=body, timeout=5)
        status = resp.status_code
        preview = resp.text[:100] if len(resp.text) > 0 else "(empty)"
        print(f"{method} {path}: {status} {preview}")
    except Exception as e:
        print(f"{method} {path}: ERROR {e}")

proc.terminate()
proc.wait()
print("\nAll API tests done")
