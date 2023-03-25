package repo

import (
	"fmt"
)

var errServiceAccountMismatch = fmt.Errorf("service account mismatch in nodeid vs xds certificate common name")
