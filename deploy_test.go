package main

import (
	"errors"
	"testing"
)

func TestCommitProjectNeedsVMEndpoint(t *testing.T) {
	if !commitProjectNeedsVMEndpoint(errors.New("Taikun Error: (TITLE Bad request) (DETAIL You need at least one worker, an odd number of master(s) and one bastion to commit changes.)")) {
		t.Fatal("expected VM commit fallback for Kubernetes layout validation error")
	}

	if commitProjectNeedsVMEndpoint(errors.New("some other error")) {
		t.Fatal("did not expect VM commit fallback for unrelated errors")
	}
}
