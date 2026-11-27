package main

import "testing"

func TestModuleReleaseVersion(t *testing.T) {
	for input, want := range map[string]string{
		"v1.14.2": "1.14.2", "v1.15.0-rc.1": "1.15.0-rc.1",
		"(devel)": "", "v1.14.2+dirty": "", "v1.14.3-0.20260408065356-8bd4fa0add7e": "",
	} {
		if got := moduleReleaseVersion(input); got != want {
			t.Errorf("moduleReleaseVersion(%q) = %q, want %q", input, got, want)
		}
	}
}
