#!/usr/bin/env python3
"""Apply i18n transformations to the 20 backend3 console pages."""
import re
from pathlib import Path

BASE = Path("/Users/zhanju/ggai/ggid/console/src/app")

# Title regex replacements per file: (title_text, key)
TITLES: dict[Path, list[tuple[str, str]]] = {
    BASE / "settings/sod/page.tsx": [("Separation of Duties", "backend3.sod.title")],
    BASE / "settings/sod-conflict-detection/page.tsx": [("SoD Conflict Detection", "backend3.sodConflictDetection.title")],
    BASE / "settings/sod-rules-config/page.tsx": [("SoD Rules Configuration", "backend3.sodRulesConfig.title")],
    BASE / "settings/access-frequency/page.tsx": [("Access Frequency", "backend3.accessFrequency.title")],
    BASE / "settings/access-graph/page.tsx": [("Access Graph", "backend3.accessGraph.title")],
    BASE / "settings/access-request-approval-workflow/page.tsx": [("Access Request Approval Workflow", "backend3.accessRequestApproval.title")],
    BASE / "settings/access-request-lifecycle-config/page.tsx": [("Access Request Lifecycle Configuration", "backend3.accessRequestLifecycle.title")],
    BASE / "settings/access-review-center/page.tsx": [("Access Review Center", "backend3.accessReviewCenter.title")],
    BASE / "settings/access-reviews/page.tsx": [("Access Reviews", "backend3.accessReviews.title")],
    BASE / "settings/attribute-governance/page.tsx": [("Attribute Governance", "backend3.attributeGovernance.title")],
    BASE / "settings/audience-mismatches/page.tsx": [("Audience Mismatches", "backend3.audienceMismatches.title")],
    BASE / "settings/budget-tracking/page.tsx": [("Budget Tracking", "backend3.budgetTracking.title")],
    BASE / "settings/cost-centers/page.tsx": [("Cost Centers", "backend3.costCenters.title")],
    BASE / "settings/coverage-matrix/page.tsx": [("Coverage Matrix", "backend3.coverageMatrix.title")],
    BASE / "settings/department-analytics/page.tsx": [("Department Analytics", "backend3.departmentAnalytics.title")],
    BASE / "settings/dynamic-roles/page.tsx": [("Dynamic Roles", "backend3.dynamicRoles.title")],
    BASE / "settings/framework-coverage/page.tsx": [("Framework Coverage", "backend3.frameworkCoverage.title")],
}

# Exact string replacements for non-title visible text.
TRANSFORMATIONS: dict[Path, list[tuple[str, str]]] = {
    BASE / "settings/sod/page.tsx": [
        (">Active Rules<", '>{t("backend3.sod.activeRules")}<'),
        (">Total Violations<", '>{t("backend3.sod.totalViolations")}<'),
        (">Critical<", '>{t("backend3.sod.critical")}<'),
        (">Clean Users<", '>{t("backend3.sod.cleanUsers")}<'),
        (">Add SoD Rule<", '>{t("backend3.sod.addRule")}<'),
        (">Description<", '>{t("backend3.sod.description")}<'),
        (">Severity<", '>{t("backend3.sod.severity")}<'),
        (">Cancel<", '>{t("backend3.sod.cancel")}<'),
        (">Delete<", '>{t("backend3.sod.delete")}<'),
    ],
    BASE / "settings/sod-conflict-detection/page.tsx": [
        ("'Add Rule'", 't("backend3.sodConflictDetection.addRule")'),
        (">Separation of Duties Conflict Detection<", '>{t("backend3.sodConflictDetection.title")}<'),
        (">Open Violations<", '>{t("backend3.sodConflictDetection.openViolations")}<'),
        (">Resolved<", '>{t("backend3.sodConflictDetection.resolved")}<'),
        (">Sensitivity Level<", '>{t("backend3.sodConflictDetection.sensitivityLevel")}<'),
        (">Automatically remove conflicting roles<", '>{t("backend3.sodConflictDetection.autoRemove")}<'),
        (">Rule<", '>{t("backend3.sodConflictDetection.rule")}<'),
        (">Role A<", '>{t("backend3.sodConflictDetection.roleA")}<'),
        (">Role B<", '>{t("backend3.sodConflictDetection.roleB")}<'),
        (">Conflict Level<", '>{t("backend3.sodConflictDetection.conflictLevel")}<'),
        (">Add Rule<", '>{t("backend3.sodConflictDetection.addRule")}<'),
        (">Conflict Matrix Heatmap<", '>{t("backend3.sodConflictDetection.conflictMatrixHeatmap")}<'),
        (">Critical<", '>{t("backend3.sodConflictDetection.critical")}<'),
        (">High<", '>{t("backend3.sodConflictDetection.high")}<'),
        (">Medium<", '>{t("backend3.sodConflictDetection.medium")}<'),
        (">Date<", '>{t("backend3.sodConflictDetection.date")}<'),
    ],
    BASE / "settings/sod-rules-config/page.tsx": [
        (">Sensitivity Level<", '>{t("backend3.sodRulesConfig.sensitivityLevel")}<'),
        (">Add<", '>{t("backend3.sodRulesConfig.add")}<'),
        (">Role A<", '>{t("backend3.sodRulesConfig.roleA")}<'),
        (">Role B<", '>{t("backend3.sodRulesConfig.roleB")}<'),
        (">Conflict Level<", '>{t("backend3.sodRulesConfig.conflictLevel")}<'),
        (">Enabled<", '>{t("backend3.sodRulesConfig.enabled")}<'),
        (">Conflict Matrix Heatmap<", '>{t("backend3.sodRulesConfig.conflictMatrixHeatmap")}<'),
        (">Violation History<", '>{t("backend3.sodRulesConfig.violationHistory")}<'),
        (">User<", '>{t("backend3.sodRulesConfig.user")}<'),
        (">Rule<", '>{t("backend3.sodRulesConfig.rule")}<'),
        (">Date<", '>{t("backend3.sodRulesConfig.date")}<'),
        (">Status<", '>{t("backend3.sodRulesConfig.status")}<'),
    ],
    BASE / "settings/access-frequency/page.tsx": [
        (">Total Accesses<", '>{t("backend3.accessFrequency.totalAccesses")}<'),
        (">Anomalies<", '>{t("backend3.accessFrequency.anomalies")}<'),
        (">Peak Hour<", '>{t("backend3.accessFrequency.peakHour")}<'),
        (">Peak Count<", '>{t("backend3.accessFrequency.peakCount")}<'),
    ],
    BASE / "settings/access-graph/page.tsx": [
        (">Analyze<", '>{t("backend3.accessGraph.analyze")}<'),
        (">Effective Permissions Summary<", '>{t("backend3.accessGraph.effectivePermissionsSummary")}<'),
        (">None<", '>{t("backend3.accessGraph.none")}<'),
    ],
    BASE / "settings/access-request-approval-workflow/page.tsx": [
        (">Manage pending access requests and approval chains<", '>{t("backend3.accessRequestApproval.subtitle")}<'),
        (">SLA Remaining<", '>{t("backend3.accessRequestApproval.slaRemaining")}<'),
    ],
    BASE / "settings/access-request-lifecycle-config/page.tsx": [
        (">Lifecycle Stages<", '>{t("backend3.accessRequestLifecycle.lifecycleStages")}<'),
        (">Global Limits<", '>{t("backend3.accessRequestLifecycle.globalLimits")}<'),
        (">Max Duration (days)<", '>{t("backend3.accessRequestLifecycle.maxDuration")}<'),
        (">Max Duration<", '>{t("backend3.accessRequestLifecycle.maxDuration")}<'),
        (">Condition<", '>{t("backend3.accessRequestLifecycle.condition")}<'),
        (">Target Role<", '>{t("backend3.accessRequestLifecycle.targetRole")}<'),
        (">No data<", '>{t("backend3.accessRequestLifecycle.noData")}<'),
    ],
    BASE / "settings/access-review-center/page.tsx": [
        ("'Create Review'", 't("backend3.accessReviewCenter.createReview")'),
        (">Create Access Review<", '>{t("backend3.accessReviewCenter.createAccessReview")}<'),
        (">Create Review<", '>{t("backend3.accessReviewCenter.createReview")}<'),
        (">Pending<", '>{t("backend3.accessReviewCenter.pending")}<'),
        (">Overdue<", '>{t("backend3.accessReviewCenter.overdue")}<'),
        (">Monthly<", '>{t("backend3.accessReviewCenter.monthly")}<'),
        (">Quarterly<", '>{t("backend3.accessReviewCenter.quarterly")}<'),
        (">Annual<", '>{t("backend3.accessReviewCenter.annual")}<'),
        (">Bulk Approve<", '>{t("backend3.accessReviewCenter.bulkApprove")}<'),
        (">Clear<", '>{t("backend3.accessReviewCenter.clear")}<'),
        (">Decision<", '>{t("backend3.accessReviewCenter.decision")}<'),
        (">Date<", '>{t("backend3.accessReviewCenter.date")}<'),
        (">Actions<", '>{t("backend3.accessReviewCenter.actions")}<'),
        (">Approve<", '>{t("backend3.accessReviewCenter.approve")}<'),
        (">Reject<", '>{t("backend3.accessReviewCenter.reject")}<'),
    ],
    BASE / "settings/access-reviews/page.tsx": [
        ("> New Campaign<", '>{t("backend3.accessReviews.newCampaign")}<'),
        (">New Campaign<", '>{t("backend3.accessReviews.newCampaign")}<'),
        (">Name<", '>{t("backend3.accessReviews.name")}<'),
        (">Reviewer: ", '>{t("backend3.accessReviews.reviewer")}: '),
        (">Deadline: ", '>{t("backend3.accessReviews.deadline")}: '),
        (">Reviewer<", '>{t("backend3.accessReviews.reviewer")}<'),
        (">Deadline<", '>{t("backend3.accessReviews.deadline")}<'),
        (">Cancel<", '>{t("backend3.accessReviews.cancel")}<'),
        (">Delete<", '>{t("backend3.accessReviews.delete")}<'),
    ],
    BASE / "settings/attribute-governance/page.tsx": [
        (">Total Attributes<", '>{t("backend3.attributeGovernance.totalAttributes")}<'),
        (">Masked<", '>{t("backend3.attributeGovernance.masked")}<'),
        (">Attribute<", '>{t("backend3.attributeGovernance.attribute")}<'),
        (">PII Class<", '>{t("backend3.attributeGovernance.piiClass")}<'),
        (">Mask Rule<", '>{t("backend3.attributeGovernance.maskRule")}<'),
        (">Access Freq<", '>{t("backend3.attributeGovernance.accessFreq")}<'),
        (">Last Accessed By<", '>{t("backend3.attributeGovernance.lastAccessedBy")}<'),
        (">Retention<", '>{t("backend3.attributeGovernance.retention")}<'),
    ],
    BASE / "settings/audience-mismatches/page.tsx": [
        (">Total Mismatches<", '>{t("backend3.audienceMismatches.totalMismatches")}<'),
        (">Blocked<", '>{t("backend3.audienceMismatches.blocked")}<'),
        (">All<", '>{t("backend3.audienceMismatches.all")}<'),
        (">Blocked Only<", '>{t("backend3.audienceMismatches.blockedOnly")}<'),
        (">Allowed Only<", '>{t("backend3.audienceMismatches.allowedOnly")}<'),
        (">Token<", '>{t("backend3.audienceMismatches.token")}<'),
        (">Expected<", '>{t("backend3.audienceMismatches.expected")}<'),
        (">Actual<", '>{t("backend3.audienceMismatches.actual")}<'),
        (">Resource<", '>{t("backend3.audienceMismatches.resource")}<'),
        (">Status<", '>{t("backend3.audienceMismatches.status")}<'),
        (">Timestamp<", '>{t("backend3.audienceMismatches.timestamp")}<'),
    ],
    BASE / "settings/budget-tracking/page.tsx": [
        (">Total Spent<", '>{t("backend3.budgetTracking.totalSpent")}<'),
        (">Total Budget<", '>{t("backend3.budgetTracking.totalBudget")}<'),
        (">Remaining<", '>{t("backend3.budgetTracking.remaining")}<'),
        (">Burn Rate<", '>{t("backend3.budgetTracking.burnRate")}<'),
        (">Projected EOY<", '>{t("backend3.budgetTracking.projectedEOY")}<'),
        (">Users<", '>{t("backend3.budgetTracking.users")}<'),
    ],
    BASE / "settings/cost-centers/page.tsx": [
        (">Departments<", '>{t("backend3.costCenters.departments")}<'),
        (">Total Budget<", '>{t("backend3.costCenters.totalBudget")}<'),
        (">Used<", '>{t("backend3.costCenters.used")}<'),
        (">Allocation<", '>{t("backend3.costCenters.allocation")}<'),
        (">Members<", '>{t("backend3.costCenters.members")}<'),
        (">Budget<", '>{t("backend3.costCenters.budget")}<'),
        (">Resource Usage<", '>{t("backend3.costCenters.resourceUsage")}<'),
    ],
    BASE / "settings/coverage-matrix/page.tsx": [
        (">Subjects<", '>{t("backend3.coverageMatrix.subjects")}<'),
        (">Resources<", '>{t("backend3.coverageMatrix.resources")}<'),
        (">Gaps<", '>{t("backend3.coverageMatrix.gaps")}<'),
        (">Subject<", '>{t("backend3.coverageMatrix.subject")}<'),
    ],
    BASE / "settings/department-analytics/page.tsx": [
        (">Departments<", '>{t("backend3.departmentAnalytics.departments")}<'),
        (">Total Headcount<", '>{t("backend3.departmentAnalytics.totalHeadcount")}<'),
        (">Open Positions<", '>{t("backend3.departmentAnalytics.openPositions")}<'),
        (">Avg Tenure<", '>{t("backend3.departmentAnalytics.avgTenure")}<'),
        (">Budget Util<", '>{t("backend3.departmentAnalytics.budgetUtil")}<'),
        (">Attrition<", '>{t("backend3.departmentAnalytics.attrition")}<'),
    ],
    BASE / "settings/dynamic-roles/page.tsx": [
        (">Test Dynamic Role Assignment<", '>{t("backend3.dynamicRoles.testAssignment")}<'),
        (">No conditions<", '>{t("backend3.dynamicRoles.noConditions")}<'),
        (">Name<", '>{t("backend3.dynamicRoles.name")}<'),
        (">Description<", '>{t("backend3.dynamicRoles.description")}<'),
        (">Conditions<", '>{t("backend3.dynamicRoles.conditions")}<'),
    ],
    BASE / "settings/framework-coverage/page.tsx": [
        (">Total Controls<", '>{t("backend3.frameworkCoverage.totalControls")}<'),
        (">Covered<", '>{t("backend3.frameworkCoverage.covered")}<'),
        (">Gaps<", '>{t("backend3.frameworkCoverage.gaps")}<'),
    ],
    # Pages with no translatable keys: only import + const t will be added.
    BASE / "settings/access-optimization/page.tsx": [],
    BASE / "security/sod-matrix/page.tsx": [],
    BASE / "security/sod-violations/page.tsx": [],
}


def add_import_and_hook(content: str) -> str:
    # Add useTranslations import if missing. Insert after the last import statement,
    # using DOTALL so multiline imports are handled correctly.
    if 'useTranslations' not in content:
        matches = list(re.finditer(r'import\s+.*?from\s+["\'][^"\']+["\'];', content, re.DOTALL))
        if matches:
            insert_pos = matches[-1].end()
            if content[insert_pos:insert_pos + 1] == "\n":
                insert_pos += 1
            content = (
                content[:insert_pos]
                + 'import { useTranslations } from "@/lib/i18n";\n'
                + content[insert_pos:]
            )

    # Add const t = useTranslations() at the start of the component function body.
    if 'const t = useTranslations()' not in content:
        content = re.sub(
            r'(export\s+default\s+function\s+\w+\s*\([^)]*\)\s*\{)',
            r'\1\n  const t = useTranslations();',
            content,
            count=1,
        )
    return content


def replace_title(content: str, title: str, key: str) -> str:
    pattern = re.compile(r'(>)\s*' + re.escape(title) + r'\s*(<)', re.DOTALL)
    replacement = r'> {t("' + key + '")}<'
    return pattern.sub(replacement, content)


if __name__ == "__main__":
    for path, replacements in TRANSFORMATIONS.items():
        if not path.exists():
            print(f"SKIP: {path} does not exist")
            continue
        text = path.read_text()
        text = add_import_and_hook(text)
        # Replace titles first.
        for title, key in TITLES.get(path, []):
            text = replace_title(text, title, key)
        # Replace remaining exact strings.
        for old, new in replacements:
            text = text.replace(old, new)
        path.write_text(text)
        print(f"DONE: {path}")
