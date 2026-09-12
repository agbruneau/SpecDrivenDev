package system

import (
	"context"
	"runtime"
	"time"
)

// Attestation de quiétude de la machine (C-010). Le critère de H-013 refuse de trancher quand la
// machine n'était pas au repos pendant la mesure, parce qu'une contention mémoire gonfle la latence
// non résidente bien plus que la latence résidente et produit une infirmation qui ne doit rien à
// l'hypothèse. La grandeur exposée est une fraction sans dimension : le travail processeur fait
// pendant la fenêtre par tout ce qui n'est pas le sujet mesuré, rapporté à la capacité de la
// machine sur la même fenêtre. Elle est donc transportable d'un boîtier à l'autre.
//
// Ce que cette attestation voit et ne voit pas est écrit dans C-010 et vaut d'être répété ici : un
// processus qui sature la bande passante mémoire occupe un cœur à plein et se voit donc dans la
// fraction, mais la fraction ne mesure pas la bande passante elle-même. C'est un indicateur
// nécessaire, non suffisant.

// CPUTimes est un relevé cumulé des temps processeur de la machine, tous cœurs confondus.
// Measured est faux quand la plateforme ne sait pas le produire : H-013 se déclare alors non
// concluante plutôt que de supposer une quiétude.
type CPUTimes struct {
	Measured bool
	// Busy est le temps processeur cumulé passé hors de la boucle d'inactivité.
	Busy time.Duration
	// Total est le temps processeur cumulé disponible, inactivité comprise.
	Total time.Duration
}

// QuietudeProbe relève les temps processeur de la machine.
type QuietudeProbe struct{}

// NewQuietudeProbe construit la sonde de quiétude.
func NewQuietudeProbe() QuietudeProbe { return QuietudeProbe{} }

// Sample relève les temps processeur courants. Un relevé non mesuré n'est pas une erreur : la
// plateforme peut simplement ne pas l'exposer. Le contexte est honoré avant la lecture, qui touche
// /proc/stat ou le noyau (A-130) ; une annulation rend un relevé non mesuré plutôt qu'une valeur
// prise trop tard, et H-013 se déclare alors non concluante.
func (QuietudeProbe) Sample(ctx context.Context) CPUTimes {
	if ctx.Err() != nil {
		return CPUTimes{}
	}
	return sampleCPUTimes()
}

// Occupancy rend la fraction d'occupation des cœurs non mesurés entre deux relevés, une fois
// retranché le temps processeur propre du sujet mesuré.
//
// Elle vaut zéro sur une machine au repos et tend vers un quand tous les cœurs travaillent pour
// autre chose que le sujet. Le résultat est borné à l'intervalle [0, 1] : un dépassement ne peut
// venir que d'un arrondi ou d'un compteur qui recule, jamais d'une charge réelle.
//
// Deux dénominateurs sont possibles et le choix compte. Rapporter au temps processeur total relevé
// par le système ferait dépendre la fraction de la précision de son compteur d'inactivité ; on
// rapporte donc au produit du nombre de cœurs par la durée écoulée, qui est la capacité réelle de
// la machine sur la fenêtre.
func Occupancy(before, after CPUTimes, own time.Duration, elapsed time.Duration, cpus int) (float64, bool) {
	if !before.Measured || !after.Measured || elapsed <= 0 || cpus <= 0 {
		return 0, false
	}
	busy := after.Busy - before.Busy
	if busy < 0 {
		return 0, false
	}
	// Le temps propre du sujet mesuré ne compte pas : c'est le travail que la campagne demande.
	other := busy - own
	if other < 0 {
		other = 0
	}
	capacity := time.Duration(cpus) * elapsed
	fraction := float64(other) / float64(capacity)
	switch {
	case fraction < 0:
		return 0, true
	case fraction > 1:
		return 1, true
	default:
		return fraction, true
	}
}

// CPUCount rend le nombre de processeurs logiques de la machine. Il ne dépend pas de GOMAXPROCS,
// que la campagne fixe à un pour le processus mesuré alors que la capacité de la machine, elle,
// ne change pas.
func CPUCount() int { return runtime.NumCPU() }
