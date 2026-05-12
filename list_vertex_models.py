import subprocess
import json

def get_token():
    try:
        return subprocess.check_output(["gcloud", "auth", "print-access-token"], text=True, shell=True).strip()
    except:
        return None

def list_models():
    token = get_token()
    if not token:
        print("No token")
        return
    
    # Using the correct REST API endpoint for listing models in a location
    cmd = f'curl.exe -s -X GET -H "Authorization: Bearer {token}" "https://us-central1-aiplatform.googleapis.com/v1/projects/jesuscolin2025-678c7/locations/us-central1/publishers/google/models"'
    
    try:
        output = subprocess.check_output(cmd, text=True, shell=True)
        data = json.loads(output)
        models = data.get("models", [])
        for model in models:
            name = model.get("name", "")
            if "gemini" in name.lower():
                print(name)
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    list_models()
