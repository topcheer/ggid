#!/usr/bin/env python3
"""Apply i18n transformations to the 44 big1 console pages."""
import re
import json
from pathlib import Path

BASE = Path("/Users/zhanju/ggai/ggid")
PAGES_DIR = BASE / "console/src/app/settings"
I18N_FILE = BASE / "console/messages/i18n_batch_backend_big1.json"

PAGE_NAMES = [
    "device-attestation", "device-authorization-flow-config", "device-fingerprint-analytics",
    "device-posture", "device-registry", "did-resolver", "dynamic-client-registration",
    "dynamic-roles", "email-template-config", "export-schedule", "feature-flag-architecture-config",
    "federation-patterns-config", "geo-fencing", "geo-fencing-config", "geo-velocity-rules",
    "grant-flows", "grant-history", "grant-type-stats", "group-permission-tree", "hash-chain",
    "hibp-breach-check", "hijack-timeline", "identity-proofing-config", "identity-recovery-config",
    "idp-config", "idp-discovery-config", "idp-failover-config", "idp-federation", "idp-metadata-import",
    "idp-metadata-import-config", "impersonation-config", "impersonation-log", "impersonation-session",
    "impossible-travel", "inactive-cleanup", "introspection-cache-config", "introspection-stats",
    "ip-reputation", "ip-reputation-config", "itdr-dashboard", "joiner-flow", "joiner-flow-dashboard",
    "jwt-claim-validation-config", "jwt-expiry-config",
]


def to_camel_case(s: str) -> str:
    s = s.strip()
    parts = re.split(r"[^a-zA-Z0-9]+", s)
    parts = [p for p in parts if p]
    if not parts:
        return ""
    return parts[0].lower() + "".join(p.capitalize() for p in parts[1:])


def page_name_to_camel(name: str) -> str:
    return to_camel_case(name)


def load_i18n():
    with open(I18N_FILE) as f:
        return json.load(f)


def save_i18n(data):
    with open(I18N_FILE, "w") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)


def get_page_keys(data, page_camel: str):
    prefix = f"big1.{page_camel}."
    return {k: v for k, v in data["en"].items() if k.startswith(prefix)}


def value_to_key(page_keys):
    return {v: k for k, v in page_keys.items()}


def get_field(key: str) -> str:
    return key.split(".", 2)[2]


def add_key(data, page_camel: str, field: str, en_value: str):
    key = f"big1.{page_camel}.{field}"
    if key not in data["en"]:
        data["en"][key] = en_value
    if key not in data["zh"]:
        data["zh"][key] = en_value
    return key


def find_key_for_value(data, page_camel: str, s: str):
    page_keys = get_page_keys(data, page_camel)
    v2k = value_to_key(page_keys)
    if s in v2k:
        return get_field(v2k[s])
    return None


def generate_key_for_string(s: str, title: str | None) -> str | None:
    if title and s.strip() == title.strip():
        return "title"
    if s.strip() == "—":
        return None
    return to_camel_case(s)


def find_quoted_strings(s: str):
    return list(re.finditer(r'"(?:[^"\\]|\\.)*"|\'(?:[^\'\\]|\\.)*\'', s))


def is_translatable(s: str) -> bool:
    return bool(re.search(r"[a-zA-Z]", s)) and s.strip()


def extract_jsx_expressions(segment: str):
    """Split segment into top-level JSX expressions and plain text."""
    result = []
    i = 0
    depth = 0
    start = None
    while i < len(segment):
        ch = segment[i]
        if ch == "{":
            if depth == 0:
                start = i
                if result and result[-1][0] == "text" and not result[-1][1]:
                    result.pop()
            depth += 1
        elif ch == "}":
            if depth > 0:
                depth -= 1
                if depth == 0 and start is not None:
                    result.append(("expr", segment[start : i + 1]))
                    start = None
        else:
            if depth == 0:
                if not result or result[-1][0] != "text":
                    result.append(("text", ""))
                result[-1] = ("text", result[-1][1] + ch)
        i += 1
    if start is not None:
        if not result or result[-1][0] != "text":
            result.append(("text", ""))
        result[-1] = ("text", result[-1][1] + segment[start:])
    return result


def translate_plain_segment(seg: str, data, page_camel: str, title: str | None) -> str:
    """Translate a plain text segment using phrase matching."""
    page_keys = get_page_keys(data, page_camel)
    v2k = value_to_key(page_keys)
    values = sorted(v2k.keys(), key=len, reverse=True)

    result = []
    i = 0
    n = len(seg)
    while i < n:
        # Pass through non-word characters
        if not re.match(r"[a-zA-Z]", seg[i]):
            result.append(seg[i])
            i += 1
            continue

        # Try to match a known value starting here
        matched = False
        for v in values:
            if seg.startswith(v, i):
                next_i = i + len(v)
                # Ensure whole phrase (not a substring of a longer word)
                if next_i < n and re.match(r"[a-zA-Z]", seg[next_i]):
                    continue
                field = get_field(v2k[v])
                add_key(data, page_camel, field, v)
                result.append(f'{{t("big1.{page_camel}.{field}")}}')
                i = next_i
                matched = True
                break
        if matched:
            continue

        # Unknown phrase: consume letters, spaces, hyphens, digits, apostrophes, and parentheses
        j = i
        while j < n and re.match(r"[a-zA-Z0-9 \-'()/]", seg[j]):
            j += 1
        phrase = seg[i:j].rstrip()
        if phrase and is_translatable(phrase):
            field = generate_key_for_string(phrase, title)
            if field:
                add_key(data, page_camel, field, phrase)
                result.append(f'{{t("big1.{page_camel}.{field}")}}')
            else:
                result.append(phrase)
        elif phrase:
            result.append(phrase)
        i = j

    return "".join(result)


def translate_expression(expr: str, data, page_camel: str, title: str | None) -> str:
    """Replace quoted strings inside a JSX expression with t() calls."""
    new_expr = expr
    for qm in reversed(find_quoted_strings(expr)):
        qs, qe = qm.start(), qm.end()
        q = qm.group(0)
        s = q[1:-1]
        if not is_translatable(s):
            continue
        field = find_key_for_value(data, page_camel, s)
        if field is None:
            field = generate_key_for_string(s, title)
        if field is None:
            continue
        add_key(data, page_camel, field, s)
        new_expr = new_expr[:qs] + f't("big1.{page_camel}.{field}")' + new_expr[qe:]
    return new_expr


def process_text_node_content(content: str, data, page_camel: str, title: str | None) -> str:
    segments = extract_jsx_expressions(content)
    new_segments = []
    for seg_type, seg in segments:
        if seg_type == "expr":
            new_segments.append(translate_expression(seg, data, page_camel, title))
        else:
            new_segments.append(translate_plain_segment(seg, data, page_camel, title))
    return "".join(new_segments)


def add_import_and_hook(content: str) -> str:
    if 'useTranslations' not in content:
        last_import_end = -1
        for m in re.finditer(r'from\s+["\'][^"\']+["\'];', content):
            last_import_end = m.end()
        if last_import_end == -1:
            insert_pos = content.find("\n") + 1
            content = content[:insert_pos] + '\nimport { useTranslations } from "@/lib/i18n";\n' + content[insert_pos:]
        else:
            insert_pos = last_import_end
            if content[insert_pos : insert_pos + 1] == "\n":
                insert_pos += 1
            content = (
                content[:insert_pos]
                + 'import { useTranslations } from "@/lib/i18n";\n'
                + content[insert_pos:]
            )
    if "const t = useTranslations()" not in content:
        content = re.sub(
            r"(export default function \w+\s*\([^)]*\)\s*\{)",
            r"\1\n  const t = useTranslations();",
            content,
            count=1,
        )
    return content


def process_page(data, page_name: str):
    page_camel = page_name_to_camel(page_name)
    path = PAGES_DIR / page_name / "page.tsx"
    if not path.exists():
        print(f"SKIP: {path} does not exist")
        return
    content = path.read_text()

    title_match = re.search(r"<h1[^>]*>([^<]*?)<", content)
    title = title_match.group(1).strip() if title_match else None

    content = add_import_and_hook(content)

    nodes = list(re.finditer(r">([^<]*?)<", content))
    for m in reversed(nodes):
        start, end = m.start(), m.end()
        between = m.group(1)
        if not between.strip():
            continue
        new_between = process_text_node_content(between, data, page_camel, title)
        if new_between != between:
            content = content[:start] + ">" + new_between + "<" + content[end:]

    path.write_text(content)
    print(f"DONE: {path}")


def main():
    data = load_i18n()
    for page_name in PAGE_NAMES:
        process_page(data, page_name)
    save_i18n(data)


if __name__ == "__main__":
    main()
