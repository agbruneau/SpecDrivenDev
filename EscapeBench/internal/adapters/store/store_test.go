package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/ports"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return New(t.TempDir())
}

func provenance() models.Provenance {
	return models.Provenance{GoVersion: "go1.25.0", GOOS: "linux", GOARCH: "amd64", CPUModel: "cpu",
		CapturedAt: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)}
}

func smallMatrix(t *testing.T) models.Matrix {
	t.Helper()
	params := models.MatrixParameters{
		Sizes:                []int{8, 24},
		PointerFieldVariants: []bool{false, true},
		Profiles:             []models.LifetimeProfile{models.ProfileLocal},
		PassingModes:         models.PassingModes(),
		Probes:               []models.ProbeSpec{{Kind: models.ProbeAppendGrow, Parameter: 1000}},
	}
	matrix, err := models.NewMatrix(params, "harness-digest", time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	return matrix
}

func TestMatrixRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	matrix := smallMatrix(t)

	exists, err := s.Exists(ctx, matrix.ID)
	if err != nil || exists {
		t.Fatalf("Exists = %v, %v", exists, err)
	}
	if err := s.WriteSources(ctx, matrix.ID, map[string]string{
		"go.mod":                     "module m\n",
		"subjects/a/subject.go":      "package subject\n",
		"subjects/a/subject_test.go": "package subject\n",
	}); err != nil {
		t.Fatalf("WriteSources : %v", err)
	}
	// Tant que matrix.json n'est pas écrit, la matrice est incomplète.
	exists, _ = s.Exists(ctx, matrix.ID)
	if exists {
		t.Fatal("une matrice sans matrix.json n'existe pas encore")
	}
	if err := s.Finalize(ctx, matrix); err != nil {
		t.Fatalf("Finalize : %v", err)
	}
	exists, _ = s.Exists(ctx, matrix.ID)
	if !exists {
		t.Fatal("la matrice devrait exister après Finalize")
	}
	loaded, err := s.Load(ctx, matrix.ID)
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if loaded.ID != matrix.ID || loaded.HarnessDigest != matrix.HarnessDigest {
		t.Fatalf("aller-retour altéré : %+v", loaded)
	}
	if len(loaded.Cells) != len(matrix.Cells) || len(loaded.Probes) != len(matrix.Probes) {
		t.Fatalf("%d cellules et %d sondes relues", len(loaded.Cells), len(loaded.Probes))
	}
	if loaded.Cells[0].ID() != matrix.Cells[0].ID() {
		t.Fatalf("identifiant de cellule altéré : %s", loaded.Cells[0].ID())
	}
	if err := loaded.Validate(); err != nil {
		t.Fatalf("la matrice relue doit rester valide : %v", err)
	}
	ids, err := s.List(ctx)
	if err != nil || len(ids) != 1 || ids[0] != matrix.ID {
		t.Fatalf("List = %v, %v", ids, err)
	}
}

func TestMatrixImmuable(t *testing.T) {
	t.Parallel()
	// BR-001-2 : une Matrix n'est jamais modifiée après génération.
	// Mutation : laisser Finalize réécrire matrix.json ⇒ échec attendu.
	ctx := context.Background()
	s := newStore(t)
	matrix := smallMatrix(t)
	if err := s.Finalize(ctx, matrix); err != nil {
		t.Fatalf("Finalize : %v", err)
	}
	if err := s.Finalize(ctx, matrix); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	if err := s.WriteSources(ctx, matrix.ID, map[string]string{"x.go": "package x"}); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	if err := s.Remove(ctx, matrix.ID); !errors.Is(err, ErrImmutable) {
		t.Fatalf("Remove ne doit pas toucher une matrice complète : %v", err)
	}
}

func TestMatrixFinalizeRefuseInvalide(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	if err := s.Finalize(context.Background(), models.Matrix{}); err == nil {
		t.Fatal("Finalize doit valider la matrice")
	}
}

func TestRemoveMatriceIncomplete(t *testing.T) {
	t.Parallel()
	// UC-001 A3 : aucun répertoire de matrice partiel ne subsiste.
	ctx := context.Background()
	s := newStore(t)
	if err := s.WriteSources(ctx, "M-part", map[string]string{"subjects/a/subject.go": "package subject\n"}); err != nil {
		t.Fatalf("WriteSources : %v", err)
	}
	if err := s.Remove(ctx, "M-part"); err != nil {
		t.Fatalf("Remove : %v", err)
	}
	if _, err := os.Stat(s.Dir("M-part")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("le répertoire partiel devrait avoir disparu")
	}
}

func TestLoadMatriceAbsente(t *testing.T) {
	t.Parallel()
	if _, err := newStore(t).Load(context.Background(), "M-inconnue"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erreur = %v, ErrNotFound attendue", err)
	}
}

func TestListSansRepertoire(t *testing.T) {
	t.Parallel()
	ids, err := newStore(t).List(context.Background())
	if err != nil || ids != nil {
		t.Fatalf("List = %v, %v", ids, err)
	}
}

func TestEscapeReportRoundTripEtImmutabilite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	report := models.EscapeReport{
		MatrixID:   "M-1",
		Provenance: provenance(),
		Verdicts: []models.EscapeVerdict{
			{CellID: "a", Escapes: true, Category: models.CategoryReturnPointer, CompilerReason: "moved to heap: t", Status: models.EscapeStatusOK},
			{CellID: "b", Category: models.CategoryNone, Status: models.EscapeStatusOK},
			{CellID: "c", Status: models.EscapeStatusCompileError, CompilerError: "boom"},
		},
	}
	path, err := s.WriteEscapeReport(ctx, report, provenance().CapturedAt)
	if err != nil {
		t.Fatalf("WriteEscapeReport : %v", err)
	}
	if path != "results/escape/M-1/20260910T120000Z.json" {
		t.Fatalf("chemin = %q", path)
	}
	// BR-002-3 : un fichier de verdicts n'est jamais réécrit.
	if _, err := s.WriteEscapeReport(ctx, report, provenance().CapturedAt); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	relu, err := s.ReadEscapeReport(ctx, path)
	if err != nil {
		t.Fatalf("ReadEscapeReport : %v", err)
	}
	if len(relu.Verdicts) != 3 || relu.Verdicts[2].CompilerError != "boom" {
		t.Fatalf("aller-retour altéré : %+v", relu)
	}
	if !relu.Provenance.SameToolchain(report.Provenance) {
		t.Fatal("la provenance doit survivre à l'aller-retour")
	}
	paths, err := s.ListEscapeReports(ctx, "M-1")
	if err != nil || len(paths) != 1 {
		t.Fatalf("ListEscapeReports = %v, %v", paths, err)
	}
	vide, err := s.ListEscapeReports(ctx, "M-inconnue")
	if err != nil || vide != nil {
		t.Fatalf("ListEscapeReports = %v, %v", vide, err)
	}
	if _, err := s.WriteEscapeReport(ctx, models.EscapeReport{MatrixID: "M-2"}, time.Now()); err == nil {
		t.Fatal("NFR-001 : une provenance incomplète doit être refusée")
	}
}

func TestCampaignCycleDeVie(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	day := time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC)

	id, err := s.NextCampaignID(ctx, day)
	if err != nil || id != "C-2026-09-10-1" {
		t.Fatalf("NextCampaignID = %q, %v", id, err)
	}
	campaign := models.Campaign{
		ID: id, MatrixID: "M-1", HarnessDigest: "h", HypothesesDigest: "d",
		HypothesisIDs: []string{"H-001"}, Count: 20, Status: models.CampaignRunning,
		Provenance: provenance(), StartedAt: day,
	}
	if err := s.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	if err := s.CreateCampaign(ctx, campaign); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	next, err := s.NextCampaignID(ctx, day)
	if err != nil || next != "C-2026-09-10-2" {
		t.Fatalf("NextCampaignID = %q, %v", next, err)
	}

	// BR-003-3, première exception : seul le statut de campaign.json change.
	finished := day.Add(time.Hour)
	if err := s.SetCampaignStatus(ctx, id, models.CampaignCompleted, finished, ""); err != nil {
		t.Fatalf("SetCampaignStatus : %v", err)
	}
	relue, err := s.LoadCampaign(ctx, id)
	if err != nil {
		t.Fatalf("LoadCampaign : %v", err)
	}
	if relue.Status != models.CampaignCompleted || !relue.FinishedAt.Equal(finished) {
		t.Fatalf("campagne = %+v", relue)
	}
	if relue.HarnessDigest != "h" || relue.HypothesesDigest != "d" || relue.Count != 20 {
		t.Fatal("BR-003-3 : aucun autre champ ne doit changer")
	}
	if len(relue.HypothesisIDs) != 1 || relue.HypothesisIDs[0] != "H-001" {
		t.Fatalf("hypothèses = %v", relue.HypothesisIDs)
	}
	if err := s.CreateCampaign(ctx, models.Campaign{}); err == nil {
		t.Fatal("CreateCampaign doit valider la campagne")
	}
	if _, err := s.LoadCampaign(ctx, "C-inconnue"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erreur = %v, ErrNotFound attendue", err)
	}
	if err := s.SetCampaignStatus(ctx, "C-inconnue", models.CampaignAborted, finished, "x"); err == nil {
		t.Fatal("SetCampaignStatus sur une campagne absente doit échouer")
	}
}

func TestMeasurementsImmuables(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	m := models.Measurement{
		CampaignID: "C-1", SubjectID: "probe/APPEND_GROW/1000", Status: models.MeasurementComplete,
		NsPerOp: []float64{1, 2}, BytesPerOp: []int64{8, 8}, AllocsPerOp: []int64{1, 1},
	}
	if err := s.WriteMeasurement(ctx, m); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
	// BR-003-3 : le runner crée des fichiers et n'en réécrit aucun.
	if err := s.WriteMeasurement(ctx, m); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	if got := s.MeasurementPath("C-1", m.SubjectID); got != "results/campaigns/C-1/measurements/probe_APPEND_GROW_1000.json" {
		t.Fatalf("MeasurementPath = %q", got)
	}
	failed := models.Measurement{CampaignID: "C-1", SubjectID: "a", Status: models.MeasurementFailed, FailureReason: "boom"}
	if err := s.WriteMeasurement(ctx, failed); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}
	all, err := s.LoadMeasurements(ctx, "C-1")
	if err != nil {
		t.Fatalf("LoadMeasurements : %v", err)
	}
	if len(all) != 2 || all[0].SubjectID != "a" {
		t.Fatalf("mesures = %+v", all)
	}
	if all[1].NsPerOp[1] != 2 {
		t.Fatalf("valeurs altérées : %+v", all[1])
	}
	vide, err := s.LoadMeasurements(ctx, "C-inconnue")
	if err != nil || vide != nil {
		t.Fatalf("LoadMeasurements = %v, %v", vide, err)
	}
	if got := s.CampaignPath("C-1"); got != "results/campaigns/C-1/campaign.json" {
		t.Fatalf("CampaignPath = %q", got)
	}
}

func TestComparisonSetRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	set := models.ComparisonSet{
		CampaignID: "C-1", MatrixID: "M-1", ComputedAt: provenance().CapturedAt, Method: "bootstrap",
		Comparisons: []models.Comparison{{
			CampaignID: "C-1", ValueCellID: "v", PointerCellID: "p", SizeBytes: 24,
			Profile: models.ProfileLocal, DeltaNsPerOp: -1.5, CILow: -2, CIHigh: -1, Significant: true,
			MedianValueNs: 10, MedianPointerNs: 8.5,
		}},
		TippingPoints: map[models.TippingKey]int{
			{Profile: models.ProfileLocal, HasPointerField: false}: 24,
			{Profile: models.ProfileLocal, HasPointerField: true}:  models.TippingNotObserved,
		},
		ExcludedPairs: []models.ExcludedPair{{ValueCellID: "x", PointerCellID: "y", Reason: "FAILED"}},
	}
	path, err := s.WriteComparisonSet(ctx, set)
	if err != nil {
		t.Fatalf("WriteComparisonSet : %v", err)
	}
	// BR-004-3 : un fichier de comparaison n'est jamais réécrit.
	if _, err := s.WriteComparisonSet(ctx, set); !errors.Is(err, ErrImmutable) {
		t.Fatalf("erreur = %v, ErrImmutable attendue", err)
	}
	relu, foundPath, err := s.LatestComparisonSet(ctx, "C-1")
	if err != nil {
		t.Fatalf("LatestComparisonSet : %v", err)
	}
	if foundPath != path {
		t.Fatalf("chemin = %q, attendu %q", foundPath, path)
	}
	if len(relu.Comparisons) != 1 || relu.Comparisons[0].DeltaNsPerOp != -1.5 {
		t.Fatalf("aller-retour altéré : %+v", relu)
	}
	if relu.TippingPoints[models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill, HasPointerField: false}] != 24 {
		t.Fatalf("point de bascule altéré : %+v", relu.TippingPoints)
	}
	if got := relu.TippingPoints[models.TippingKey{Profile: models.ProfileLocal, Layout: models.LayoutArrayFill, HasPointerField: true}]; got != models.TippingNotObserved {
		t.Fatalf("« non observé » doit survivre à l'aller-retour, obtenu %d", got)
	}
	if len(relu.ExcludedPairs) != 1 || relu.ExcludedPairs[0].Reason != "FAILED" {
		t.Fatalf("BR-004-1 : exclusions altérées : %+v", relu.ExcludedPairs)
	}
	if _, _, err := s.LatestComparisonSet(ctx, "C-inconnue"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erreur = %v, ErrNotFound attendue", err)
	}
	// Une campagne sans fichier de comparaison est signalée comme telle.
	if err := s.CreateCampaign(ctx, models.Campaign{ID: "C-2", MatrixID: "M", HarnessDigest: "h",
		HypothesesDigest: "d", Count: 20, Status: models.CampaignRunning, Provenance: provenance()}); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	if _, _, err := s.LatestComparisonSet(ctx, "C-2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erreur = %v, ErrNotFound attendue", err)
	}
}

func TestVerdictReportRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	if _, _, err := s.LatestVerdictReport(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatalf("erreur = %v, ErrNotFound attendue", err)
	}
	report := models.VerdictReport{
		CampaignID: "C-1", ProducedAt: provenance().CapturedAt,
		Verdicts: []models.Verdict{{HypothesisID: "H-001", CampaignID: "C-1",
			Outcome: models.OutcomeRefuted, Rationale: "r", ResultFiles: []string{"f"}}},
	}
	path, err := s.WriteVerdictReport(ctx, report)
	if err != nil {
		t.Fatalf("WriteVerdictReport : %v", err)
	}
	if _, err := s.WriteVerdictReport(ctx, report); !errors.Is(err, ErrImmutable) {
		t.Fatalf("NFR-004 : erreur = %v, ErrImmutable attendue", err)
	}
	plusRecent := report
	plusRecent.ProducedAt = report.ProducedAt.Add(time.Hour)
	plusRecent.Verdicts[0].Outcome = models.OutcomeConfirmed
	if _, err := s.WriteVerdictReport(ctx, plusRecent); err != nil {
		t.Fatalf("WriteVerdictReport : %v", err)
	}
	latest, latestPath, err := s.LatestVerdictReport(ctx)
	if err != nil {
		t.Fatalf("LatestVerdictReport : %v", err)
	}
	if latestPath == path {
		t.Fatal("le fichier le plus récent doit être retenu")
	}
	if latest.Verdicts[0].Outcome != models.OutcomeConfirmed {
		t.Fatalf("verdict = %+v", latest.Verdicts[0])
	}
}

func TestLock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newStore(t)
	held, err := s.LockHeld(ctx)
	if err != nil || held {
		t.Fatalf("LockHeld = %v, %v", held, err)
	}
	if err := s.AcquireLock(ctx, "C-1"); err != nil {
		t.Fatalf("AcquireLock : %v", err)
	}
	held, _ = s.LockHeld(ctx)
	if !held {
		t.Fatal("le verrou devrait être posé")
	}
	// Le verrou est exclusif : une seconde campagne ne peut pas démarrer.
	if err := s.AcquireLock(ctx, "C-2"); err == nil {
		t.Fatal("AcquireLock devrait refuser un second verrou")
	}
	content, err := os.ReadFile(filepath.Join(s.Root(), "results", LockName))
	if err != nil || string(content) != "C-1\n" {
		t.Fatalf("contenu du verrou = %q, %v", content, err)
	}
	if err := s.ReleaseLock(ctx); err != nil {
		t.Fatalf("ReleaseLock : %v", err)
	}
	// Le retrait d'un verrou absent n'est pas une erreur : la campagne peut se terminer deux fois.
	if err := s.ReleaseLock(ctx); err != nil {
		t.Fatalf("ReleaseLock : %v", err)
	}
}

func TestStampEtChemins(t *testing.T) {
	t.Parallel()
	if got := Stamp(time.Date(2026, 9, 10, 13, 30, 25, 0, time.UTC)); got != "20260910T133025Z" {
		t.Fatalf("Stamp = %q", got)
	}
	root := t.TempDir()
	s := New(root)
	if got := s.abs("results/x.json"); got != filepath.Join(root, "results", "x.json") {
		t.Fatalf("abs = %q", got)
	}
	// Un chemin déjà absolu est rendu tel quel, quelle que soit la plateforme.
	absolute := filepath.Join(t.TempDir(), "x.json")
	if got := s.abs(absolute); got != absolute {
		t.Fatalf("abs = %q, attendu %q", got, absolute)
	}
	if got := s.rel(filepath.Join(root, "results", "x.json")); got != "results/x.json" {
		t.Fatalf("rel = %q", got)
	}
	if got := stampOf("C-1-20260910T133025Z.json"); got != "20260910T133025Z" {
		t.Fatalf("stampOf = %q", got)
	}
	if got := stampOf("sansTiret.json"); got != "sansTiret" {
		t.Fatalf("stampOf = %q", got)
	}
}

func TestWriteFileAtomicNeLaissePasDeFichierPartiel(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sousrepertoire-absent", "x.json")
	if err := writeFileAtomic(path, []byte("x")); err == nil {
		t.Fatal("l'écriture dans un répertoire absent doit échouer")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir : %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("aucun fichier temporaire ne doit subsister : %v", entries)
	}
}

// TestUC003_A4_RepriseDuVerrouOrphelin verrouille A-044 : UC-003 A4 se déclenche précisément
// quand le processus n'est plus actif, donc quand son defer de libération n'a pas tourné. Le
// verrou orphelin interdisait la reprise de la campagne qu'il protège, et aucune sous-commande ne
// sait le retirer — le hook guard-paths l'interdit même à un agent.
func TestUC003_A4_RepriseDuVerrouOrphelin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := New(t.TempDir())
	if err := s.AcquireLock(ctx, "C-2026-09-11-1"); err != nil {
		t.Fatalf("AcquireLock : %v", err)
	}
	// Le processus meurt ici : ReleaseLock n'est jamais appelé.

	if err := s.AcquireLock(ctx, "C-2026-09-11-1"); err != nil {
		t.Fatalf("la reprise de la même campagne doit reprendre son verrou : %v", err)
	}
	err := s.AcquireLock(ctx, "C-2026-09-11-2")
	if !errors.Is(err, ports.ErrLockHeld) {
		t.Fatalf("erreur = %v, ports.ErrLockHeld attendue", err)
	}
	if !strings.Contains(err.Error(), "C-2026-09-11-1") {
		t.Fatalf("le refus doit nommer le détenteur : %v", err)
	}
	// Le démarrage d'une campagne neuve ne reprend rien : son identifiant n'est pas encore connu.
	if err := s.AcquireLock(ctx, ""); !errors.Is(err, ports.ErrLockHeld) {
		t.Fatalf("erreur = %v, ports.ErrLockHeld attendue", err)
	}
}

// TestUC003_BR3_NommageDuVerrou : le verrou est posé avant que l'identifiant soit dérivé (A-263),
// donc il est nommé ensuite. Sans ce nommage, A4 ne saurait plus reconnaître son propre verrou.
func TestUC003_BR3_NommageDuVerrou(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := New(t.TempDir())
	if err := s.AcquireLock(ctx, ""); err != nil {
		t.Fatalf("AcquireLock : %v", err)
	}
	if err := s.AdoptLock(ctx, "C-2026-09-11-1"); err != nil {
		t.Fatalf("AdoptLock : %v", err)
	}
	content, err := os.ReadFile(filepath.Join(s.Root(), "results", LockName))
	if err != nil {
		t.Fatalf("lecture du verrou : %v", err)
	}
	if strings.TrimSpace(string(content)) != "C-2026-09-11-1" {
		t.Fatalf("contenu du verrou = %q", content)
	}
	if err := s.AcquireLock(ctx, "C-2026-09-11-1"); err != nil {
		t.Fatalf("le verrou nommé doit être reconnu par sa campagne : %v", err)
	}
}
