# Project Wisdom: Technical Debt & Roadmap

## 1. Core Engine (Algorithmic & Scalability)
- [ ] **Dynamic Topic Clustering:** Enhance REM cycle to automatically cluster findings into namespaces instead of `ns-general`.
- [ ] **Spaced Repetition Scheduler:** Implement a "forgetting curve" logic in Thalamus to resurface nodes (vocabulary, chess tactics) based on `last_reviewed` metadata.
- [ ] **RPForest Partitioning:** Advanced partitioning using centroids for higher accuracy.

## 2. Communication & UI
- [ ] **WebSocket Layer:** Implement a WebSocket sub-system in `pkg/api` for real-time chat and event streaming.
- [ ] **Gemini Live Integration:** Create a bridge for Realtime Audio/Video processing to enable voice tutoring.
- [ ] **Neural Atlas Visualizer:** Real-time graph rendering for the UI.

## 3. Specialized Skills (Skill Hub)
- [ ] **Chess Skill:** Logic for ELO tracking, PGN parsing, and specific "weakness" node analysis.
- [ ] **Language Coach Skill:** Vocabulary repetition algorithms and level-appropriate content generation.
- [ ] **Troubleshooting Skill:** Deep integration with `Production` MCP to automatically distill bug patterns and monitoring metrics into Cortex nodes.

## 4. Ingestion & Connectivity
- [ ] **Universal Document Ingestor:** Add support for PDF/EPUB parsing into Cortex nodes.
- [ ] **Internet Search MCP:** Bridge to search engines to feed the Cortex with external facts.
- [ ] **Codebase Mapper:** Integration with Gemini CLI's `architect` sub-agent to automatically populate the Neural Atlas with function/file relationships.

## 5. Metadata & Personalization
- [ ] **Identity Management:** Native support for `Person` nodes with historical role/strength tracking.
- [ ] **Dopamine Loop:** Token efficiency monitoring and TSR-based alerts.

## 6. Deployment & Infrastructure (Immediate Fixes)
- [x] **Restore/Fix Default Compute Service Account:** 
    - **Status:** BYPASS IMPLEMENTED. The default compute SA is permanently purged. We are now using `cloud-run-build-sa` for builds and `nexusstate-sa` for the service.
    - **Evidence:** `deploy_fix.sh` successfully overrides defaults.
    - **Action:** Continue using user-managed service accounts as per Duckie's recommendation.
- [x] **Automate Build via GCloud Builds:**
    - **Status:** COMPLETED. All deployments use `gcloud builds submit`.
    - **Action:** `deploy_fix.sh` is the source of truth for deployments.
- [x] **Fix Cloud Run Permissions:**
    - **Status:** COMPLETED. `roles/iam.serviceAccountUser` granted to jesuscolin@google.com on both build and service SAs.
- [x] **GCS Fuse for SQLite Persistence:**
    - **Status:** COMPLETED. Bucket `gs://wisdom-cortex-jesus-mvp` created and mounted to `/mnt/wisdom-cortex/wisdom.db`.

## 7. Research & Limitations (Duckie/Panteon Findings)
- **Gaia ID Not Found:** This error confirms the default Compute Engine SA (`374889098326-compute@developer.gserviceaccount.com`) is deleted or purged.
- **Restoration window:** If deleted < 30 days, can be restored via `gcloud beta iam service-accounts undelete UNIQUE_ID`. If > 30 days, it is permanently purged.
- **Permanent Name Lock:** You **cannot** recreate a service account with the exact same `...@developer.gserviceaccount.com` email format manually.
- **UI Dependency:** Cloud Run "Continuous Deployment" UI button is HARD-CODED to expect the original default SA format and will fail if it's missing.
- **Cloud Build Default:** Newer projects default to using the GCE SA for builds. If missing, `gcloud builds submit` fails unless `--service-account` is explicitly provided AND the staging bucket permissions are fixed.
- **Bypass Strategy:** Use user-managed service accounts (`cloud-run-build-sa`, `nexusstate-sa`) and explicitly override all defaults in CLI/Terraform.
