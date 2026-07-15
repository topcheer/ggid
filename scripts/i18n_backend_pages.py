import re
from pathlib import Path

ROOT = Path('/Users/zhanju/ggai/ggid/console/src/app/settings')

PAGES = [
    {
        'file': 'token-management/page.tsx',
        'replacements': [
            ('<div><h1 className="text-2xl font-bold">Token Management</h1>', '<div><h1 className="text-2xl font-bold">{t("backend.tokenManagement.title")}</h1>'),
            ('<th className="p-3">Type</th>', '<th className="p-3">{t("backend.tokenManagement.type")}</th>'),
            ('<th className="p-3">User</th>', '<th className="p-3">{t("backend.tokenManagement.user")}</th>'),
            ('<th className="p-3">Client</th>', '<th className="p-3">{t("backend.tokenManagement.client")}</th>'),
            ('<th className="p-3">Issued</th>', '<th className="p-3">{t("backend.tokenManagement.issued")}</th>'),
            ('<th className="p-3">Expires</th>', '<th className="p-3">{t("backend.tokenManagement.expires")}</th>'),
            ('<th className="p-3">Scopes</th>', '<th className="p-3">{t("backend.tokenManagement.scopes")}</th>'),
            ('<th className="p-3">DPoP</th>', '<th className="p-3">{t("backend.tokenManagement.dpop")}</th>'),
            ('<th className="p-3">Action</th>', '<th className="p-3">{t("backend.tokenManagement.action")}</th>'),
            ('>Revoke</button>', '>{t("backend.tokenManagement.revoke")}</button>'),
        ],
    },
    {
        'file': 'token-lifecycle-dashboard/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> Token Lifecycle</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> {t("backend.tokenLifecycle.title")}</h1>'),
            ('<h3 className="text-sm font-semibold mb-3">Tokens by Stage</h3>', '<h3 className="text-sm font-semibold mb-3">{t("backend.tokenLifecycle.tokensByStage")}</h3>'),
            ('<h3 className="text-sm font-semibold mb-3">Metrics</h3>', '<h3 className="text-sm font-semibold mb-3">{t("backend.tokenLifecycle.metrics")}</h3>'),
            ('<span className="text-xs text-gray-500">Avg Lifetime</span>', '<span className="text-xs text-gray-500">{t("backend.tokenLifecycle.avgLifetime")}</span>'),
            ('<span className="text-xs text-gray-500">Refresh Rate</span>', '<span className="text-xs text-gray-500">{t("backend.tokenLifecycle.refreshRate")}</span>'),
            ('<span className="text-xs text-gray-500">Issuance Rate</span>', '<span className="text-xs text-gray-500">{t("backend.tokenLifecycle.issuanceRate")}</span>'),
            ('<span className="text-xs text-gray-500">Revocation Rate</span>', '<span className="text-xs text-gray-500">{t("backend.tokenLifecycle.revocationRate")}</span>'),
            ('<th className="px-4 py-3 text-left font-medium">Active</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.active")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Expiring</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.expiring")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Revoked</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.revoked")}</th>'),
        ],
    },
    {
        'file': 'token-rotation/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><RefreshCw className="w-6 h-6 text-teal-500" /> Token Rotation</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><RefreshCw className="w-6 h-6 text-teal-500" /> {t("backend.tokenRotation.title")}</h1>'),
            ('<option value="">Select Client</option>', '<option value="">{t("backend.tokenRotation.selectClient")}</option>'),
            ('>Retry</button>', '>{t("backend.tokenRotation.retry")}</button>'),
        ],
    },
    {
        'file': 'token-family/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> Token Family</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> {t("backend.tokenFamily.title")}</h1>'),
            ('>REUSE</span>', '>{t("backend.tokenFamily.reuse")}</span>'),
            ('<h4 className="text-xs font-semibold text-gray-500 mb-2">Child Tokens</h4>', '<h4 className="text-xs font-semibold text-gray-500 mb-2">{t("backend.tokenFamily.childTokens")}</h4>'),
        ],
    },
    {
        'file': 'token-binding-config/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">Token Binding Configuration</h1>', '<h1 className="text-2xl font-bold">{t("backend.tokenBindingConfig.title")}</h1>'),
            ('<h2 className="text-lg font-semibold">Binding Settings</h2>', '<h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.bindingSettings")}</h2>'),
            ('<span className="text-sm">DPoP (Demonstration of Proof of Possession)</span>', '<span className="text-sm">{t("backend.tokenBindingConfig.dpop")}</span>'),
            ('<h2 className="text-lg font-semibold">Proof Token Expiry</h2>', '<h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.proofTokenExpiry")}</h2>'),
            ('<h2 className="text-lg font-semibold">Binding Enforcement Policy</h2>', '<h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.title")}</h2>'),
            ('>Add Override</button>', '>{t("backend.tokenBindingConfig.addOverride")}</button>'),
            ('<option value="dpop">DPoP</option>', '<option value="dpop">{t("backend.tokenBindingConfig.dpop")}</option>'),
            ('<option value="none">None</option>', '<option value="none">{t("backend.tokenBindingConfig.none")}</option>'),
            ('<option value="required">Required</option>', '<option value="required">{t("backend.tokenBindingConfig.required")}</option>'),
            ('<option value="optional">Optional</option>', '<option value="optional">{t("backend.tokenBindingConfig.optional")}</option>'),
            ('<option value="disabled">Disabled</option>', '<option value="disabled">{t("backend.tokenBindingConfig.disabled")}</option>'),
            ('>Remove</button>', '>{t("backend.tokenBindingConfig.remove")}</button>'),
        ],
    },
    {
        'file': 'identity-proofing/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">Identity Proofing</h1>\n          <p className="text-sm text-gray-400 mt-1">Configure and monitor identity verification workflows</p>', '<h1 className="text-2xl font-bold">{t("backend.identityProofing.title")}</h1>\n          <p className="text-sm text-gray-400 mt-1">{t("backend.identityProofing.subtitle")}</p>'),
            ('<h2 className="text-sm text-gray-400 mb-2">Completion Rate</h2>', '<h2 className="text-sm text-gray-400 mb-2">{t("backend.identityProofing.completionRate")}</h2>'),
            ('<h2 className="text-sm text-gray-400 mb-2">Confidence Threshold</h2>', '<h2 className="text-sm text-gray-400 mb-2">{t("backend.identityProofing.confidenceThreshold")}</h2>'),
            ('<h2 className="text-sm text-gray-400 mb-2">In Progress</h2>', '<h2 className="text-sm text-gray-400 mb-2">{t("backend.identityProofing.inProgress")}</h2>'),
            ('<h2 className="text-lg font-semibold mb-4">Configuration</h2>', '<h2 className="text-lg font-semibold mb-4">{t("backend.identityProofing.configuration")}</h2>'),
            ('<label className="text-xs text-gray-400 mb-1 block">Document Type</label>', '<label className="text-xs text-gray-400 mb-1 block">{t("backend.identityProofing.documentType")}</label>'),
            ('<option value="license">Driver License</option>', '<option value="license">{t("backend.identityProofing.driverLicense")}</option>'),
            ('<option value="jumio">Jumio</option>', '<option value="jumio">{t("backend.identityProofing.jumio")}</option>'),
            ('<p className="text-xs text-gray-500">Confidence</p>', '<p className="text-xs text-gray-500">{t("backend.identityProofing.confidence")}</p>'),
        ],
    },
    {
        'file': 'identity-federation/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">Identity Federation</h1><p className="text-gray-600">Configure federation trust relationships with external Identity Providers.</p>', '<h1 className="text-2xl font-bold">{t("backend.identityFederation.title")}</h1><p className="text-gray-600">Configure federation trust relationships with external Identity Providers.</p>'),
            ('<h2 className="text-lg font-semibold">Add Trust Relationship</h2>', '<h2 className="text-lg font-semibold">{t("backend.identityFederation.addTrustRelationship")}</h2>'),
            ('<label className="text-sm font-medium">IdP Name</label>', '<label className="text-sm font-medium">{t("backend.identityFederation.idpName")}</label>'),
            ('<label className="text-sm font-medium">Protocol</label>', '<label className="text-sm font-medium">{t("backend.identityFederation.protocol")}</label>'),
            ('<label className="text-sm font-medium">Metadata URL</label>', '<label className="text-sm font-medium">{t("backend.identityFederation.metadataUrl")}</label>'),
            ('>OpenID Connect</label>', '>{t("backend.identityFederation.openIdConnect")}</label>'),
            ('<div className="text-sm text-gray-500">Active</div>', '<div className="text-sm text-gray-500">{t("backend.identityFederation.active")}</div>'),
            ('<th className="p-3">IdP Name</th>', '<th className="p-3">{t("backend.identityFederation.idpName")}</th>'),
            ('<th className="p-3">Protocol</th>', '<th className="p-3">{t("backend.identityFederation.protocol")}</th>'),
            ('<th className="p-3">Entity ID</th>', '<th className="p-3">{t("backend.identityFederation.entityId")}</th>'),
            ('<th className="p-3">Last Sync</th>', '<th className="p-3">{t("backend.identityFederation.lastSync")}</th>'),
            ('<th className="p-3">Action</th>', '<th className="p-3">{t("backend.identityFederation.action")}</th>'),
        ],
    },
    {
        'file': 'identity-governance/page.tsx',
        'replacements': [
            ('<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-indigo-600" /> Identity Governance</h1>', '<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-indigo-600" /> {t("backend.identityGovernance.title")}</h1>'),
            ('<span className="text-xs font-semibold uppercase text-gray-400">Open Campaigns</span>', '<span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.openCampaigns")}</span>'),
            ('<span className="text-xs font-semibold uppercase text-gray-400">Pending Reviews</span>', '<span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.pendingReviews")}</span>'),
            ('<span className="text-xs font-semibold uppercase text-gray-400">SoD Violations</span>', '<span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.sodViolations")}</span>'),
            ('<span className="text-xs font-semibold uppercase text-gray-400">Orphaned Accounts</span>', '<span className="text-xs font-semibold uppercase text-gray-400">{t("backend.identityGovernance.orphanedAccounts")}</span>'),
            ('<span className="text-gray-500">Completion Rate</span>', '<span className="text-gray-500">{t("backend.identityGovernance.completionRate")}</span>'),
            ('<span className="text-gray-500">Avg Review Time</span>', '<span className="text-gray-500">{t("backend.identityGovernance.avgReviewTime")}</span>'),
            ('<span className="text-gray-500">Dormant Accounts</span>', '<span className="text-gray-500">{t("backend.identityGovernance.dormantAccounts")}</span>'),
            ('<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Recent Campaigns</h2>', '<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.identityGovernance.recentCampaigns")}</h2>'),
        ],
    },
    {
        'file': 'device-management/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Smartphone className="w-6 h-6 text-blue-500" /> Device Management</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Smartphone className="w-6 h-6 text-blue-500" /> {t("backend.deviceManagement.title")}</h1>'),
            ('>Retry</button>', '>{t("backend.deviceManagement.retry")}</button>'),
            ('<span className="text-sm text-gray-500 capitalize">managed</span>', '<span className="text-sm text-gray-500 capitalize">{t("backend.deviceManagement.managed")}</span>'),
            ('<span className="text-sm text-gray-500 capitalize">byod</span>', '<span className="text-sm text-gray-500 capitalize">{t("backend.deviceManagement.byod")}</span>'),
            ('<span className="text-sm text-gray-500">Total</span>', '<span className="text-sm text-gray-500">{t("backend.deviceManagement.total")}</span>'),
            ('<option value="">All Trust Levels</option>', '<option value="">{t("backend.deviceManagement.allTrustLevels")}</option>'),
            ('<option value="managed">Managed</option>', '<option value="managed">{t("backend.deviceManagement.managed")}</option>'),
            ('<option value="byod">BYOD</option>', '<option value="byod">{t("backend.deviceManagement.byod")}</option>'),
            ('<th className="px-4 py-3 text-left font-medium">Device</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.device")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Platform</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.platform")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Last Seen</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.lastSeen")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Fingerprint</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.fingerprint")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Action</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.action")}</th>'),
        ],
    },
    {
        'file': 'device-trust/page.tsx',
        'replacements': [
            ('<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-cyan-600" /> Device Trust</h1>', '<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-cyan-600" /> {t("backend.deviceTrust.title")}</h1>'),
            ('>Block Jailbroken</span>', '>{t("backend.deviceTrust.blockJailbroken")}</span>'),
            ('<label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Min Trust Score</label>', '<label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("backend.deviceTrust.minTrustScore")}</label>'),
            ('<label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Min OS Versions</label>', '<label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("backend.deviceTrust.minOsVersions")}</label>'),
            ('<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Device Inventory</h2>', '<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.deviceTrust.deviceInventory")}</h2>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Device</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.device")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Flags</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.flags")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Seen</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.lastSeen")}</th>'),
            ('>MDM</span>', '>{t("backend.deviceTrust.mdm")}</span>'),
            ('>Encrypted</span>', '>{t("backend.deviceTrust.encrypted")}</span>'),
            ('>Jailbroken</span>', '>{t("backend.deviceTrust.jailbroken")}</span>'),
        ],
    },
    {
        'file': 'device-binding-config/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">Device Binding Configuration</h1>', '<h1 className="text-2xl font-bold">{t("backend.deviceBindingConfig.title2")}</h1>'),
            ('<div className="text-sm text-gray-500">Bound Devices</div>', '<div className="text-sm text-gray-500">{t("backend.deviceBindingConfig.boundDevices")}</div>'),
            ('<th className="p-3">Device</th>', '<th className="p-3">{t("backend.deviceBindingConfig.device")}</th>'),
            ('<th className="p-3">Fingerprint</th>', '<th className="p-3">{t("backend.deviceBindingConfig.fingerprint")}</th>'),
            ('<th className="p-3">Bound At</th>', '<th className="p-3">{t("backend.deviceBindingConfig.boundAt")}</th>'),
            ('<th className="p-3">Last Seen</th>', '<th className="p-3">{t("backend.deviceBindingConfig.lastSeen")}</th>'),
            ('<th className="p-3">Action</th>', '<th className="p-3">{t("backend.deviceBindingConfig.action")}</th>'),
            ('>No data available</td>', '>{t("backend.deviceBindingConfig.noData")}</td>'),
            ('>Cancel</button>', '>{t("backend.deviceBindingConfig.cancel")}</button>'),
            ('>Confirm Unbind</button>', '>{t("backend.deviceBindingConfig.confirmUnbind")}</button>'),
        ],
    },
    {
        'file': 'client-certs/page.tsx',
        'replacements': [
            ('<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-indigo-600" /> Client Certificates</h1>', '<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-indigo-600" /> {t("backend.clientCerts.title")}</h1>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Client</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.client")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Serial</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.serial")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Issuer</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.issuer")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Issued</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.issued")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.expires")}</th>'),
            ('<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>', '<th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.status")}</th>'),
            ('<th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>', '<th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">{t("backend.clientCerts.actions")}</th>'),
        ],
    },
    {
        'file': 'certificate-management/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> Certificate Management</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("backend.certManagement.title")}</h1>'),
            ('>Generate CSR</button>', '>{t("backend.certManagement.generateCsr")}</button>'),
            ('<th className="px-4 py-3 text-left font-medium">Name</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.name")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Issuer</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.issuer")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Expiry</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.expiry")}</th>'),
            ('<th className="px-4 py-3 text-left font-medium">Actions</th>', '<th className="px-4 py-3 text-left font-medium">{t("backend.certManagement.actions")}</th>'),
            ('<span className="text-xs text-gray-400">No</span>', '<span className="text-xs text-gray-400">{t("backend.certManagement.no")}</span>'),
            ('<h3 className="font-semibold">Upload Certificate</h3>', '<h3 className="font-semibold">Upload Certificate</h3>'),  # no key, keep
            ('<label className="text-sm font-medium">Name</label>', '<label className="text-sm font-medium">{t("backend.certManagement.name")}</label>'),
            ('>Cancel</button>', '>{t("backend.certManagement.cancel")}</button>'),
            ('<h3 className="font-semibold">Generate CSR</h3>', '<h3 className="font-semibold">{t("backend.certManagement.generateCsr")}</h3>'),
            ('<label className="text-sm font-medium">CN</label>', '<label className="text-sm font-medium">{t("backend.certManagement.cn")}</label>'),
            ('>Generate</button>', '>{t("backend.certManagement.generate")}</button>'),
            ('<option>JWT</option>', '<option>{t("backend.certManagement.jwt")}</option>'),
        ],
    },
    {
        'file': 'cert-expiry-tracker/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">Certificate Expiry Tracker</h1>', '<h1 className="text-2xl font-bold">{t("backend.certExpiry.title")}</h1>'),
            ('<p className="text-xs text-gray-400">Expired</p>', '<p className="text-xs text-gray-400">{t("backend.certExpiry.expired")}</p>'),
            ('<h2 className="text-sm font-semibold mb-4">Certificates</h2>', '<h2 className="text-sm font-semibold mb-4">{t("backend.certExpiry.certificates")}</h2>'),
            ('<th className="text-left py-2 pr-3">Issuer</th>', '<th className="text-left py-2 pr-3">{t("backend.certExpiry.issuer")}</th>'),
            ('<th className="text-left py-2 pr-3">Expiry</th>', '<th className="text-left py-2 pr-3">{t("backend.certExpiry.expiry")}</th>'),
            ('<th className="text-left py-2 pr-3">Days Left</th>', '<th className="text-left py-2 pr-3">{t("backend.certExpiry.daysLeft")}</th>'),
            ('<h2 className="text-sm font-semibold mb-3">Alert Configuration</h2>', '<h2 className="text-sm font-semibold mb-3">{t("backend.certExpiry.alertConfig")}</h2>'),
            ('<p className="text-xs text-gray-500">First Alert</p>', '<p className="text-xs text-gray-500">{t("backend.certExpiry.firstAlert")}</p>'),
            ('<p className="text-xs text-gray-500">Escalation</p>', '<p className="text-xs text-gray-500">{t("backend.certExpiry.escalation")}</p>'),
            ('<p className="text-xs text-gray-500">Channel</p>', '<p className="text-xs text-gray-500">{t("backend.certExpiry.channel")}</p>'),
        ],
    },
    {
        'file': 'client-lifecycle/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold">OAuth Client Lifecycle</h1>', '<h1 className="text-2xl font-bold">{t("backend.clientLifecycle.title")}</h1>'),
            ('<label className="text-sm font-medium">Client Name</label>', '<label className="text-sm font-medium">{t("backend.clientLifecycle.clientName")}</label>'),
            ('<label className="text-sm font-medium">Grant Types</label>', '<label className="text-sm font-medium">{t("backend.clientLifecycle.grantTypes")}</label>'),
            ('<div className="text-sm text-gray-500">Active</div>', '<div className="text-sm text-gray-500">{t("backend.clientLifecycle.active")}</div>'),
            ('<th className="p-3">Client ID</th>', '<th className="p-3">{t("backend.clientLifecycle.clientId")}</th>'),
            ('<th className="p-3">Name</th>', '<th className="p-3">{t("backend.clientLifecycle.clientName")}</th>'),
            ('<th className="p-3">Grant Types</th>', '<th className="p-3">{t("backend.clientLifecycle.grantTypes")}</th>'),
            ('<th className="p-3">Created</th>', '<th className="p-3">{t("backend.clientLifecycle.created")}</th>'),
            ('<th className="p-3">Actions</th>', '<th className="p-3">{t("backend.clientLifecycle.actions")}</th>'),
            ('>Delete</button>', '>{t("backend.clientLifecycle.delete")}</button>'),
            ('<h2 className="text-lg font-semibold">Delete OAuth Client</h2>', '<h2 className="text-lg font-semibold">{t("backend.clientLifecycle.deleteClient")}</h2>'),
            ('>Cancel</button>', '>{t("backend.clientLifecycle.cancel")}</button>'),
            ('>Confirm Delete</button>', '>{t("backend.clientLifecycle.confirmDelete")}</button>'),
        ],
    },
    {
        'file': 'client-onboarding/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Rocket className="w-6 h-6 text-blue-500" /> Client Onboarding</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Rocket className="w-6 h-6 text-blue-500" /> {t("backend.clientOnboarding.title")}</h1>'),
            ('<label className="text-sm font-medium">App Name</label>', '<label className="text-sm font-medium">{t("backend.clientOnboarding.appName")}</label>'),
            ('<label className="text-sm font-medium">Description</label>', '<label className="text-sm font-medium">{t("backend.clientOnboarding.description")}</label>'),
            ('<div className="text-sm text-gray-500">Redirect URIs</div>', '<div className="text-sm text-gray-500">{t("backend.clientOnboarding.redirectUris")}</div>'),
            ('>Generate Credentials</button>', '>{t("backend.clientOnboarding.generateCredentials")}</button>'),
            ('>Back</button>', '>{t("backend.clientOnboarding.back")}</button>'),
            ('>Next</button>', '>{t("backend.clientOnboarding.next")}</button>'),
            ('>No data available</div>', '>{t("backend.clientOnboarding.noData")}</div>'),
        ],
    },
    {
        'file': 'client-rate-limits/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-cyan-500" /> Client Rate Limits</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-cyan-500" /> {t("backend.clientRateLimits.title")}</h1>'),
            ('<option value="">Select Client</option>', '<option value="">{t("backend.clientRateLimits.selectClient")}</option>'),
            ('<label className="text-sm font-medium">Burst</label>', '<label className="text-sm font-medium">{t("backend.clientRateLimits.burst")}</label>'),
            ('<label className="text-sm font-medium">Daily Quota</label>', '<label className="text-sm font-medium">{t("backend.clientRateLimits.dailyQuota")}</label>'),
        ],
    },
    {
        'file': 'introspection/page.tsx',
        'replacements': [
            ('<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">\n            <KeyRound className="h-6 w-6 text-indigo-600" /> Token Introspection\n          </h1>', '<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">\n            <KeyRound className="h-6 w-6 text-indigo-600" /> {t("backend.introspection.title")}\n          </h1>'),
            ('<h3 className="text-xs font-semibold uppercase text-gray-500">Hit Ratio</h3>', '<h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.hitRatio")}</h3>'),
            ('<h3 className="text-xs font-semibold uppercase text-gray-500">Cache Size</h3>', '<h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.cacheSize")}</h3>'),
            ('<h3 className="text-xs font-semibold uppercase text-gray-500">Avg Response</h3>', '<h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.avgResponse")}</h3>'),
            ('<h3 className="text-xs font-semibold uppercase text-gray-500">Entries</h3>', '<h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.entries")}</h3>'),
            ('<p className="mt-1 text-xs text-gray-400">Active cached tokens</p>', '<p className="mt-1 text-xs text-gray-400">{t("backend.introspection.activeCached")}</p>'),
            ('<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Cached Tokens</h2>', '<h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.introspection.cachedTokens")}</h2>'),
            ('<th className="px-4 py-3">Client ID</th>', '<th className="px-4 py-3">{t("backend.introspection.clientId")}</th>'),
            ('<th className="px-4 py-3">Expires</th>', '<th className="px-4 py-3">{t("backend.introspection.expires")}</th>'),
            ('<th className="px-4 py-3 text-right">Action</th>', '<th className="px-4 py-3 text-right">{t("backend.introspection.action")}</th>'),
            ('>Cancel</button>', '>{t("backend.introspection.cancel")}</button>'),
        ],
    },
    {
        'file': 'jwks/page.tsx',
        'replacements': [
            ('<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">\n            <KeyRound className="h-6 w-6 text-indigo-600" /> JWKS Management\n          </h1>', '<h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">\n            <KeyRound className="h-6 w-6 text-indigo-600" /> {t("backend.jwks.title")}\n          </h1>'),
            ('<h3 className="font-semibold text-gray-800 dark:text-gray-200">Active Key</h3>', '<h3 className="font-semibold text-gray-800 dark:text-gray-200">{t("backend.jwks.activeKey")}</h3>'),
            ('<p className="text-xs font-semibold uppercase text-gray-400">Rotation Interval</p>', '<p className="text-xs font-semibold uppercase text-gray-400">{t("backend.jwks.rotationInterval")}</p>'),
            ('<p className="text-xs font-semibold uppercase text-gray-400">Grace Period</p>', '<p className="text-xs font-semibold uppercase text-gray-400">{t("backend.jwks.gracePeriod")}</p>'),
            ('<th className="px-4 py-3">Key ID</th>', '<th className="px-4 py-3">{t("backend.jwks.keyId")}</th>'),
            ('<th className="px-4 py-3">Retired At</th>', '<th className="px-4 py-3">{t("backend.jwks.retiredAt")}</th>'),
            ('<th className="px-4 py-3">Status</th>', '<th className="px-4 py-3">{t("backend.jwks.status")}</th>'),
            ('>Retired</span>', '>{t("backend.jwks.retired")}</span>'),
            ('>Cancel</button>', '>{t("backend.jwks.cancel")}</button>'),
        ],
    },
    {
        'file': 'discovery-config/page.tsx',
        'replacements': [
            ('<h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> Discovery Config</h1>', '<h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> {t("backend.discoveryConfig.title")}</h1>'),
            ('<span className="text-sm text-gray-500">Issuer</span>', '<span className="text-sm text-gray-500">{t("backend.discoveryConfig.issuer")}</span>'),
            ('<h3 className="text-sm font-semibold mb-3">Supported Scopes</h3>', '<h3 className="text-sm font-semibold mb-3">{t("backend.discoveryConfig.supportedScopes")}</h3>'),
            ('<h3 className="text-sm font-semibold mb-3">Supported Grants</h3>', '<h3 className="text-sm font-semibold mb-3">{t("backend.discoveryConfig.supportedGrants")}</h3>'),
            ('<h3 className="text-sm font-semibold mt-4 mb-3">Signing Algorithms</h3>', '<h3 className="text-sm font-semibold mt-4 mb-3">{t("backend.discoveryConfig.signingAlgorithms")}</h3>'),
            ('<span className="text-sm text-gray-500">Userinfo Endpoint</span>', '<span className="text-sm text-gray-500">{t("backend.discoveryConfig.userinfoEndpoint")}</span>'),
            ('<span className="text-sm text-gray-500">JWKS URI</span>', '<span className="text-sm text-gray-500">{t("backend.discoveryConfig.jwksUri")}</span>'),
        ],
    },
]


def replace_one(content, old, new, path):
    count = content.count(old)
    if count != 1:
        raise ValueError(f"{path}: expected 1 occurrence of {old!r}, found {count}")
    return content.replace(old, new)


for page in PAGES:
    path = ROOT / page['file']
    content = path.read_text()

    # 1. Insert import
    lines = content.splitlines(keepends=True)
    import_line = 'import { useTranslations } from "@/lib/i18n";\n'
    if import_line not in content:
        # Insert after the first line ("use client" directive)
        lines.insert(1, import_line)
        content = ''.join(lines)

    # 2. Insert t = useTranslations()
    def insert_hook(s):
        # Find the export default function declaration line and insert after it
        pattern = re.compile(r'^(export default function \w+\(\) \{)\s*$', re.MULTILINE)
        match = pattern.search(s)
        if not match:
            raise ValueError(f"{path}: could not find default function declaration")
        return s[:match.end()] + '\n  const t = useTranslations();' + s[match.end():]

    content = insert_hook(content)

    # 3. Apply replacements
    for old, new in page['replacements']:
        if old == new:
            continue
        content = replace_one(content, old, new, path)

    path.write_text(content)
    print('updated', path)

print('done')
