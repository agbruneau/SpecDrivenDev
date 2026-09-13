// Package system fournit l'horloge et la provenance des mesures (NFR-001). Le service ne fait que
// consigner ce qu'il reçoit : la détection du CPU et de la version de Go vit ici.
package system

import (
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// Clock rend l'heure système en UTC.
type Clock struct{}

// Now rend l'heure courante en UTC.
func (Clock) Now() time.Time { return time.Now().UTC() }

// FixedClock rend toujours la même heure ; utilisée par les tests.
type FixedClock struct{ Instant time.Time }

// Now rend l'heure figée.
func (c FixedClock) Now() time.Time { return c.Instant.UTC() }

// CPUReader lit le modèle de processeur ; injecté pour rendre la capture testable. Il prend un
// contexte parce qu'il touche le système de fichiers ou le registre (CLAUDE.md : « toute I/O prend
// un context.Context en premier paramètre », A-130).
type CPUReader func(ctx context.Context) string

// Prober capture la provenance d'une mesure.
type Prober struct {
	clock    interface{ Now() time.Time }
	cpu      CPUReader
	topology func() Topology
	pageSize func() int
	procs    func() int
	version  func() string
	goos     string
	goarch   string
}

// NewProber construit un Prober. Un lecteur de CPU nil retombe sur la détection système.
func NewProber(clock interface{ Now() time.Time }, cpu CPUReader) *Prober {
	if clock == nil {
		clock = Clock{}
	}
	if cpu == nil {
		cpu = DetectCPUModel
	}
	return &Prober{
		clock:    clock,
		cpu:      cpu,
		topology: detectTopology,
		pageSize: os.Getpagesize,
		procs:    func() int { return runtime.GOMAXPROCS(0) },
		version:  runtime.Version,
		goos:     runtime.GOOS,
		goarch:   runtime.GOARCH,
	}
}

// Capture rend la provenance courante. Aucun champ obligatoire n'est laissé vide : un résultat
// sans provenance complète est invalide (NFR-001). Les trois tailles mémoire ajoutées par C-008
// valent zéro quand la machine ne les expose pas ; une hypothèse qui en dépend se déclare alors
// non concluante plutôt que de supposer une valeur.
func (p *Prober) Capture(ctx context.Context) (models.Provenance, error) {
	cpu := strings.TrimSpace(p.cpu(ctx))
	if cpu == "" {
		cpu = "inconnu"
	}
	topology := p.topology()
	provenance := models.Provenance{
		GoVersion:           p.version(),
		GOOS:                p.goos,
		GOARCH:              p.goarch,
		CPUModel:            cpu,
		L1DataCacheBytes:    topology.L1DataCacheBytes,
		LastLevelCacheBytes: topology.LastLevelCacheBytes,
		PageSizeBytes:       int64(p.pageSize()),
		GOMAXPROCS:          p.procs(),
		CapturedAt:          p.clock.Now().UTC(),
	}
	return provenance, provenance.Validate()
}

// DetectCPUModel lit le modèle de processeur de la machine (NFR-001). La source est propre à la
// plateforme ; quand aucune ne répond, l'architecture est consignée avec la mention que le modèle
// n'a pas été détecté, plutôt qu'une valeur inventée.
func DetectCPUModel(ctx context.Context) string {
	if err := ctx.Err(); err != nil {
		return runtime.GOARCH + " (modèle non détecté)"
	}
	if model := platformCPUModel(); model != "" {
		return model
	}
	return runtime.GOARCH + " (modèle non détecté)"
}

// cpuModelFromCPUInfo extrait le champ « model name » d'un contenu /proc/cpuinfo.
func cpuModelFromCPUInfo(content string) string {
	for _, line := range strings.Split(content, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "model name", "Model name", "Hardware":
			return strings.TrimSpace(value)
		}
	}
	return ""
}
