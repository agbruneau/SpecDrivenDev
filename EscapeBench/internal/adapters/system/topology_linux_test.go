//go:build linux

package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLinuxCacheSize(t *testing.T) {
	t.Parallel()
	cases := map[string]int64{
		"48K":      49152,
		"32k":      32768,
		"3M":       3 << 20,
		"36M":      36 << 20,
		"1G":       1 << 30,
		"1024":     1024,
		" 48K ":    49152,
		"":         0,
		"beaucoup": 0,
		"0K":       0,
		"-4K":      0,
	}
	for value, want := range cases {
		if got := parseLinuxCacheSize(value); got != want {
			t.Fatalf("parseLinuxCacheSize(%q) = %d, %d attendu", value, got, want)
		}
	}
}

func TestReadLinuxCaches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(index, level, kind, size string) {
		dir := filepath.Join(root, "cache", index)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir : %v", err)
		}
		for name, content := range map[string]string{"level": level, "type": kind, "size": size} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content+"\n"), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
		}
	}
	write("index0", "1", "Data", "48K")
	write("index1", "1", "Instruction", "32K")
	write("index3", "3", "Unified", "36M")
	// Une entrée sans niveau lisible est ignorée plutôt que devinée.
	if err := os.MkdirAll(filepath.Join(root, "cache", "index9"), 0o755); err != nil {
		t.Fatalf("mkdir : %v", err)
	}

	caches := readLinuxCaches(root)
	if len(caches) != 3 {
		t.Fatalf("%d instances, 3 attendues : %+v", len(caches), caches)
	}
	topology := topologyFrom(caches)
	if topology.L1DataCacheBytes != 49152 || topology.LastLevelCacheBytes != 36<<20 {
		t.Fatalf("topologie = %+v", topology)
	}
	if got := readLinuxCaches(filepath.Join(root, "absent")); got != nil {
		t.Fatalf("un répertoire absent doit rendre une liste vide : %+v", got)
	}
}
