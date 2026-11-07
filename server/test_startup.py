import subprocess, time, requests, json, sys, os, signal
from pathlib import Path

os.chdir(Path(__file__).resolve().parent)
env = dict(os.environ)
env["DB_PATH"] = ":memory:"
env["HTTP_ADDR"] = ":18080"

proc = subprocess.Popen(["./server_test.exe"], env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
time.sleep(3)

print("=== Health Check ===")
try:
    resp = requests.get("http://127.0.0.1:18080/health", timeout=5)
    print(f"Health: {resp.status_code} {resp.text[:100]}")
except Exception as e:
    print(f"Health: {e}")

print("\n=== Login ===")
try:
    resp = requests.post("http://127.0.0.1:18080/api/v1/auth/login",
                        json={"username": "admin", "password": "admin123!"},
                        headers={"Content-Type": "application/json"}, timeout=5)
    print(f"Login: {resp.status_code}")
    if resp.status_code == 200:
        token = resp.json().get("token", "")
        print(f"Token received: {token[:20]}...")
    else:
        print(f"Response: {resp.text[:200]}")
except Exception as e:
    print(f"Login: {e}")

print("\n=== Agents List ===")
try:
    token = requests.post("http://127.0.0.1:18080/api/v1/auth/login",
                         json={"username": "admin", "password": "admin123!"},
                         headers={"Content-Type": "application/json"}, timeout=5).json().get("token", "")
    resp = requests.get("http://127.0.0.1:18080/api/v1/agents",
                       headers={"Authorization": f"Bearer {token}"}, timeout=5)
    print(f"Agents: {resp.status_code} {resp.text[:200]}")
except Exception as e:
    print(f"Agents: {e}")

proc.terminate()
proc.wait()
print("\nServer stopped")
