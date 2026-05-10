#!/bin/bash
echo "Testing literal project ID..."
gcloud projects describe jesuscolin2025-678c7 --format='value(projectNumber)'
