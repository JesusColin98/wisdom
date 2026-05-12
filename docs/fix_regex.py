import os
import re

path = "/usr/local/google/home/jesuscolin/brujula/wisdom/pkg/cortex/storage.go"
with open(path, "r") as f:
    content = f.read()

# Fix Scan in search methods
content = re.sub(
    r"(&sn.ConfidenceScore,\s+)(&sn.CreatedAt)",
    r"\1&sn.ImpactScore, &linksRaw, \2",
    content
)

# Fix Unmarshal in search methods
content = re.sub(
    r"(if err := json.Unmarshal\(metadataRaw, &sn.Metadata\); err != nil \{\s+return nil, err\s+\})",
    r"\1
                if err := json.Unmarshal(linksRaw, &sn.ExternalLinks); err != nil {
                        return nil, err
                }",
    content
)

with open(path, "w") as f:
    f.write(content)
