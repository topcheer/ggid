#!/usr/bin/env node
/**
 * Apply i18n transformations to the 44 big1 console pages.
 * Uses @babel/parser to locate JSX text nodes and visible string literals.
 */
const fs = require("fs");
const path = require("path");
const parser = require("@babel/parser");
const traverse = require("@babel/traverse").default;
const t = require("@babel/types");

const BASE = "/Users/zhanju/ggai/ggid";
const PAGES_DIR = path.join(BASE, "console/src/app/settings");
const I18N_FILE = path.join(BASE, "console/messages/i18n_batch_backend_big1.json");
const EN_FLAT_FILE = path.join(BASE, "console/messages/en_flat.json");

const PAGE_NAMES = [
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
];

function toCamelCase(s) {
  s = s.trim();
  const parts = s.split(/[^a-zA-Z0-9]+/).filter(Boolean);
  if (!parts.length) return "";
  return parts[0].toLowerCase() + parts.slice(1).map(p => p.charAt(0).toUpperCase() + p.slice(1)).join("");
}

function pageNameToCamel(name) {
  return toCamelCase(name);
}

function pageNameToTitle(name) {
  return name.split("-").map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(" ");
}

function loadI18n() {
  return JSON.parse(fs.readFileSync(I18N_FILE, "utf8"));
}

function loadEnFlat() {
  return JSON.parse(fs.readFileSync(EN_FLAT_FILE, "utf8"));
}

function saveI18n(data) {
  fs.writeFileSync(I18N_FILE, JSON.stringify(data, null, 2) + "\n", "utf8");
}

function getPageKeys(data, pageCamel) {
  const prefix = `big1.${pageCamel}.`;
  return Object.fromEntries(Object.entries(data.en).filter(([k]) => k.startsWith(prefix)));
}

function valueToKey(pageKeys) {
  const map = {};
  for (const [k, v] of Object.entries(pageKeys)) {
    map[v] = k;
  }
  return map;
}

function getField(key) {
  return key.split(".")[2];
}

function addKey(data, pageCamel, field, enValue) {
  const key = `big1.${pageCamel}.${field}`;
  if (!(key in data.en)) data.en[key] = enValue;
  if (!(key in data.zh)) data.zh[key] = enValue;
  return key;
}

function findKeyForValue(data, pageCamel, s) {
  const v2k = valueToKey(getPageKeys(data, pageCamel));
  if (s in v2k) return getField(v2k[s]);
  return null;
}

function generateKeyForString(s, title) {
  if (title && s.trim() === title.trim()) return "title";
  if (s.trim() === "—") return null;
  return toCamelCase(s);
}

function isTranslatable(s) {
  return /[a-zA-Z]/.test(s) && s.trim() !== "";
}

function addImportAndHook(source) {
  if (!source.includes("useTranslations")) {
    let insertPos = -1;
    for (const m of source.matchAll(/from\s+["'][^"']+["'];\n/g)) {
      insertPos = m.index + m[0].length;
    }
    if (insertPos === -1) {
      insertPos = source.indexOf("\n") + 1;
      source = source.slice(0, insertPos) + '\nimport { useTranslations } from "@/lib/i18n";\n' + source.slice(insertPos);
    } else {
      source = source.slice(0, insertPos) + 'import { useTranslations } from "@/lib/i18n";\n' + source.slice(insertPos);
    }
  }
  if (!source.includes("const t = useTranslations()")) {
    source = source.replace(/(export default function \w+\s*\([^)]*\)\s*\{)/, "$1\n  const t = useTranslations();");
  }
  return source;
}

function extractTitleFromH1(source) {
  // Try to find the literal text of an h1 element; ignore if h1 is already a JSX expression.
  const match = source.match(/\u003ch1[^\u003e]*\u003e([\s\S]*?)\u003c\/h1\u003e/);
  if (!match) return null;
  const inner = match[1].trim();
  // If the h1 is already wrapped in {t(...)} or contains other JSX, return null
  if (/\{t\(/.test(inner)) return null;
  // Strip simple JSX tags and expressions to get plain text
  const plain = inner.replace(/\{[^}]*\}/g, "").replace(/\u003c[^\u003e]+\u003e/g, "").trim();
  return plain || null;
}

function processPage(data, enFlat, pageName) {
  const pageCamel = pageNameToCamel(pageName);
  const filePath = path.join(PAGES_DIR, pageName, "page.tsx");
  if (!fs.existsSync(filePath)) {
    console.log(`SKIP: ${filePath} does not exist`);
    return;
  }
  let source = fs.readFileSync(filePath, "utf8");

  // Normalize legacy t("pageCamel.xxx") and t("backend3.pageCamel.xxx") to big1.
  source = source.replace(new RegExp(`t\\("backend3\\.${pageCamel}\\.`, "g"), `t("big1.${pageCamel}.`);
  source = source.replace(new RegExp(`t\\("${pageCamel}\\.`, "g"), `t("big1.${pageCamel}.`);

  // Title for the "title" key heuristic
  let title = extractTitleFromH1(source);
  // Fallback to the flat dictionary title value
  if (!title) {
    title = enFlat[`${pageCamel}.title`] || pageNameToTitle(pageName);
  }

  // Ensure title key is present in the batch JSON
  if (title) {
    addKey(data, pageCamel, "title", title);
  }

  const ast = parser.parse(source, {
    sourceType: "module",
    plugins: ["jsx", "typescript"],
  });

  const replacements = [];

  traverse(ast, {
    JSXText(path) {
      const node = path.node;
      const text = node.value;
      if (!isTranslatable(text)) return;
      const field = findKeyForValue(data, pageCamel, text.trim()) || generateKeyForString(text, title);
      if (!field) return;
      addKey(data, pageCamel, field, text.trim());
      replacements.push({
        start: node.start,
        end: node.end,
        replacement: `{t("big1.${pageCamel}.${field}")}`,
      });
    },
    StringLiteral(path) {
      const node = path.node;
      const s = node.value;
      if (!isTranslatable(s)) return;

      // Only translate string literals that are direct children of JSX element children
      // and not inside attributes, call expressions, object literals, etc.
      let inJsxElementChild = false;
      let insideBadParent = false;
      let p = path;
      while (p) {
        const parent = p.parentPath;
        if (!parent) break;
        const pt = parent.node;
        if (t.isJSXElement(pt) || t.isJSXFragment(pt)) {
          if (parent.node.children.includes(p.node)) {
            inJsxElementChild = true;
          }
        }
        if (t.isJSXAttribute(pt)) {
          insideBadParent = true;
          break;
        }
        if (t.isCallExpression(pt) || t.isObjectExpression(pt) || t.isArrayExpression(pt) || t.isObjectProperty(pt)) {
          insideBadParent = true;
          break;
        }
        p = parent;
      }
      if (insideBadParent || !inJsxElementChild) return;

      const field = findKeyForValue(data, pageCamel, s) || generateKeyForString(s, title);
      if (!field) return;
      addKey(data, pageCamel, field, s);
      replacements.push({
        start: node.start,
        end: node.end,
        replacement: `t("big1.${pageCamel}.${field}")`,
      });
    },
  });

  replacements.sort((a, b) => b.start - a.start);
  for (const { start, end, replacement } of replacements) {
    source = source.slice(0, start) + replacement + source.slice(end);
  }

  source = addImportAndHook(source);
  fs.writeFileSync(filePath, source, "utf8");
  console.log(`DONE: ${filePath}`);
}

function main() {
  const data = loadI18n();
  const enFlat = loadEnFlat();

  // Pre-populate the batch JSON with big1.* values from the flat dictionary for the target pages only.
  const pageCamels = new Set(PAGE_NAMES.map(pageNameToCamel));
  for (const [key, value] of Object.entries(enFlat)) {
    if (!key.startsWith("big1.")) continue;
    const parts = key.split(".");
    if (parts.length !== 3) continue;
    if (!pageCamels.has(parts[1])) continue;
    addKey(data, parts[1], parts[2], value);
  }

  for (const pageName of PAGE_NAMES) {
    processPage(data, enFlat, pageName);
  }
  saveI18n(data);
}

main();
