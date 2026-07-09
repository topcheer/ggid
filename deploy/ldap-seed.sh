#!/bin/sh
# Seed LDAP with test users
set -e

LDAP_HOST="${LDAP_HOST:-ldap}"
LDAP_PORT="${LDAP_PORT:-389}"
LDAP_ADMIN="cn=admin,dc=corp,dc=local"
LDAP_PASS="${LDAP_ADMIN_PASSWORD:-admin123}"

echo "Waiting for LDAP to be ready..."
sleep 5

# Create OU structure
ldapadd -x -H ldap://${LDAP_HOST}:${LDAP_PORT} -D ${LDAP_ADMIN} -w ${LDAP_PASS} <<'EOF'
dn: ou=users,dc=corp,dc=local
objectClass: organizationalUnit
ou: users

dn: ou=groups,dc=corp,dc=local
objectClass: organizationalUnit
ou: groups
EOF

# Create test users
ldapadd -x -H ldap://${LDAP_HOST}:${LDAP_PORT} -D ${LDAP_ADMIN} -w ${LDAP_PASS} <<'EOF'
dn: cn=johndoe,ou=users,dc=corp,dc=local
objectClass: inetOrgPerson
cn: johndoe
sn: Doe
givenName: John
displayName: John Doe
mail: johndoe@corp.local
userPassword: Password123!

dn: cn=janedoe,ou=users,dc=corp,dc=local
objectClass: inetOrgPerson
cn: janedoe
sn: Doe
givenName: Jane
displayName: Jane Doe
mail: janedoe@corp.local
userPassword: Password456!
EOF

# Create group
ldapadd -x -H ldap://${LDAP_HOST}:${LDAP_PORT} -D ${LDAP_ADMIN} -w ${LDAP_PASS} <<'EOF'
dn: cn=engineers,ou=groups,dc=corp,dc=local
objectClass: groupOfNames
cn: engineers
member: cn=johndoe,ou=users,dc=corp,dc=local
member: cn=janedoe,ou=users,dc=corp,dc=local
EOF

echo "LDAP seed complete"
