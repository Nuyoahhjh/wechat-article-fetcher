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
loginData = resp.json()
print("Login response keys:", list(loginData.keys()))
print("Login response:", json.dumps(loginData, indent=2)[:300])

# Try different token field names
token = loginData.get("token") or loginData.get("access_token") or loginData.get("jwt") or ""
if token:
    print(f"\nToken: {token[:30]}...")

    # Try agents endpoint
    for auth_header in [f"Bearer {token}", f"Token {token}", token]:
        resp2 = requests.get("http://127.0.0.1:18080/api/v1/agents",
                           headers={"Authorization": auth_header}, timeout=5)
        print(f"Agents with '{auth_header[:20]}...': {resp2.status_code}")
        if resp2.status_code == 200:
            print(f"  Response: {resp2.text[:200]}")
            break

proc.terminate()
proc.wait()
print("\nDone")
