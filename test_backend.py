import requests
import subprocess
import os

SERVICE_URL = "https://wisdom-engine-3yamn3zhlq-uc.a.run.app"

def get_auth_token():
    try:
        token = subprocess.check_output(["gcloud", "auth", "print-identity-token"], 
                                      text=True, stderr=subprocess.DEVNULL, shell=True).strip()
        return token
    except Exception as e:
        print(f"Error getting token: {e}")
        return None

def test_chat():
    url = f"{SERVICE_URL}/chat"
    token = get_auth_token()
    if not token:
        print("No token available")
        return

    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    data = {"message": "Hello, this is a test from the bridge investigation script."}
    
    try:
        resp = requests.post(url, json=data, headers=headers)
        print(f"Status Code: {resp.status_code}")
        print(f"Response: {resp.text}")
    except Exception as e:
        print(f"Request failed: {e}")

if __name__ == "__main__":
    test_chat()
