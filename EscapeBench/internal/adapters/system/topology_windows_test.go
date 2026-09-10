//go:build windows

package system

import (
	"encoding/binary"
	"testing"
)

// record construit un SYSTEM_LOGICAL_PROCESSOR_INFORMATION synthétique.
func record(relationship uint32, level byte, size uint32, kind uint32) []byte {
	buffer := make([]byte, recordSize)
	binary.LittleEndian.PutUint64(buffer[0:8], 0xFF) // masque de processeurs
	binary.LittleEndian.PutUint32(buffer[8:12], relationship)
	buffer[16] = level
	binary.LittleEndian.PutUint32(buffer[20:24], size)
	binary.LittleEndian.PutUint32(buffer[24:28], kind)
	return buffer
}

func TestParseWindowsCaches(t *testing.T) {
	t.Parallel()
	var buffer []byte
	buffer = append(buffer, record(0, 0, 0, 0)...)                             // RelationProcessorCore : ignoré
	buffer = append(buffer, record(relationCache, 1, 49152, cacheTypeData)...) // L1 de données
	buffer = append(buffer, record(relationCache, 1, 32768, 1)...)             // L1 d'instructions
	buffer = append(buffer, record(relationCache, 3, 36<<20, cacheTypeUnified)...)
	buffer = append(buffer, []byte{1, 2, 3}...) // fin de tampon incomplète : ignorée

	caches := parseWindowsCaches(buffer)
	if len(caches) != 3 {
		t.Fatalf("%d instances de cache, 3 attendues : %+v", len(caches), caches)
	}
	if !caches[0].Data || caches[0].SizeBytes != 49152 || caches[0].Level != 1 {
		t.Fatalf("première instance = %+v", caches[0])
	}
	// Un cache d'instructions ne sert pas une hypothèse qui compare des accès mémoire.
	// Mutation : compter le type 1 comme donnée ⇒ échec attendu.
	if caches[1].Data {
		t.Fatalf("le cache d'instructions ne doit pas compter comme donnée : %+v", caches[1])
	}
	if !caches[2].Data || caches[2].Level != 3 {
		t.Fatalf("troisième instance = %+v", caches[2])
	}
	topology := topologyFrom(caches)
	if topology.L1DataCacheBytes != 49152 || topology.LastLevelCacheBytes != 36<<20 {
		t.Fatalf("topologie = %+v", topology)
	}
}

func TestParseWindowsCachesTamponVide(t *testing.T) {
	t.Parallel()
	if caches := parseWindowsCaches(nil); caches != nil {
		t.Fatalf("tampon vide = %+v", caches)
	}
	if caches := parseWindowsCaches(make([]byte, recordSize-1)); caches != nil {
		t.Fatalf("tampon tronqué = %+v", caches)
	}
}
