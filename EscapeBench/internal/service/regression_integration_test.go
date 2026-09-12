//go:build integration_test

// Ce fichier est le harnais de non-régression des verdicts publiés (UC-005). Il ne tourne que
// sous le tag `integration_test` : il lit `results/` sur le disque réel, en lecture seule, au
// lieu de travailler sur des fakes en mémoire.
//
// Il existe parce qu'aucun test ne rejouait les campagnes archivées : rien ne garantissait qu'un
// correctif aux évaluateurs, au store ou au modèle laisse inchangés les verdicts déjà publiés au
// tableau de bord et au rapport final. Toute divergence échoue en nommant l'hypothèse et la
// campagne.
//
// Le dépôt ne versionne pas `matrices/` (sources générées, reproductibles depuis les paramètres :
// BR-001-1, C-007). Les matrices citées par les campagnes archivées sont donc reconstruites à
// partir des spécifications de paramètres consignées dans LANCEMENT.md, et retenues seulement si
// leur identifiant canonique est bien celui que la campagne nomme — la reconstruction se prouve
// elle-même. Les trois évaluateurs qui lisent la Matrix sont sautés, en le disant, pour les deux
// matrices dont les paramètres ne sont consignés nulle part.
package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/adapters/store"
	"github.com/agbruneau/escapebench/internal/models"
)

// archivedParameters sont les paramètres des matrices des campagnes archivées, tels que
// LANCEMENT.md les consigne. Ils sont écrits ici en littéraux du modèle plutôt que passés par
// cli.ParseParameters : internal/adapters/cli importe internal/service, et le sens inverse serait
// un cycle. Un écart de traduction est sans effet sur ce que le harnais prouve — une matrice
// reconstruite n'est retenue que si son identifiant canonique est celui que la campagne nomme.
func archivedParameters() []models.MatrixParameters {
	const (
		kiB = 1024
		miB = 1024 * kiB
	)
	return []models.MatrixParameters{
		models.ReferenceParameters(),
		{
			Sizes:                []int{8, 16, 24, 128, 1024},
			PointerFieldVariants: []bool{false, true},
			Profiles: []models.LifetimeProfile{
				models.ProfileLocal, models.ProfileReturned, models.ProfileCapturedByClosure,
				models.ProfileSentOnChannel, models.ProfileStoredInMap, models.ProfileStoredInSlice,
				models.ProfileStoredInStruct, models.ProfileReturnedAlloc,
			},
			PassingModes: models.PassingModes(),
			Layouts:      []models.Layout{models.LayoutArrayFill, models.LayoutNamedFields, models.LayoutNamedFieldsSham},
			Repeats:      []int{1, 4, 16},
			Payloads:     []int{1, 2},
			Replicates:   models.DefaultReplicates(),
			Probes: []models.ProbeSpec{
				{Kind: models.ProbeSequentialScan, Parameter: 32 * miB},
				{Kind: models.ProbeSequentialScan, Parameter: 128 * miB},
				{Kind: models.ProbeScatteredScan, Parameter: 32 * miB},
				{Kind: models.ProbeScatteredScan, Parameter: 128 * miB},
				{Kind: models.ProbeAppendPrealloc, Parameter: 100000},
				{Kind: models.ProbeAppendGrow, Parameter: 100000},
				{Kind: models.ProbePointerChase, Parameter: 16 * kiB},
				{Kind: models.ProbePointerChase, Parameter: 256 * kiB},
				{Kind: models.ProbePointerChase, Parameter: 256 * miB},
			},
		},
		{
			Sizes:                []int{8},
			PointerFieldVariants: []bool{false},
			Profiles:             []models.LifetimeProfile{models.ProfileLocal},
			PassingModes:         models.PassingModes(),
			Layouts:              []models.Layout{models.LayoutNamedFields},
			Repeats:              models.DefaultRepeats(),
			Payloads:             models.DefaultPayloads(),
			Replicates:           models.DefaultReplicates(),
			Probes: []models.ProbeSpec{
				{Kind: models.ProbePointerChase, Parameter: 16384},
				{Kind: models.ProbePointerChase, Parameter: 262144},
				{Kind: models.ProbePointerChase, Parameter: 268435456},
			},
		},
		{
			Sizes:                []int{8, 16, 24, 128},
			PointerFieldVariants: []bool{false, true},
			Profiles:             []models.LifetimeProfile{models.ProfileLocal},
			PassingModes:         models.PassingModes(),
			Layouts:              []models.Layout{models.LayoutNamedFields},
			Repeats:              models.DefaultRepeats(),
			Payloads:             models.DefaultPayloads(),
			Replicates:           models.ReplicateCount,
			Probes: []models.ProbeSpec{
				{Kind: models.ProbeAppendPrealloc, Parameter: 100000},
				{Kind: models.ProbeAppendGrow, Parameter: 100000},
			},
		},
	}
}

// noEvaluatorRationalePrefix est le motif que UC-005 inscrit quand aucun évaluateur mécanique
// n'existe pour une hypothèse (verdict.go, evaluate).
const noEvaluatorRationalePrefix = "aucun évaluateur mécanique n'est implémenté pour "

// matrixDependent nomme les hypothèses dont l'évaluateur lit Evidence.Matrix. Elles ne sont
// rejouées que si la matrice de la campagne a pu être reconstruite.
var matrixDependent = map[string]bool{"H-003": true, "H-009": true, "H-010": true}

// projectRoot remonte jusqu'au répertoire qui porte go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("racine du module introuvable (go.mod)")
		}
		dir = parent
	}
}

// placeholderDigest tient lieu d'empreinte du harnais pour une matrice reconstruite : le modèle
// l'exige non vide (C-005) mais elle n'entre ni dans les cellules ni dans l'identifiant canonique.
const placeholderDigest = "0000000000000000000000000000000000000000000000000000000000000000"

// rebuiltMatrices reconstruit les matrices connues, indexées par identifiant canonique.
func rebuiltMatrices(t *testing.T) map[string]models.Matrix {
	t.Helper()
	out := map[string]models.Matrix{}
	for _, params := range archivedParameters() {
		// L'empreinte du harnais ne participe ni aux cellules ni à l'identifiant : une valeur
		// quelconque mais valide suffit à la reconstruction.
		matrix, err := models.NewMatrix(params, placeholderDigest, time.Unix(0, 0).UTC())
		if err != nil {
			t.Fatalf("reconstruction d'une matrice archivée : %v", err)
		}
		out[matrix.ID] = matrix
	}
	return out
}

// TestUC005_NonRegressionVerdictsArchives rejoue les évaluateurs gelés sur chaque campagne
// archivée et exige que les verdicts publiés soient reproduits à l'identique (UC-005, BR-005-1).
func TestUC005_NonRegressionVerdictsArchives(t *testing.T) {
	ctx := context.Background()
	root := projectRoot(t)
	disk := store.New(root)
	matrices := rebuiltMatrices(t)

	reports, err := disk.VerdictReports(ctx)
	if err != nil {
		t.Fatalf("lecture de results/verdicts : %v", err)
	}
	if len(reports) == 0 {
		t.Fatal("aucun rapport de verdicts archivé : le harnais de non-régression ne prouve rien")
	}

	replayed, skipped, historical := 0, 0, 0
	for _, report := range reports {
		campaign, err := disk.LoadCampaign(ctx, report.CampaignID)
		if err != nil {
			t.Fatalf("campagne %s : %v", report.CampaignID, err)
		}
		evidence, err := archivedEvidence(ctx, disk, campaign, matrices)
		if err != nil {
			t.Fatalf("preuves de %s : %v", campaign.ID, err)
		}
		_, matrixKnown := matrices[campaign.MatrixID]

		for _, archived := range report.Verdicts {
			evaluate, ok := evaluators[archived.HypothesisID]
			if !ok {
				t.Errorf("%s / %s : aucun évaluateur alors qu'un verdict est archivé",
					campaign.ID, archived.HypothesisID)
				continue
			}
			// Trois campagnes ont rendu un verdict sur une hypothèse dont l'évaluateur
			// mécanique n'existait pas encore (H-012 avant C-009, H-013 avant C-010). Ce
			// motif est la signature de cette histoire : le rejouer n'aurait aucun sens,
			// mais le verdict archivé doit être non concluant.
			if strings.HasPrefix(archived.Rationale, noEvaluatorRationalePrefix) {
				if archived.Outcome != models.OutcomeInconclusive {
					t.Errorf("%s / %s : verdict %s sans évaluateur à l'époque, INCONCLUSIVE attendu",
						campaign.ID, archived.HypothesisID, archived.Outcome)
				}
				historical++
				continue
			}
			if matrixDependent[archived.HypothesisID] && !matrixKnown {
				t.Logf("%s / %s : sauté, les paramètres de la matrice %s ne sont consignés nulle part",
					campaign.ID, archived.HypothesisID, campaign.MatrixID)
				skipped++
				continue
			}
			t.Run(campaign.ID+"/"+archived.HypothesisID, func(t *testing.T) {
				result := evaluate(evidence)
				if result.Outcome != archived.Outcome {
					t.Fatalf("verdict rejoué %s, archivé %s\nrationale rejoué : %s\nrationale archivé : %s",
						result.Outcome, archived.Outcome, result.Rationale, archived.Rationale)
				}
				if result.Rationale != archived.Rationale {
					t.Fatalf("le motif du verdict a changé\nrejoué  : %s\narchivé : %s",
						result.Rationale, archived.Rationale)
				}
			})
			replayed++
		}
	}

	// Garde contre une dégradation silencieuse du harnais : s'il cesse de rejouer quoi que ce
	// soit (résultats déplacés, store muet), il doit échouer plutôt que passer à vide.
	if replayed == 0 {
		t.Fatal("aucun verdict rejoué : le harnais de non-régression est devenu inopérant")
	}
	t.Logf("%d verdicts rejoués sur %d campagnes, %d sautés faute de matrice reconstructible, "+
		"%d rendus avant l'existence de leur évaluateur", replayed, len(reports), skipped, historical)
}

// archivedEvidence rassemble, en lecture seule, les preuves d'une campagne archivée. Elle suit la
// même règle de sélection que VerdictService.gather : dernier fichier de comparaison, dernier
// rapport d'échappement dont la toolchain est celle de la campagne.
func archivedEvidence(ctx context.Context, disk *store.Store, campaign models.Campaign,
	matrices map[string]models.Matrix) (Evidence, error) {
	evidence := Evidence{
		Campaign:         campaign,
		Matrix:           matrices[campaign.MatrixID],
		Measurements:     map[string]models.Measurement{},
		MeasurementPaths: map[string]string{},
	}
	measurements, err := disk.LoadMeasurements(ctx, campaign.ID)
	if err != nil {
		return Evidence{}, err
	}
	for _, m := range measurements {
		evidence.Measurements[m.SubjectID] = m
		evidence.MeasurementPaths[m.SubjectID] = disk.MeasurementPath(campaign.ID, m.SubjectID)
	}
	if set, path, err := disk.LatestComparisonSet(ctx, campaign.ID); err == nil {
		evidence.ComparisonSet = &set
		evidence.ComparisonPath = path
	}
	paths, err := disk.ListEscapeReports(ctx, campaign.MatrixID)
	if err != nil {
		return Evidence{}, err
	}
	for i := len(paths) - 1; i >= 0; i-- {
		report, err := disk.ReadEscapeReport(ctx, paths[i])
		if err != nil {
			return Evidence{}, err
		}
		if report.Provenance.SameToolchain(campaign.Provenance) {
			evidence.EscapeReport = &report
			evidence.EscapePath = paths[i]
			break
		}
	}
	return evidence, nil
}
