package repository

import "errors"

// ErrAAGUIDNotApproved is returned when an authenticator's AAGUID is not in the approved allowlist.
var ErrAAGUIDNotApproved = errors.New("authenticator_not_approved: device AAGUID is not in the approved allowlist")
