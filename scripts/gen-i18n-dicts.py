#!/usr/bin/env python3
"""Generate console/src/lib/i18n-dicts.ts from messages/*_flat.json files.

Usage: python3 scripts/gen-i18n-dicts.py
"""
import json
import os
import glob

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MSG_DIR = os.path.join(ROOT, "console", "messages")
OUT_PATH = os.path.join(ROOT, "console", "src", "lib", "i18n-dicts.ts")

# Discover all language files: *_flat.json
lang_files = {}
for path in sorted(glob.glob(os.path.join(MSG_DIR, "*_flat.json"))):
    basename = os.path.basename(path)
    lang_code = basename.replace("_flat.json", "")
    lang_files[lang_code] = path

# TS-safe identifier: replace hyphen with underscore
def ts_id(lang):
    return lang.replace("-", "_")

with open(OUT_PATH, "w") as f:
    f.write("// Auto-generated from messages/*_flat.json — DO NOT EDIT\n")
    f.write("// To add translations, edit messages/*.json and run: python3 scripts/gen-i18n-dicts.py\n")
    f.write(f"// Languages: {', '.join(sorted(lang_files.keys()))}\n\n")

    for lang, path in sorted(lang_files.items()):
        data = json.load(open(path))
        var_name = ts_id(lang)
        f.write(f"export const {var_name}: Record<string, string> = {{\n")
        for k in sorted(data.keys()):
            v = str(data[k]).replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n')
            f.write(f'  "{k}": "{v}",\n')
        f.write("};\n\n")

    # Export locale registry
    f.write("export const locales: Record<string, Record<string, string>> = {\n")
    for lang in sorted(lang_files.keys()):
        f.write(f'  "{lang}": {ts_id(lang)},\n')
    f.write("};\n")

total_keys = max(len(json.load(open(p))) for p in lang_files.values())
print(f"Generated {OUT_PATH}: {len(lang_files)} languages, {total_keys} keys each")
