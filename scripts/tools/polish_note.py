import os
import sys
import json
import requests
import subprocess
import datetime
from typing import Optional

# Wisdom Ingestion Standards - Polishing Prompt
POLISH_PROMPT_TEMPLATE = """
You are a Wisdom Knowledge Engineer. Your task is to refactor the following Obsidian Markdown note to comply with the Wisdom Ingestion Standards (LIFT).

### CONSTRAINTS:
1. **YAML Frontmatter**: It MUST contain:
   - `id`: YYYYMMDDHHMMSS format (generate a unique one based on current time: {current_time})
   - `title`: A concise title.
   - `aliases`: List of alternative names.
   - `tags`: Relevant tags (PARA-style).
   - `mastery_score`: 0
   - `tipo`: (e.g., "concept", "project", "resource")
   - `estado`: "draft" or "polished"
   - `fase`: 1
   - `date`: {current_date}
2. **Atomic Structure**: Break down long prose into focused, atomic blocks.
3. **Heading Hierarchy**: Use ## for main sections and ### for sub-sections.
4. **Block IDs**: Append a unique `^blockid` (6 alphanumeric characters) to paragraphs or lists that are high-value for flashcards.
5. **Wikilinks**: Surround key concepts with [[Wikilinks]].
6. **Tone**: Clear, technical, yet accessible. SRE-style clarity.

### INPUT NOTE:
---
{content}
---

### OUTPUT:
Return ONLY the refactored Markdown content. No preamble, no explanation.
"""

def get_access_token():
    try:
        result = subprocess.run(['gcloud', 'auth', 'print-access-token'], capture_output=True, text=True, check=True, shell=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        print(f"Error getting access token: {e.stderr}", file=sys.stderr)
        sys.exit(1)

def polish_note(file_path: str, inplace: bool = False):
    if not os.path.exists(file_path):
        print(f"Error: File {file_path} not found.")
        return

    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    project_id = os.environ.get("GOOGLE_CLOUD_PROJECT", "jesuscolin2025-678c7")
    region = os.environ.get("GOOGLE_CLOUD_REGION", "us-central1")
    model_id = "gemini-3.1-pro-preview"
    
    token = get_access_token()
    url = f"https://{region}-aiplatform.googleapis.com/v1/projects/{project_id}/locations/{region}/publishers/google/models/{model_id}:generateContent"
    
    now = datetime.datetime.now()
    prompt = POLISH_PROMPT_TEMPLATE.format(
        current_time=now.strftime("%Y%m%d%H%M%S"),
        current_date=now.strftime("%Y-%m-%d"),
        content=content
    )

    payload = {
        "contents": [{
            "parts": [{"text": prompt}]
        }],
        "generationConfig": {
            "temperature": 0.2,
            "topP": 0.95,
            "maxOutputTokens": 8192
        }
    }

    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    print(f"Polishing {file_path} with {model_id}...")
    response = requests.post(url, headers=headers, json=payload)

    if response.status_code != 200:
        print(f"Error from Vertex AI: {response.status_code} - {response.text}")
        return

    result_json = response.json()
    try:
        polished_content = result_json['candidates'][0]['content']['parts'][0]['text']
        # Remove markdown code block wrappers if Gemini adds them
        if polished_content.startswith("```markdown"):
            polished_content = polished_content[11:].strip()
            if polished_content.endswith("```"):
                polished_content = polished_content[:-3].strip()
        elif polished_content.startswith("```"):
            polished_content = polished_content[3:].strip()
            if polished_content.endswith("```"):
                polished_content = polished_content[:-3].strip()
    except (KeyError, IndexError):
        print("Error: Could not parse response from Gemini.")
        print(json.dumps(result_json, indent=2))
        return

    if inplace:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(polished_content)
        print(f"Successfully polished and updated {file_path}")
    else:
        print("\n--- POLISHED CONTENT ---\n")
        print(polished_content)
        print("\n------------------------\n")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python polish_note.py <file_path> [--inplace]")
        sys.exit(1)
    
    path = sys.argv[1]
    do_inplace = "--inplace" in sys.argv
    polish_note(path, do_inplace)
