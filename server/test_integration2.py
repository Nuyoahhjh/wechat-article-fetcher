import subprocess, time, requests, json, sys, os

os.chdir(r"D:\work\wechat-article-fetcher-v3\server")
env = dict(os.environ)
env["DB_PATH"] = ":memory:"
env["HTTP_ADDR"] = ":18080"

proc = subprocess.Popen(["./server_test.exe"], env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
time.sleep(3)

resp = requests.post("http://127.0.0.1:18080/api/v1/auth/login",
                    json={"username": "admin", "password": "admin123!"},
                    headers={"Content-Type": "application/json"}, timeout=5)
token = resp.json()["data"]["access_token"]
auth = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

# Create task with correct payload format (not stringified)
task_payload = {
    "type": "fetch_articles",
    "priority": 1,
    "payload": {
        "account_name": "RunningHUB",
        "max_articles": 5,
        "include_content": True
    }
}
resp = requests.post("http://127.0.0.1:18080/api/v1/tasks", headers=auth, json=task_payload, timeout=5)
print(f"Create Task: {resp.status_code}")
if resp.status_code == 201:
    taskData = resp.json().get("data", {})
    print(f"  Task ID: {taskData.get('id', 'N/A')}")
    print(f"  Status: {taskData.get('status', 'N/A')}")
    print(f"  Type: {taskData.get('type', 'N/A')}")

# Dashboard after creating task
resp = requests.get("http://127.0.0.1:18080/api/v1/dashboard", headers=auth, timeout=5)
print(f"Dashboard: {resp.json().get('data', {})}")

proc.terminate()
proc.wait()
print("\n=== INTEGRATION TEST COMPLETE ===")
