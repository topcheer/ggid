'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Factor {
  id: string;
  type: string;
  label: string;
  enrolledAt: string;
  lastUsed: string;
  priority: number;
}

export default function MfaEnrollmentPage() {
  const t = useTranslations();

  const [factors, setFactors] = useState<Factor[]>([]);

  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState(0);
  const [selectedType, setSelectedType] = useState('');
  const [showRecoveryCodes, setShowRecoveryCodes] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);

  const factorTypes = ['TOTP', 'WebAuthn', 'SMS', 'Email', 'Backup Codes'];
  const [stats, setStats] = useState({ enrolled: 0, pending: 0, totp: 0, webauthn: 0, backupCodes: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/mfa/enrollment-stats', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.factors) setFactors(data.factors);
          if (data.recovery_codes) setRecoveryCodes(data.recovery_codes);
          if (data.stats) setStats(data.stats);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const removeFactor = (id: string) => setFactors(prev => prev.filter(f => f.id !== id));

  const typeColor = (t: string): string =>
    t === 'TOTP' ? 'bg-blue-100 text-blue-700' :
    t === 'WebAuthn' ? 'bg-purple-100 text-purple-700' :
    t === 'SMS' ? 'bg-green-100 text-green-700' :
    t === 'Email' ? 'bg-yellow-100 text-yellow-700' :
    'bg-gray-100 text-gray-700';

  const finishWizard = () => {
    setFactors(prev => [...prev, { id: `f${prev.length + 1}`, type: selectedType, label: selectedType, enrolledAt: new Date().toISOString().slice(0, 10), lastUsed: '-', priority: prev.length + 1 }]);
    setShowWizard(false); setWizardStep(0); setSelectedType('');
    if (selectedType === 'TOTP') setShowRecoveryCodes(true);
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">MFA Enrollment Center</h1><p className="text-gray-600">Enroll, manage, and monitor multi-factor authentication factors.</p></div>
        <button onClick={() => setShowWizard(!showWizard)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showWizard ? 'Cancel' : 'Enroll New Factor'}</button>
      </div>

      <div className="grid grid-cols-5 gap-3">
        <div className="bg-white rounded-lg shadow p-3 text-center"><div className="text-lg font-bold">{stats.enrolled}</div><div className="text-xs text-gray-500">Enrolled</div></div>
        <div className="bg-white rounded-lg shadow p-3 text-center"><div className="text-lg font-bold text-amber-600">{stats.pending}</div><div className="text-xs text-gray-500">Pending</div></div>
        <div className="bg-white rounded-lg shadow p-3 text-center"><div className="text-lg font-bold text-blue-600">{stats.totp}</div><div className="text-xs text-gray-500">TOTP</div></div>
        <div className="bg-white rounded-lg shadow p-3 text-center"><div className="text-lg font-bold text-purple-600">{stats.webauthn}</div><div className="text-xs text-gray-500">WebAuthn</div></div>
        <div className="bg-white rounded-lg shadow p-3 text-center"><div className="text-lg font-bold text-gray-600">{stats.backupCodes}</div><div className="text-xs text-gray-500">Backup Codes</div></div>
      </div>

      {showWizard && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Enrollment Wizard — Step {wizardStep + 1}/4</h2>
          <div className="flex gap-2">
            {['Select', 'Configure', 'Verify', 'Backup'].map((s, i) => (
              <div key={s} className={`flex-1 text-center text-xs py-1 rounded ${i <= wizardStep ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-400'}`}>{i + 1}. {s}</div>
            ))}
          </div>
          {wizardStep === 0 && (
            <div className="space-y-2">
              <label className="text-sm font-medium">Select Factor Type</label>
              <div className="flex flex-wrap gap-3">
                {factorTypes.map(t => <label key={t} className={`px-4 py-2 rounded border text-sm cursor-pointer ${selectedType === t ? 'border-blue-500 bg-blue-50' : 'border-gray-200'}`}><input aria-label="Selected type" type="radio" checked={selectedType === t} onChange={() => setSelectedType(t)} className="hidden" />{t}</label>)}
              </div>
              <button onClick={() => setWizardStep(1)} disabled={!selectedType} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Next</button>
            </div>
          )}
          {wizardStep === 1 && (
            <div className="space-y-3">
              <p className="text-sm text-gray-600">Configure {selectedType}: Scan QR code or enter secret manually.</p>
              <div className="bg-gray-100 rounded p-8 text-center text-sm text-gray-400">QR Code Placeholder</div>
              <div className="font-mono text-xs bg-gray-900 text-green-400 rounded p-2">Secret: JBSWY3DPEHPK3PXP</div>
              <button onClick={() => setWizardStep(2)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Next</button>
            </div>
          )}
          {wizardStep === 2 && (
            <div className="space-y-3">
              <label className="text-sm font-medium">Enter verification code</label>
              <input aria-label="123456" type="text" placeholder="123456" className="w-32 border rounded px-3 py-2 text-sm font-mono" />
              <button onClick={() => setWizardStep(3)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Verify</button>
            </div>
          )}
          {wizardStep === 3 && (
            <div className="space-y-3">
              <p className="text-sm text-gray-600">Save your backup codes. They can be used if you lose access to your factor.</p>
              <div className="grid grid-cols-4 gap-2 font-mono text-xs">{recoveryCodes.map(c => <div key={c} className="bg-gray-100 rounded p-2 text-center">{c}</div>)}</div>
              <button onClick={finishWizard} className="px-4 py-2 bg-green-600 text-white rounded text-sm">Complete Enrollment</button>
            </div>
          )}
        </section>
      )}

      {showRecoveryCodes && (
        <div className="bg-amber-50 border border-amber-200 rounded p-4 space-y-2">
          <div className="font-medium text-amber-800 text-sm">Recovery Codes (one-time view):</div>
          <div className="grid grid-cols-4 gap-2 font-mono text-xs">{recoveryCodes.map(c => <div key={c} className="bg-white rounded p-2 text-center">{c}</div>)}</div>
          <button onClick={() => setShowRecoveryCodes(false)} className="text-xs text-blue-600">I've saved these codes</button>
        </div>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Enrolled Factors</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Type</th><th className="p-3">Label</th><th className="p-3">Enrolled</th><th className="p-3">Last Used</th><th className="p-3">Priority</th><th className="p-3">Action</th></tr></thead>
          <tbody>
            {factors.map(f => (
              <tr key={f.id} className="border-b">
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${typeColor(f.type)}`}>{f.type}</span></td>
                <td className="p-3 font-medium">{f.label}</td>
                <td className="p-3 text-gray-500">{f.enrolledAt}</td>
                <td className="p-3 text-gray-500">{f.lastUsed}</td>
                <td className="p-3">{f.priority}</td>
                <td className="p-3"><button onClick={() => removeFactor(f.id)} className="text-red-600 text-xs hover:underline">Remove</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}