//go:build windows

package filesystem

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// trustedSIDs are the accounts that may access a secret: the current user, plus
// SYSTEM and Administrators, which can take any file anyway. OpenSSH applies the
// same rule to private keys.
var trustedSIDs = sync.OnceValues(func() ([]*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("looking up current user: %w", err)
	}

	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, fmt.Errorf("looking up SYSTEM account: %w", err)
	}

	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, fmt.Errorf("looking up Administrators group: %w", err)
	}

	return []*windows.SID{user.User.Sid, system, admins}, nil
})

// checkSecretAccess requires that the access control list of path grants access to
// trusted accounts only. Windows reports no meaningful permission bits, so info and
// perm are ignored.
func checkSecretAccess(path, kind string, _ os.FileInfo, _ os.FileMode) error {
	trusted, err := trustedSIDs()
	if err != nil {
		return err
	}

	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("reading access control list of %s: %w", path, err)
	}

	dacl, _, err := sd.DACL()
	// A missing or null DACL grants everyone full access.
	if errors.Is(err, windows.ERROR_OBJECT_NOT_FOUND) || (err == nil && dacl == nil) {
		return fmt.Errorf("secret %s %s must only be accessible by you, but is accessible by everyone", kind, path)
	}
	if err != nil {
		return fmt.Errorf("reading access control list of %s: %w", path, err)
	}

	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return fmt.Errorf("reading access control list of %s: %w", path, err)
		}

		// Deny entries only restrict access, and inherit-only entries apply to children, not path itself.
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}

		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !slices.ContainsFunc(trusted, sid.Equals) {
			return fmt.Errorf("secret %s %s must only be accessible by you, but grants access to %s", kind, path, accountName(sid))
		}
	}

	return nil
}

func accountName(sid *windows.SID) string {
	account, domain, _, err := sid.LookupAccount("")
	if err != nil {
		return sid.String()
	}
	if domain == "" {
		return account
	}
	return domain + `\` + account
}
