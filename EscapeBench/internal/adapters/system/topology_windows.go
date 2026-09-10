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

// recordSize est la taille d'un SYSTEM_LOGICAL_PROCESSOR_INFORMATION sur une plateforme 64 bits :
// un masque de processeurs, la relation, un remplissage, puis une union de seize octets.
const recordSize = 32

// parseWindowsCaches lit les enregistrements de cache d'un tampon rendu par
// GetLogicalProcessorInformation. La fonction est pure : elle est éprouvée sur un tampon construit
// à la main, sans appeler le système.
func parseWindowsCaches(buffer []byte) []CacheLevel {
	var out []CacheLevel
	for offset := 0; offset+recordSize <= len(buffer); offset += recordSize {
		record := buffer[offset : offset+recordSize]
		if binary.LittleEndian.Uint32(record[8:12]) != relationCache {
			continue
		}
		// CACHE_DESCRIPTOR occupe l'union : niveau, associativité, taille de ligne, taille, type.
		level := int(record[16])
		size := int64(binary.LittleEndian.Uint32(record[20:24]))
		kind := binary.LittleEndian.Uint32(record[24:28])
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
