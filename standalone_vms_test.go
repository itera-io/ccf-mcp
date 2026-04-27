package main

import (
	"fmt"
	"testing"

	taikuncore "github.com/itera-io/taikungoclient/client"
)

func TestApplyCreateStandaloneVMDefaultsSetsVolumeSizeWhenOmitted(t *testing.T) {
	command := taikuncore.NewCreateStandAloneVmCommand()
	command.SetImage("Ubuntu 20.04 LTS")

	defaulted, hint := applyCreateStandaloneVMDefaults(command)

	if !defaulted {
		t.Fatal("expected volume size to be defaulted")
	}
	if got := command.GetVolumeSize(); got != defaultStandaloneVMVolumeSizeGiB {
		t.Fatalf("expected default volume size %d, got %d", defaultStandaloneVMVolumeSizeGiB, got)
	}
	if hint != "" {
		t.Fatalf("expected no Windows hint, got %q", hint)
	}
}

func TestApplyCreateStandaloneVMDefaultsPreservesExplicitVolumeSize(t *testing.T) {
	command := taikuncore.NewCreateStandAloneVmCommand()
	command.SetImage("Ubuntu 20.04 LTS")
	command.SetVolumeSize(25)

	defaulted, hint := applyCreateStandaloneVMDefaults(command)

	if defaulted {
		t.Fatal("expected explicit volume size to be preserved")
	}
	if got := command.GetVolumeSize(); got != 25 {
		t.Fatalf("expected explicit volume size 25, got %d", got)
	}
	if hint != "" {
		t.Fatalf("expected no Windows hint, got %q", hint)
	}
}

func TestApplyCreateStandaloneVMDefaultsAddsWindowsHint(t *testing.T) {
	command := taikuncore.NewCreateStandAloneVmCommand()
	command.SetImage("Windows Server 2022")

	defaulted, hint := applyCreateStandaloneVMDefaults(command)

	if !defaulted {
		t.Fatal("expected volume size to be defaulted")
	}
	if got := command.GetVolumeSize(); got != defaultStandaloneVMVolumeSizeGiB {
		t.Fatalf("expected default volume size %d, got %d", defaultStandaloneVMVolumeSizeGiB, got)
	}
	expectedHint := fmt.Sprintf("Windows images typically need a %d GiB root volume; consider setting volumeSize explicitly.", windowsStandaloneVMVolumeHintGiB)
	if hint != expectedHint {
		t.Fatalf("expected Windows hint %q, got %q", expectedHint, hint)
	}
}
