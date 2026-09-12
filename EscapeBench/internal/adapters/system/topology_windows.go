//go:build windows

package system

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

// Relation et types de cache tels que les définit GetLogicalProcessorInformation.
const (
	relationCache    = 2
	cacheTypeUnified = 0
	cacheTypeData    = 2
)

// SYSTEM_LOGICAL_PROCESSOR_INFORMATION commence par un ULONG_PTR — le masque de processeurs —,
// suivi de la relation, d'un remplissage d'alignement, puis d'une union de seize octets.
//
// Révision du 2026-09-12 (A-066) : la taille et les décalages étaient écrits en dur pour une
// plateforme 64 bits, alors que la contrainte de build est `windows` sans restriction
// d'architecture. Sur windows/386 ou windows/arm, l'enregistrement fait 24 octets et le code
// lisait des champs décalés : les tailles de cache rendues étaient arbitraires, donc H-008 et
// H-013 rendaient un verdict sur des bandes de résidence fausses. Les décalages dérivent
// maintenant de la taille du pointeur.
const (
	ptrSize     = int(unsafe.Sizeof(uintptr(0)))
	relOffset   = ptrSize        // ULONG_PTR ProcessorMask
	unionOffset = 2 * ptrSize    // relation + remplissage d'alignement
	recordSize  = 2*ptrSize + 16 // union CACHE_DESCRIPTOR / autres
)

// parseWindowsCaches lit les enregistrements de cache d'un tampon rendu par
// GetLogicalProcessorInformation. La fonction est pure : elle est éprouvée sur un tampon construit
// à la main, sans appeler le système.
func parseWindowsCaches(buffer []byte) []CacheLevel {
	var out []CacheLevel
	for offset := 0; offset+recordSize <= len(buffer); offset += recordSize {
		record := buffer[offset : offset+recordSize]
		if binary.LittleEndian.Uint32(record[relOffset:relOffset+4]) != relationCache {
			continue
		}
		// CACHE_DESCRIPTOR occupe l'union : niveau, associativité, taille de ligne, taille, type.
		level := int(record[unionOffset])
		size := int64(binary.LittleEndian.Uint32(record[unionOffset+4 : unionOffset+8]))
		kind := binary.LittleEndian.Uint32(record[unionOffset+8 : unionOffset+12])
		out = append(out, CacheLevel{
			Level:     level,
			Data:      kind == cacheTypeData || kind == cacheTypeUnified,
			SizeBytes: size,
		})
	}
	return out
}

// detectTopology interroge le noyau. Une erreur rend une topologie vide : H-008 se déclare alors
// non concluante plutôt que de supposer une taille.
func detectTopology() Topology {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetLogicalProcessorInformation")
	var length uint32
	// Premier appel : obtenir la taille du tampon. Il échoue avec ERROR_INSUFFICIENT_BUFFER.
	proc.Call(0, uintptr(unsafe.Pointer(&length)))
	if length == 0 || length > 1<<20 {
		return Topology{}
	}
	buffer := make([]byte, length)
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&length)))
	if ret == 0 {
		return Topology{}
	}
	return topologyFrom(parseWindowsCaches(buffer[:length]))
}
