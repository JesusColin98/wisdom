# Wisdom Ecosystem: Security & Connectivity Architecture

This document describes the implemented security architecture for the Wisdom ecosystem on Google Cloud Run, following the principle of **Least Privilege**.

## 1. Authentication Strategy (OIDC)

Services in the ecosystem communicate using **OpenID Connect (OIDC)** tokens. This ensures that endpoints remain private and only authorized identities can invoke them.

### Implementation in Python (Chat Agent)
The Python Chat Agent fetches an identity token from the Google Cloud Metadata Server:
```python
def get_id_token(audience: str):
    metadata_url = f"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience={audience}"
    headers = {"Metadata-Flavor": "Google"}
    resp = requests.get(metadata_url, headers=headers)
    return resp.text
```
This token is then passed in the `Authorization: Bearer <token>` header for all requests to the Wisdom Engine.

## 2. Identity and Access Management (IAM)

We avoid using `allUsers` to comply with organizational policies (Domain Restricted Sharing). Instead, we use specific identity bindings.

### Service-to-Service
- **Identity:** `nexusstate-sa@jesus-mvp.iam.gserviceaccount.com` (Runtime Service Account)
- **Permission:** `roles/run.invoker` on the `wisdom-engine` service.
- **Result:** The Chat Agent can call the Engine, but the Engine is not public.

### User-to-Service
- **Identity:** User LDAP (e.g., `jesuscolin@google.com`)
- **Permission:** `roles/run.invoker` on `wisdom-portal` and `wisdom-chat-agent`.
- **Result:** Only the authorized user can access the frontend and the agent endpoint.

## 3. Automated Deployment

The `scripts/deploy_all.sh` script handles the secure configuration automatically:
1. Deploys services with `--no-allow-unauthenticated`.
2. Resolves the current user identity.
3. Applies the necessary IAM policy bindings.

## 4. Root Cause Analysis of Previous Failures

| Symptom | Root Cause | Resolution |
| :--- | :--- | :--- |
| `Setting IAM policy failed` | Domain Restricted Sharing policy blocked `allUsers`. | Used specific IAM bindings for SA and User. |
| Container failed to start | `IndentationError` and literal `...` in `main.py`. | Fixed Python syntax and cleaned code block. |
| 403 Forbidden | Request missing valid OIDC token. | Integrated OIDC token retrieval in `handle_mcp_call`. |

---
*Maintained by Gemini CLI for project Brujula.*
