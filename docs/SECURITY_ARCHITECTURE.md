# Wisdom Project Security Configuration

## Overview

This document tracks the security configurations applied to the Wisdom project to restrict unauthorized access.

## Identity-Aware Proxy (IAP) Implementation

To prevent unauthorized access (such as the suspicious IP interactions detected on `2026-05-10`), we have implemented Google Cloud Identity-Aware Proxy (IAP) on the Global External Load Balancer.

### Changes Made

1.  **Removed Public Cloud Run Access:**
    *   Removed `allUsers` from the `roles/run.invoker` policy on the `wisdom-portal` Cloud Run service.
    *   Explicitly granted `roles/run.invoker` to:
        *   `jealcovi98@gmail.com`
        *   `jesuscolin2025@gmail.com`
    *   *Note: The backend services (`wisdom-chat-agent`, `wisdom-engine`) currently still have `allUsers` invoke permissions at the Cloud Run level, relying on the Load Balancer to filter traffic. For defense-in-depth, these should also be restricted in the future to only allow internal traffic or the Load Balancer service account.*

2.  **Enabled IAP on Load Balancer Backends:**
    *   Enabled IAP on the following backend services:
        *   `wisdom-portal-backend`
        *   `wisdom-chat-agent-backend`
        *   `wisdom-engine-backend`

3.  **Configured IAP Access Policies:**
    *   Granted the `roles/iap.httpsResourceAccessor` role to allow access *only* to the authorized users on all three backends:
        *   `jealcovi98@gmail.com`
        *   `jesuscolin2025@gmail.com`

### Pending Configuration

The IAP setup is currently incomplete because it requires an **OAuth Client ID and Secret** to handle the Google Sign-In redirection. This must be configured manually in the GCP Console.

### Next Steps for OAuth Setup

(See conversation for instructions on generating the OAuth Client ID and Secret).
