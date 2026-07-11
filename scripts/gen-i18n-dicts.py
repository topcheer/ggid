#!/usr/bin/env python3
"""Generate console/src/lib/i18n-dicts.ts from messages/en_flat.json + zh_flat.json.

Usage: python3 scripts/gen-i18n-dicts.py
"""
import json
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

en_path = os.path.join(ROOT, "console", "messages", "en_flat.json")
zh_path = os.path.join(ROOT, "console", "messages", "zh_flat.json")
out_path = os.path.join(ROOT, "console", "src", "lib", "i18n-dicts.ts")

en = json.load(open(en_path))
zh = json.load(open(zh_path))

with open(out_path, "w") as f:
    f.write("// Auto-generated from messages/en_flat.json + zh_flat.json — DO NOT EDIT\n")
    f.write("// To add translations, edit messages/*.json and run: python3 scripts/gen-i18n-dicts.py\n\n")
    
    f.write("export const en: Record<string, string> = {\n")
    for k in sorted(en.keys()):
        v = str(en[k]).replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n')
        f.write(f'  "{k}": "{v}",\n')
    f.write("};\n\n")
    
    f.write("export const zh: Record<string, string> = {\n")
    for k in sorted(zh.keys()):
        v = str(zh[k]).replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n')
        f.write(f'  "{k}": "{v}",\n')
    f.write("};\n")

print(f"Generated {out_path}: {len(en)} EN keys, {len(zh)} ZH keys")
