//go:build !windows && !linux

package system

// detectTopology ne sait rien relever sur les autres systèmes : la topologie reste vide et H-008
// se déclare non concluante plutôt que de supposer une taille de cache.
func detectTopology() Topology { return Topology{} }
