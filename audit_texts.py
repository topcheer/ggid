#!/usr/bin/env python3
import json, re, subprocess
from pathlib import Path

files = [
    "console/src/app/settings/sod/page.tsx",
    "console/src/app/settings/sod-conflict-detection/page.tsx",
    "console/src/app/settings/sod-rules-config/page.tsx",
    "console/src/app/settings/sod-matrix/page.tsx",
    "console/src/app/settings/sod-violations/page.tsx",
    "console/src/app/settings/access-frequency/page.tsx",
    "console/src/app/settings/access-graph/page.tsx",
    "console/src/app/settings/access-optimization/page.tsx",
    "console/src/app/settings/access-request-approval-workflow/page.tsx",
    "console/src/app/settings/access-request-lifecycle-config/page.tsx",
    "console/src/app/settings/access-review-center/page.tsx",
    "console/src/app/settings/access-reviews/page.tsx",
    "console/src/app/settings/attribute-governance/page.tsx",
    "console/src/app/settings/audience-mismatches/page.tsx",
    "console/src/app/settings/budget-tracking/page.tsx",
    "console/src/app/settings/cost-centers/page.tsx",
    "console/src/app/settings/coverage-matrix/page.tsx",
    "console/src/app/settings/department-analytics/page.tsx",
    "console/src/app/settings/dynamic-roles/page.tsx",
    "console/src/app/settings/framework-coverage/page.tsx",
]

with open("console/messages/i18n_batch_backend3.json") as f:
    data = json.load(f)
values = set(data["en"].values())

def extract_nodes(content):
    # Find every span between > and <, possibly with nested {expr}. Strip whitespace.
    nodes = []
    for m in re.finditer(r'>([^\u003c]*)<', content):
        text = m.group(1).strip()
        if text and re.search(r'[A-Za-z]', text):
            nodes.append(text)
    return nodes

for f in files:
    try:
        orig = subprocess.check_output(["git", "show", f"HEAD:{f}"], text=True)
    except Exception as e:
        print(f"SKIP {f}: {e}")
        continue
    nodes = extract_nodes(orig)
    if not nodes:
        print(f"NO STRINGS: {f}")
        continue
    unmatched = [t for t in nodes if t not in values]
    print(f"{f}: {len(nodes)} nodes, {len(unmatched)} unmatched")
    for t in unmatched:
        print(f"  UNMATCHED: {t!r}")
