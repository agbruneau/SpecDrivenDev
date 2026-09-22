//go:build windows

package system

import (
	"encoding/binary"
	"reflect"
	"testing"
)

// coreRecord construit un enregistrement SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX de relation
// donnée et de classe d'efficacité donnée, de la taille réelle d'un cœur à un groupe (48 octets).
func coreRecord(relation uint32, class byte) []byte {
	record := make([]byte, 48)
	binary.LittleEndian.PutUint32(record[0:4], relation)
	binary.LittleEndian.PutUint32(record[4:8], uint32(len(record)))
	record[9] = class
	return record
}

// Mutation : lire EfficiencyClass au premier octet (Flags) ⇒ échec attendu.
func TestUC003_Etape3_ParseWindowsCoreClasses(t *testing.T) {
	t.Parallel()
	var buffer []byte
	for i := 0; i < 2; i++ {
		buffer = append(buffer, coreRecord(relationProcessorCore, 1)...)
	}
	for i := 0; i < 3; i++ {
		buffer = append(buffer, coreRecord(relationProcessorCore, 0)...)
	}
	buffer = append(buffer, coreRecord(2, 7)...) // une relation de cache est ignorée
	if got := parseWindowsCoreClasses(buffer); !reflect.DeepEqual(got, map[int]int{1: 2, 0: 3}) {
		t.Fatalf("classes = %v", got)
	}
	if got := parseWindowsCoreClasses(buffer[:20]); len(got) != 0 {
		t.Fatalf("un enregistrement tronqué ne compte pas : %v", got)
	}
}

func TestUC003_Etape3_AffinityLabel(t *testing.T) {
	t.Parallel()
	if got := affinityLabel(0xffffff, 0xffffff); got != Unpinned {
		t.Fatalf("masque complet = %q", got)
	}
	if got := affinityLabel(0xff, 0xffffff); got != "0xff" {
		t.Fatalf("masque partiel = %q", got)
	}
}
