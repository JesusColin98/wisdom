# Wisdom Project Security Configuration

## Overview

This document tracks the security configurations applied to the Wisdom project to restrict unauthorized access, specifically detailing the implementation of Identity-Aware Proxy (IAP) to protect the application behind a Google Login prompt.

## Identity-Aware Proxy (IAP) Implementation

To prevent unauthorized access (such as the suspicious IP interactions detected on `2026-05-10`), we implemented Google Cloud Identity-Aware Proxy (IAP) on the Global External Load Balancer. This ensures all traffic is authenticated via Google Workspace/Gmail accounts before reaching the underlying Cloud Run services.

### 1. Securing Cloud Run Services (Defense in Depth)

We restricted the network "Ingress" of the Cloud Run instances to ensure users cannot bypass the Load Balancer and IAP by calling the `*.run.app` URLs directly.

*   Set Ingress to **Internal and Cloud Load Balancing** on:
    *   `wisdom-portal`
    *   `wisdom-chat-agent`
    *   `wisdom-engine`
*   With the network restricted, the IAM policy is set to allow `allUsers` to invoke the services. This allows the Load Balancer to fetch the application *after* IAP has authenticated the user. The public internet cannot reach the services due to the Ingress restriction.

### 2. OAuth Consent and Client Credentials Setup

IAP requires an OAuth Client to handle the authentication flow.

1.  **OAuth Consent Screen:**
    *   Configured as **External**.
    *   Restricted to specific **Test users** (`jealcovi98@gmail.com`, `jesuscolin2025@gmail.com`) to prevent anyone else from even seeing the login prompt successfully.
2.  **OAuth Client ID:**
    *   Created a **Web application** credential.
    *   **Authorized redirect URIs** configured exactly as:
        `https://iap.googleapis.com/v1/oauth/clientIds/<CLIENT_ID>:handleRedirect`
        *(Example: `https://iap.googleapis.com/v1/oauth/clientIds/384412501694-q5h6p4r8764ilng1i2l4s6q8m2lic3e0.apps.googleusercontent.com:handleRedirect`)*

*(Note: The actual Client Secret is securely stored in local uncommitted memory (`MEMORY.md`) and GCP Secrets Manager, never in source control).*

### 3. Provisioning the IAP Service Identity

For IAP to communicate with Cloud Run, a specific Google-managed service account must be created and authorized.

1.  **Created the Service Identity:**
    ```bash
    gcloud beta services identity create --service=iap.googleapis.com --project=jesuscolin2025-678c7
    ```
2.  **Authorized IAP to Invoke Cloud Run:**
    Granted the `roles/run.invoker` role to the newly created service account (`service-384412501694@gcp-sa-iap.iam.gserviceaccount.com`) on all protected Cloud Run services.

### 4. Enabling IAP on the Load Balancer

With the OAuth credentials ready, IAP was enabled on the External Load Balancer's backend services:

*   `wisdom-portal-backend`
*   `wisdom-chat-agent-backend`
*   `wisdom-engine-backend`

Command used:
```bash
gcloud compute backend-services update <backend-name> \
  --global \
  --iap="enabled,oauth2-client-id=<CLIENT_ID>,oauth2-client-secret=<SECRET>"
```

### 4. Configuring IAP Access Policies

Finally, we instructed IAP on *who* is allowed to pass through the proxy once they authenticate with Google.

*   Granted the `roles/iap.httpsResourceAccessor` role to the authorized users on all three backends:
    *   `jealcovi98@gmail.com`
    *   `jesuscolin2025@gmail.com`

## Result

Accessing the portal at `https://34-49-82-216.nip.io/` now intercepts the request and redirects the user to Google Sign-In. Only the explicitly whitelisted emails can successfully authenticate and reach the application.