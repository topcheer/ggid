package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/go-ldap/ldap/v3"
	"github.com/ggid/ggid/pkg/authprovider"
)

// testLDAPConnection connects to the LDAP server, binds with the service
// account, and counts users and groups matching the configured filters.
func (h *HTTPHandler) testLDAPConnection(ctx context.Context, cfg *LDAPSyncConfig) (*ldapTestResult, error) {
	conn, err := dialLDAP(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("bind: %w", err)
	}

	userCount, err := countLDAPEntries(conn, cfg.BaseDN, cfg.UserFilter)
	if err != nil {
		return nil, fmt.Errorf("user search: %w", err)
	}

	groupCount, err := countLDAPEntries(conn, cfg.BaseDN, cfg.GroupFilter)
	if err != nil {
		return nil, fmt.Errorf("group search: %w", err)
	}

	return &ldapTestResult{
		usersFound:  userCount,
		groupsFound: groupCount,
	}, nil
}

// runLDAPSync performs a full LDAP user sync: connects, searches for users,
// and provisions each one via the identity service's ProvisionFromLDAP.
func (h *HTTPHandler) runLDAPSync(ctx context.Context, cfg *LDAPSyncConfig) (*ldapTestResult, []map[string]any) {
	var errs []map[string]any

	conn, err := dialLDAP(cfg)
	if err != nil {
		errs = append(errs, formatLDAPError("", fmt.Sprintf("connect: %v", err)))
		return &ldapTestResult{}, errs
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		errs = append(errs, formatLDAPError("", fmt.Sprintf("bind: %v", err)))
		return &ldapTestResult{}, errs
	}

	searchReq := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		cfg.UserFilter,
		[]string{"mail", "displayName", "cn", "sAMAccountName", "uid", "givenName", "sn", "memberOf"},
		nil,
	)

	sr, err := conn.Search(searchReq)
	if err != nil {
		errs = append(errs, formatLDAPError("", fmt.Sprintf("search: %v", err)))
		return &ldapTestResult{}, errs
	}

	synced := 0
	for _, entry := range sr.Entries {
		attrs := map[string]any{}
		for _, attr := range entry.Attributes {
			if len(attr.Values) == 1 {
				attrs[attr.Name] = attr.Values[0]
			} else {
				attrs[attr.Name] = attr.Values
			}
		}
		attrs["dn"] = entry.DN

		externalID := getStr(attrs, "sAMAccountName")
		if externalID == "" {
			externalID = getStr(attrs, "uid")
		}
		if externalID == "" {
			continue
		}

		result := &authprovider.AuthResult{
			ExternalID: externalID,
			Attributes: attrs,
			Provider:   "ldap",
		}

		_, provisionErr := h.svc.ProvisionFromLDAP(ctx, result)
		if provisionErr != nil {
			errs = append(errs, formatLDAPError(entry.DN, provisionErr.Error()))
			continue
		}
		synced++
	}

	slog.Info("LDAP sync complete", "synced", synced, "errors", len(errs))
	return &ldapTestResult{usersFound: synced}, errs
}

// dialLDAP creates and returns an LDAP connection based on config.
func dialLDAP(cfg *LDAPSyncConfig) (*ldap.Conn, error) {
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("server_url is empty")
	}

	conn, err := ldap.DialURL(cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	if cfg.StartTLS {
		if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
			conn.Close()
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}

	return conn, nil
}

// countLDAPEntries runs a search and returns the count of entries.
func countLDAPEntries(conn *ldap.Conn, baseDN, filter string) (int, error) {
	if filter == "" || baseDN == "" {
		return 0, nil
	}
	searchReq := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"dn"},
		nil,
	)
	sr, err := conn.Search(searchReq)
	if err != nil {
		return 0, err
	}
	return len(sr.Entries), nil
}

// getStr extracts a string value from a map[string]any.
func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// formatLDAPError creates a user-friendly error map.
func formatLDAPError(userDN, msg string) map[string]any {
	return map[string]any{
		"dn":    userDN,
		"error": fmt.Sprintf("ldap: %s", msg),
	}
}
