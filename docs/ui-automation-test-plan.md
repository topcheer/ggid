# GGID UI Automation Test Plan

> Use this checklist to run full UI automation tests. Update with new features as they are added.
> Each item should be tested via browser automation (browser tool) or manual verification.
> Mark [x] for pass, [ ] for fail, [~] for skip (with reason).

---

## 1. Authentication & Session

### 1.1 Registration
- [ ] Open /register page, verify form renders (username, password, email, display_name fields)
- [ ] Fill form with valid data, submit, verify 201 response
- [ ] Fill form with duplicate username, verify error message displayed
- [ ] Fill form with short password (< 12 chars), verify validation error
- [ ] Fill form with mismatched passwords, verify validation error
- [ ] Click "already have account" link, verify redirect to /login

### 1.2 Login
- [ ] Open /login page, verify form renders (username, password, remember me, sign in button)
- [ ] Verify social login buttons visible (Google, GitHub, SSO)
- [ ] Enter valid credentials, click Sign In, verify redirect to /dashboard
- [ ] Verify JWT token stored in localStorage (ggid_access_token)
- [ ] Verify refresh token stored in localStorage (ggid_refresh_token)
- [ ] Enter invalid password, verify 401 error message displayed on page
- [ ] Enter non-existent user, verify error message displayed
- [ ] Leave username empty, verify validation
- [ ] Leave password empty, verify validation
- [ ] Check "Remember me" checkbox, verify persistent login

### 1.3 Session Management
- [ ] After login, verify /api/v1/auth/me returns user profile
- [ ] Verify /api/v1/tokens returns active sessions list
- [ ] Navigate between pages, verify token persists in localStorage
- [ ] Close browser tab, reopen, verify still logged in (if remember me checked)
- [ ] Click logout, verify redirect to /login, verify localStorage cleared

### 1.4 401 Handling
- [ ] Clear localStorage token, navigate to /users, verify redirect to /login
- [ ] Navigate to /settings/sso with expired token, verify redirect to /login
- [ ] Verify /login page itself does NOT redirect when receiving 401
- [ ] Verify no redirect loop on /login page

### 1.5 Password Management
- [ ] Navigate to password change page, verify form renders
- [ ] Change password with valid old + new password, verify success
- [ ] Change password with wrong old password, verify error
- [ ] Navigate to forgot password page, verify email input
- [ ] Submit forgot password, verify success message (even without SMTP)

---

## 2. Dashboard

### 2.1 Overview
- [ ] Navigate to /, verify dashboard renders
- [ ] Verify stat cards visible (users count, roles count, orgs count, audit events)
- [ ] Verify recent activity section renders
- [ ] Verify quick action buttons visible
- [ ] Verify sidebar navigation renders with all sections

### 2.2 Sidebar Navigation
- [ ] Verify sidebar sections: Overview, Management, Security, System
- [ ] Click each sidebar item, verify page loads
- [ ] Verify active state highlighting on current page
- [ ] Verify sidebar collapses on mobile viewport

---

## 3. User Management

### 3.1 User List
- [ ] Navigate to /users, verify data table renders
- [ ] Verify columns: username, email, roles, status, created date
- [ ] Verify pagination controls visible
- [ ] Click column header, verify sorting works
- [ ] Enter search text, verify filtering works
- [ ] Verify page size selector works

### 3.2 User Create
- [ ] Click "Create User" button, verify modal/form opens
- [ ] Fill user form, submit, verify new user appears in table
- [ ] Create user with duplicate username, verify error
- [ ] Create user with invalid email, verify validation

### 3.3 User Edit
- [ ] Click edit button on a user row, verify edit form opens
- [ ] Modify user fields, submit, verify changes saved
- [ ] Change user role, verify role updated

### 3.4 User Delete
- [ ] Click delete button on a user row, verify confirmation dialog
- [ ] Confirm deletion, verify user removed from table
- [ ] Cancel deletion, verify user still in table

### 3.5 User Detail
- [ ] Click user row, verify detail view opens
- [ ] Verify user info, roles, sessions, audit trail sections
- [ ] Verify "Assign Role" button works
- [ ] Verify "Revoke Session" button works

---

## 4. Roles & Permissions

### 4.1 Role List
- [ ] Navigate to /roles, verify role list renders
- [ ] Verify columns: name, key, description, permissions count, users count

### 4.2 Role Create
- [ ] Click "Create Role", verify form opens
- [ ] Fill role name, key, description, submit, verify role created
- [ ] Create role with duplicate key, verify error

### 4.3 Role Permissions
- [ ] Click a role, verify permission matrix renders
- [ ] Toggle a permission on, verify saved
- [ ] Toggle a permission off, verify saved
- [ ] Verify permission categories display correctly

### 4.4 Role Delete
- [ ] Delete a role, verify confirmation and removal

### 4.5 Permission Tree
- [ ] Navigate to /settings/permission-tree, verify tree structure renders
- [ ] Expand/collapse tree nodes, verify works
- [ ] Verify permission inheritance display

---

## 5. Organizations

### 5.1 Org List
- [ ] Navigate to /organizations, verify org list renders
- [ ] Verify columns: name, member count, created date

### 5.2 Org Create
- [ ] Click "Create Org", fill form, submit, verify org created

### 5.3 Org Detail
- [ ] Click an org, verify detail view with members tab
- [ ] Add member to org, verify member appears
- [ ] Remove member from org, verify member removed

### 5.4 Org Delete
- [ ] Delete an org, verify confirmation and removal

### 5.5 Org Analytics
- [ ] Navigate to /organizations/analytics, verify charts render
- [ ] Verify member growth chart
- [ ] Verify activity timeline

---

## 6. Policies

### 6.1 Policy List
- [ ] Navigate to /policies, verify policy list renders
- [ ] Verify columns: name, effect, actions, resources, status

### 6.2 Policy Create
- [ ] Click "Create Policy", fill form (name, effect, actions, resources), submit
- [ ] Verify policy appears in list

### 6.3 Policy Edit
- [ ] Edit a policy, modify effect (allow→deny), verify saved

### 6.4 Policy Delete
- [ ] Delete a policy, verify removal

### 6.5 Policy Dry Run
- [ ] Navigate to policy simulation/sandbox page
- [ ] Enter subject, action, resource, click "Evaluate"
- [ ] Verify decision result displayed (allow/deny)

---

## 7. Audit Log

### 7.1 Event List
- [ ] Navigate to /audit, verify event table renders
- [ ] Verify columns: timestamp, user, action, resource, IP, status
- [ ] Verify events are sorted by timestamp (newest first)

### 7.2 Event Filtering
- [ ] Filter by date range, verify filtered results
- [ ] Filter by user, verify filtered results
- [ ] Filter by action type, verify filtered results
- [ ] Clear filters, verify all events show

### 7.3 Event Detail
- [ ] Click an event row, verify detail modal/panel opens
- [ ] Verify full event metadata displayed

### 7.4 Hash Chain
- [ ] Navigate to audit hash chain verification page
- [ ] Verify hash chain status displayed
- [ ] Verify chain integrity check result

---

## 8. Security Center

### 8.1 Security Overview
- [ ] Navigate to /security, verify overview dashboard renders
- [ ] Verify security score/risk level displayed
- [ ] Verify threat indicators visible

### 8.2 Incidents
- [ ] Navigate to /security/incidents, verify incident list renders
- [ ] Click an incident, verify detail view
- [ ] Change incident status, verify saved

### 8.3 Threats
- [ ] Navigate to /security/threats, verify threat list renders

### 8.4 Access Requests
- [ ] Navigate to /access-requests, verify request list renders
- [ ] Create a new access request, verify it appears
- [ ] Approve/reject a request, verify status change

---

## 9. AI Agents

### 9.1 Agent List
- [ ] Navigate to /agents, verify agent list renders
- [ ] Verify columns: name, status, scopes, last active

### 9.2 Agent Register
- [ ] Click "Register Agent", fill form, submit
- [ ] Verify agent appears in list
- [ ] Verify agent credentials (client_id, client_secret) displayed

### 9.3 Agent Token Exchange
- [ ] Click "Exchange Token" on an agent
- [ ] Verify token displayed

### 9.4 Agent Suspend/Resume
- [ ] Suspend an agent, verify status changes
- [ ] Resume an agent, verify status changes

---

## 10. OAuth / OIDC

### 10.1 Client List
- [ ] Navigate to /settings/oauth-clients, verify client list renders

### 10.2 Client Register
- [ ] Click "Register Client", fill form, submit
- [ ] Verify client appears in list with client_id

### 10.3 Client Detail
- [ ] Click a client, verify detail view with scopes, redirect URIs

### 10.4 Client Delete
- [ ] Delete a client, verify removal

### 10.5 OIDC Discovery
- [ ] Verify /.well-known/openid-configuration returns valid JSON
- [ ] Verify issuer, authorization_endpoint, token_endpoint present
- [ ] Verify /.well-known/jwks.json returns valid keys

### 10.6 OAuth Flows
- [ ] Navigate to /oauth/flows page, verify flow diagram renders
- [ ] Verify authorization code flow steps displayed
- [ ] Verify PKCE flow steps displayed
- [ ] Verify client credentials flow steps displayed

---

## 11. Settings

### 11.1 SSO Configuration
- [ ] Navigate to /settings/sso, verify SSO provider list renders
- [ ] Click "Add Provider", verify form opens
- [ ] Fill SAML provider config, verify save
- [ ] Fill OIDC provider config, verify save
- [ ] Test SSO connection, verify test result

### 11.2 API Keys
- [ ] Navigate to /settings/api-keys, verify key list renders
- [ ] Click "Create API Key", fill form, verify key generated
- [ ] Verify key secret displayed (only once)
- [ ] Revoke a key, verify removal

### 11.3 MFA Configuration
- [ ] Navigate to /settings/mfa, verify MFA settings render
- [ ] Enable TOTP MFA, verify saved
- [ ] Verify QR code displayed for TOTP setup
- [ ] Configure MFA policy (required/optional), verify saved

### 11.4 Certificates
- [ ] Navigate to /settings/certificates, verify cert list renders
- [ ] Upload a certificate, verify it appears
- [ ] Verify cert expiry date displayed
- [ ] Delete a certificate, verify removal

### 11.5 Branding
- [ ] Navigate to /settings/branding, verify branding form renders
- [ ] Upload logo, verify preview
- [ ] Change primary color, verify preview
- [ ] Save branding, verify persistence

### 11.6 Tenant Config
- [ ] Navigate to /settings/tenant-config, verify config form renders
- [ ] Modify tenant name, verify save
- [ ] Modify session timeout, verify save

### 11.7 Login Flows
- [ ] Navigate to /settings/login-flows, verify flow builder renders
- [ ] Add a login step (e.g., MFA), verify it appears in flow
- [ ] Reorder login steps, verify order saved
- [ ] Save login flow, verify persistence

---

## 12. Internationalization

### 12.1 Language Switch
- [ ] On any page, click language switcher (EN button)
- [ ] Verify page text changes to Chinese (中文)
- [ ] Verify sidebar items translated
- [ ] Switch back to English, verify text reverts
- [ ] Navigate to another page, verify language persists

### 12.2 Translation Coverage
- [ ] /dashboard — verify no hardcoded English in Chinese mode
- [ ] /users — verify table headers translated
- [ ] /roles — verify all labels translated
- [ ] /settings/sso — verify form labels translated
- [ ] /audit — verify column headers translated
- [ ] /login — verify all text translated

---

## 13. Theme & Responsive

### 13.1 Dark/Light Theme
- [ ] Click theme toggle, verify dark mode applied
- [ ] Verify sidebar, cards, tables render correctly in dark mode
- [ ] Verify text contrast is readable in dark mode
- [ ] Switch to light mode, verify all elements render correctly
- [ ] Verify theme persists on page navigation

### 13.2 Responsive Layout
- [ ] Resize to mobile width (375px), verify sidebar collapses
- [ ] Verify tables become scrollable on mobile
- [ ] Verify forms are usable on mobile
- [ ] Verify buttons are tappable on mobile
- [ ] Resize to tablet (768px), verify layout adapts

---

## 14. Webhooks & Notifications

### 14.1 Webhooks
- [ ] Navigate to webhook config page, verify list renders
- [ ] Create a webhook, fill URL + events, verify saved
- [ ] Test webhook delivery, verify test result
- [ ] Delete a webhook, verify removal

### 14.2 Notifications
- [ ] Verify notification bell icon visible in header
- [ ] Click bell, verify notification dropdown renders
- [ ] Verify unread count badge displayed
- [ ] Mark notification as read, verify badge updates

---

## 15. SIEM & Compliance

### 15.1 SIEM Forwarder
- [ ] Navigate to SIEM config page, verify status renders
- [ ] Verify forwarder health status (healthy/unhealthy)
- [ ] Configure SIEM destination, verify save
- [ ] Test SIEM forwarding, verify test event sent

### 15.2 Compliance
- [ ] Navigate to compliance schedules page, verify list renders
- [ ] Create compliance schedule, verify saved
- [ ] View compliance report, verify renders

---

## 16. SoD (Segregation of Duties)

### 16.1 SoD Rules
- [ ] Navigate to SoD rules config page, verify list renders
- [ ] Create SoD rule (conflicting roles), verify saved
- [ ] Verify rule appears in list

### 16.2 SoD Conflicts
- [ ] Navigate to SoD conflict detection page
- [ ] Run conflict check, verify results displayed
- [ ] Verify conflicting users/roles listed

---

## 17. Advanced Access Control

### 17.1 ABAC
- [ ] Navigate to ABAC config page, verify attribute list renders
- [ ] Create ABAC policy with conditions, verify saved
- [ ] Evaluate ABAC policy, verify decision

### 17.2 Delegation
- [ ] Navigate to delegation page, verify list renders
- [ ] Create delegation (delegate to another user), verify saved
- [ ] Verify delegation appears in list
- [ ] Revoke delegation, verify removed

### 17.3 Account Linking
- [ ] Navigate to account linking page, verify list renders
- [ ] Link external account, verify saved
- [ ] Unlink account, verify removed

### 17.4 Device Bindings
- [ ] Navigate to device bindings page, verify list renders
- [ ] Register device, verify saved
- [ ] Unbind device, verify removed

---

## 18. Rate Limiting & Security

### 18.1 Rate Limits
- [ ] Navigate to rate limits config page, verify list renders
- [ ] Create rate limit rule, verify saved
- [ ] Verify existing rules displayed

### 18.2 Security Headers
- [ ] Inspect HTTP response headers, verify X-Content-Type-Options: nosniff
- [ ] Verify X-Frame-Options: DENY or SAMEORIGIN
- [ ] Verify Strict-Transport-Security header present
- [ ] Verify Content-Security-Policy header present

### 18.3 CORS
- [ ] Verify CORS headers present on API responses
- [ ] Verify preflight OPTIONS requests handled

---

## 19. Demo App Integration

### 19.1 OAuth Authorization Code Flow
- [ ] Register an OAuth client with redirect_uri
- [ ] Build authorization URL with client_id, redirect_uri, scope, state
- [ ] Navigate to authorization URL, verify consent page
- [ ] Approve consent, verify redirect with code
- [ ] Exchange code for token, verify access_token returned
- [ ] Use token to call /api/v1/users, verify 200

### 19.2 Token Introspection
- [ ] Call /oauth/introspect with valid token, verify active: true
- [ ] Call /oauth/introspect with expired token, verify active: false

### 19.3 Token Revocation
- [ ] Call /oauth/revoke with valid token
- [ ] Verify token no longer works on API calls

### 19.4 UserInfo
- [ ] Call /oauth/userinfo with valid token, verify user profile returned

---

## 20. Error Handling & Edge Cases

### 20.1 404 Pages
- [ ] Navigate to /nonexistent-page, verify 404 page renders
- [ ] Verify 404 page has link back to dashboard

### 20.2 Network Errors
- [ ] Simulate network failure, verify error message displayed
- [ ] Verify retry button works

### 20.3 Empty States
- [ ] Navigate to page with no data, verify empty state message
- [ ] Verify empty state has CTA button

### 20.4 Large Data
- [ ] Load page with 1000+ records, verify pagination works
- [ ] Verify no UI freeze or crash

---

## 21. Performance

### 21.1 Page Load Times
- [ ] Dashboard loads in < 2s
- [ ] Users page loads in < 2s
- [ ] Audit page loads in < 3s
- [ ] Settings pages load in < 2s

### 21.2 API Response Times
- [ ] Login API responds in < 500ms
- [ ] User list API responds in < 500ms
- [ ] Audit events API responds in < 1s

---

## Execution Notes

- Use `browser` tool for all UI interactions
- Use `curl` with `-H 'Accept-Encoding: identity'` for API verification
- Register fresh test user for each full run to avoid rate limiting
- Test in both light and dark mode
- Test in both English and Chinese
- Document any failures with screenshots
