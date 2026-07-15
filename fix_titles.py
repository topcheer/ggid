#!/usr/bin/env python3
"""Fix title translation keys for the backend3 i18n pages."""
from pathlib import Path

BASE = Path("/Users/zhanju/ggai/ggid/console/src/app")

# Pages that should use backend3.{pageKey}.title.
TITLE_KEY_FIXES: dict[Path, tuple[str, str]] = {
    # path: (current title key, pageKey)
    BASE / "settings/sod/page.tsx": ("sod.title", "sod"),
    BASE / "settings/access-frequency/page.tsx": ("accessFrequency.title", "accessFrequency"),
    BASE / "settings/access-graph/page.tsx": ("accessGraph.title", "accessGraph"),
    BASE / "settings/access-reviews/page.tsx": ("accessReviews.title", "accessReviews"),
    BASE / "settings/attribute-governance/page.tsx": ("attributeGovernance.title", "attributeGovernance"),
    BASE / "settings/audience-mismatches/page.tsx": ("audienceMismatches.title", "audienceMismatches"),
    BASE / "settings/budget-tracking/page.tsx": ("budgetTracking.title", "budgetTracking"),
    BASE / "settings/cost-centers/page.tsx": ("costCenters.title", "costCenters"),
    BASE / "settings/coverage-matrix/page.tsx": ("coverageMatrix.title", "coverageMatrix"),
    BASE / "settings/department-analytics/page.tsx": ("departmentAnalytics.title", "departmentAnalytics"),
    BASE / "settings/framework-coverage/page.tsx": ("frameworkCoverage.title", "frameworkCoverage"),
}

# Pages marked as "no strings found": remove the title t() and restore the hardcoded string.
NO_STRINGS_TITLES: dict[Path, tuple[str, str]] = {
    # path: (current title key, hardcoded title string)
    BASE / "security/sod-matrix/page.tsx": ("securitySodMatrix.title", "Separation of Duties Matrix"),
    BASE / "settings/access-optimization/page.tsx": ("accessOptimization.title", "Access Path Optimization"),
}


def fix_title_keys(content: str) -> str:
    for path, (old_key, page_key) in TITLE_KEY_FIXES.items():
        old = f't("{old_key}")'
        new = f't("backend3.{page_key}.title")'
        if old in content:
            content = content.replace(old, new)
    return content


def restore_no_strings_titles(content: str) -> str:
    for path, (old_key, hardcoded) in NO_STRINGS_TITLES.items():
        old = f'{{t("{old_key}")}}'
        new = hardcoded
        if old in content:
            content = content.replace(old, new)
    return content


if __name__ == "__main__":
    for path in list(TITLE_KEY_FIXES.keys()) + list(NO_STRINGS_TITLES.keys()):
        text = path.read_text()
        text = fix_title_keys(text)
        text = restore_no_strings_titles(text)
        path.write_text(text)
        print(f"FIXED: {path}")
