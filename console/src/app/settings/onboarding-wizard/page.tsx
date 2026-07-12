'use client';
import { useState } from 'react';

const steps = ['Organization', 'Admin Account', 'SSO Config', 'MFA Setup', 'Password Policy', 'Branding', 'Review'];

export default function OnboardingWizardPage() {
  const [current, setCurrent] = useState(0);
  const [data, setData] = useState({ orgName: '', adminEmail: '', ssoProvider: 'none', mfaType: 'totp', minLen: 12, logo: '#3B82F6' });
  const [completed, setCompleted] = useState(false);
  const [skipped, setSkipped] = useState<number[]>([]);

  const next = () => { if (current < steps.length - 1) setCurrent(c => c + 1); else setCompleted(true); };
  const prev = () => { if (current > 0) setCurrent(c => c - 1); };
  const skip = () => { setSkipped(prev => [...prev, current]); next(); };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Onboarding Wizard</h1><p className="text-gray-600">7-step setup wizard for new GGID tenants.</p></div>

      <div className="flex gap-1">
        {steps.map((s, i) => (
          <div key={s} className="flex-1">
            <div className={`h-2 rounded-full ${i <= current ? 'bg-blue-600' : 'bg-gray-200'}`} />
            <div className={`text-xs mt-1 text-center ${i === current ? 'font-bold text-blue-600' : i < current ? 'text-green-600' : 'text-gray-400'}`}>{i + 1}. {s}</div>
            {skipped.includes(i) && <div className="text-xs text-amber-600 text-center">skipped</div>}
          </div>
        ))}
      </div>

      {!completed ? (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Step {current + 1}: {steps[current]}</h2>
          {current === 0 && <div><label className="text-sm font-medium">Organization Name</label><input type="text" value={data.orgName} onChange={e => setData(prev => ({ ...prev, orgName: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>}
          {current === 1 && <div><label className="text-sm font-medium">Admin Email</label><input type="email" value={data.adminEmail} onChange={e => setData(prev => ({ ...prev, adminEmail: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>}
          {current === 2 && <div><label className="text-sm font-medium">SSO Provider</label><select value={data.ssoProvider} onChange={e => setData(prev => ({ ...prev, ssoProvider: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="none">None (skip)</option><option value="saml">SAML</option><option value="oidc">OIDC</option><option value="google">Google Social</option></select></div>}
          {current === 3 && <div><label className="text-sm font-medium">MFA Type</label><select value={data.mfaType} onChange={e => setData(prev => ({ ...prev, mfaType: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="totp">TOTP (Google Auth)</option><option value="webauthn">WebAuthn</option><option value="sms">SMS</option></select></div>}
          {current === 4 && <div><label className="text-sm font-medium">Minimum Password Length: {data.minLen}</label><input type="range" min={8} max={32} value={data.minLen} onChange={e => setData(prev => ({ ...prev, minLen: parseInt(e.target.value) }))} className="w-full mt-2" /></div>}
          {current === 5 && <div><label className="text-sm font-medium">Brand Color</label><input type="color" value={data.logo} onChange={e => setData(prev => ({ ...prev, logo: e.target.value }))} className="w-20 h-10 rounded mt-1" /></div>}
          {current === 6 && <div className="space-y-2 text-sm"><div><strong>Org:</strong> {data.orgName || '(not set)'}</div><div><strong>Admin:</strong> {data.adminEmail || '(not set)'}</div><div><strong>SSO:</strong> {data.ssoProvider}</div><div><strong>MFA:</strong> {data.mfaType}</div><div><strong>Min Password:</strong> {data.minLen}</div><div><strong>Brand Color:</strong> {data.logo}</div><div className="text-xs text-amber-600">{skipped.length} step(s) skipped</div></div>}

          <div className="flex justify-between pt-4">
            <button onClick={prev} disabled={current === 0} className="px-4 py-2 border rounded text-sm disabled:opacity-50">Previous</button>
            <div className="flex gap-2">
              {current < 6 && <button onClick={skip} className="px-4 py-2 text-gray-500 text-sm">Skip</button>}
              <button onClick={next} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{current === 6 ? 'Complete Setup' : 'Next'}</button>
            </div>
          </div>
        </section>
      ) : (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold text-green-600">Setup Complete!</h2>
          <p className="text-sm text-gray-600">Your GGID tenant has been configured. You can modify these settings anytime from the Settings menu.</p>
          <button onClick={() => { setCurrent(0); setCompleted(false); setSkipped([]); }} className="px-4 py-2 border rounded text-sm">Start Over</button>
        </section>
      )}
    </div>
  );
}