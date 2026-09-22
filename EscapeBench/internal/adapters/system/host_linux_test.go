//go:build linux

package system

import "testing"

func TestUC003_Etape3_LinuxAffinity(t *testing.T) {
	t.Parallel()
	status := "Name:\tx\nCpus_allowed_list:\t0-3\n"
	if got := linuxAffinity(procStatusField(status, "Cpus_allowed_list"), "0-3"); got != Unpinned {
		t.Fatalf("tous les processeurs = %q", got)
	}
	if got := linuxAffinity("0-1", "0-3"); got != "0-1" {
		t.Fatalf("épinglé = %q", got)
	}
	if got := linuxAffinity("", "0-3"); got != "" {
		t.Fatalf("illisible = %q", got)
	}
}
