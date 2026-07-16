// LDAP vendor presets for common directory servers.
// Each preset pre-fills sensible defaults when user selects a vendor.

export interface LDAPVendorPreset {
  id: string;
  label: string;
  description: string;
  config: {
    server_url: string;
    bind_dn: string;
    base_dn: string;
    user_filter: string;
    group_filter: string;
    start_tls: boolean;
    attribute_mapping: Record<string, string>;
  };
}

export const LDAP_VENDOR_PRESETS: LDAPVendorPreset[] = [
  {
    id: "custom",
    label: "Custom LDAP",
    description: "Generic LDAP v3 directory server",
    config: {
      server_url: "",
      bind_dn: "",
      base_dn: "",
      user_filter: "(objectClass=person)",
      group_filter: "(objectClass=groupOfNames)",
      start_tls: true,
      attribute_mapping: {
        uid: "username",
        mail: "email",
        cn: "display_name",
        memberOf: "groups",
      },
    },
  },
  {
    id: "active-directory",
    label: "Active Directory",
    description: "Microsoft AD / Windows Server",
    config: {
      server_url: "ldap://dc01.corp.local:389",
      bind_dn: "CN=svc-ldap,OU=Service Accounts,DC=corp,DC=local",
      base_dn: "DC=corp,DC=local",
      user_filter: "(&(objectClass=user)(objectCategory=person)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))",
      group_filter: "(objectClass=group)",
      start_tls: true,
      attribute_mapping: {
        sAMAccountName: "username",
        userPrincipalName: "email",
        displayName: "display_name",
        memberOf: "groups",
        givenName: "first_name",
        sn: "last_name",
      },
    },
  },
  {
    id: "openldap",
    label: "OpenLDAP",
    description: "Open-source LDAP server",
    config: {
      server_url: "ldap://ldap.example.com:389",
      bind_dn: "cn=admin,dc=example,dc=com",
      base_dn: "dc=example,dc=com",
      user_filter: "(objectClass=inetOrgPerson)",
      group_filter: "(objectClass=groupOfNames)",
      start_tls: true,
      attribute_mapping: {
        uid: "username",
        mail: "email",
        cn: "display_name",
        memberOf: "groups",
      },
    },
  },
  {
    id: "freeipa",
    label: "FreeIPA / IdM",
    description: "Red Hat Identity Management",
    config: {
      server_url: "ldap://ipa.example.com:389",
      bind_dn: "uid=ldap-sync,cn=users,cn=accounts,dc=example,dc=com",
      base_dn: "dc=example,dc=com",
      user_filter: "(objectClass=posixAccount)",
      group_filter: "(objectClass=posixGroup)",
      start_tls: true,
      attribute_mapping: {
        uid: "username",
        mail: "email",
        displayName: "display_name",
        memberOf: "groups",
      },
    },
  },
  {
    id: "389ds",
    label: "389 Directory Server",
    description: "Red Hat 389 DS",
    config: {
      server_url: "ldap://dir.example.com:389",
      bind_dn: "cn=Directory Manager",
      base_dn: "dc=example,dc=com",
      user_filter: "(objectClass=inetOrgPerson)",
      group_filter: "(objectClass=groupOfUniqueNames)",
      start_tls: true,
      attribute_mapping: {
        uid: "username",
        mail: "email",
        cn: "display_name",
        uniqueMember: "groups",
      },
    },
  },
  {
    id: "open-dj",
    label: "OpenDJ",
    description: "ForgeRock OpenDJ",
    config: {
      server_url: "ldap://opendj.example.com:389",
      bind_dn: "cn=Directory Manager",
      base_dn: "dc=example,dc=com",
      user_filter: "(objectClass=person)",
      group_filter: "(objectClass=groupOfUniqueNames)",
      start_tls: true,
      attribute_mapping: {
        uid: "username",
        mail: "email",
        cn: "display_name",
        isMemberOf: "groups",
      },
    },
  },
];
