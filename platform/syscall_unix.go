// +build !windows

package platform

import "errors"

var ErrNotImplemented = errors.New("not implemented")

func createUserProfile(username string) error {
	return ErrNotImplemented
}

func deleteUserProfile(username string) error {
	return ErrNotImplemented
}

func userHomeDirectory(username string) (string, error) {
	return "", ErrNotImplemented
}

func localAccountNames() ([]string, error) {
	return nil, ErrNotImplemented
}

func isServiceRunning(_ string) error {
	return ErrNotImplemented
}

func areSSHServicesRunning() error {
	return ErrNotImplemented
}