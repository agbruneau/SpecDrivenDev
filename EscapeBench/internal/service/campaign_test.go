package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

type campaignFixture struct {
	service    *CampaignService
	store      *memoryStore
	runner     *fakeRunner
	digester   *fakeDigester
	provenance *fakeProvenance
	hypotheses *fakeHypotheses
	clock      *fakeClock
	matrix     models.Matrix
}

func newCampaignFixture(t *testing.T) *campaignFixture {
	t.Helper()
	f := &campaignFixture{
		store:      newStore(),
		runner:     &fakeRunner{measurements: map[string]models.Measurement{}},
		digester:   &fakeDigester{digest: "harness-v1"},
		provenance: newProvenance(),
		hypotheses: &fakeHypotheses{hypotheses: catalogue()},
		clock:      newClock(),
	}
	matrix, err := models.NewMatrix(smallParameters(), "harness-v1", f.clock.Now())
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	f.matrix = matrix
	f.store.matrices[matrix.ID] = matrix
	// La précondition de UC-003 exige des verdicts d'échappement pour la toolchain courante.
	f.store.escapes[matrix.ID] = []models.EscapeReport{{MatrixID: matrix.ID, Provenance: f.provenance.provenance}}
	f.store.escapePaths[matrix.ID] = []string{"results/escape/" + matrix.ID + "/0001.json"}
	f.service = NewCampaignService(f.store, f.store, f.store, f.runner, f.digester, f.provenance,
		f.hypotheses, digestOf, f.clock)
	return f
}

func defaultOptions(matrixID string) CampaignOptions {
	return CampaignOptions{MatrixID: matrixID, Count: models.MinCount, BenchTime: "250ms", CPU: 1}
}

func TestUC003_MainFlow(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	f.runner.onRun = func(string) { f.clock.advance(time.Second) }

	report, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	// Étape 8 : statut, décomptes et durée.
	if report.Status != models.CampaignCompleted {
		t.Fatalf("statut = %s", report.Status)
	}
	if report.CellCount != len(f.matrix.Cells) || report.ProbeCount != len(f.matrix.Probes) {
		t.Fatalf("décomptes = %+v", report)
	}
	if report.Measured != len(f.matrix.SubjectIDs()) || report.Failed != 0 {
		t.Fatalf("mesurés / FAILED = %d / %d", report.Measured, report.Failed)
	}
	if report.Duration <= 0 {
		t.Fatalf("durée = %v", report.Duration)
	}
	// Étape 3 : la Campaign porte provenance, empreinte du harnais et empreinte des critères.
	campaign, err := f.store.LoadCampaign(context.Background(), report.CampaignID)
	if err != nil {
		t.Fatalf("LoadCampaign : %v", err)
	}
	if campaign.HarnessDigest != "harness-v1" || campaign.HypothesesDigest == "" {
		t.Fatalf("campagne = %+v", campaign)
	}
	if len(campaign.HypothesisIDs) != len(catalogue()) {
		t.Fatalf("sans sélection, toutes les hypothèses sont couvertes : %v", campaign.HypothesisIDs)
	}
	// Étapes 5 et 6 : chaque sujet, Cell puis Probe, a sa Measurement écrite.
	measurements, _ := f.store.LoadMeasurements(context.Background(), report.CampaignID)
	if len(measurements) != len(f.matrix.SubjectIDs()) {
		t.Fatalf("%d mesures pour %d sujets", len(measurements), len(f.matrix.SubjectIDs()))
	}
	for _, m := range measurements {
		if err := m.Validate(models.MinCount); err != nil {
			t.Fatalf("NFR-003 : %v", err)
		}
	}
	// Les drapeaux de C-003 sont transmis au runner.
	if f.runner.opts.Count != models.MinCount || f.runner.opts.BenchTime != "250ms" || f.runner.opts.CPU != 1 {
		t.Fatalf("options de mesure = %+v", f.runner.opts)
	}
	// Les cellules sont mesurées avant les sondes (étape 5).
	if _, _, isProbe := models.ParseProbeID(f.runner.order[0]); isProbe {
		t.Fatalf("ordre de mesure = %v", f.runner.order)
	}
	// Le verrou est retiré à la fin de la campagne.
	if held, _ := f.store.LockHeld(context.Background()); held {
		t.Fatal("le verrou de campagne doit être libéré")
	}
}

func TestUC003_A1_RepetitionsInsuffisantes(t *testing.T) {
	t.Parallel()
	// A1 : le système refuse, affiche le minimum et ne crée aucune Campaign.
	// Mutation : accepter count < 20 ⇒ échec attendu.
	f := newCampaignFixture(t)
	opts := defaultOptions(f.matrix.ID)
	opts.Count = models.MinCount - 1
	_, err := f.service.Run(context.Background(), opts)
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
	if !strings.Contains(err.Error(), "20") {
		t.Fatalf("le minimum doit être affiché : %v", err)
	}
	if len(f.store.campaigns) != 0 {
		t.Fatal("A1 : aucune Campaign n'est créée")
	}
}

func TestUC003_A2_HarnaisModifiePendantLaCampagne(t *testing.T) {
	t.Parallel()
	// A2 : la campagne passe ABORTED, les mesures déjà écrites restent, le verdict est interdit.
	// Mutation : retirer la vérification finale d'empreinte ⇒ échec attendu.
	f := newCampaignFixture(t)
	f.runner.onRun = func(subjectID string) {
		if subjectID == f.matrix.SubjectIDs()[1] {
			f.digester.digest = "harness-v2"
		}
	}
	report, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
	if report.Status != models.CampaignAborted {
		t.Fatalf("statut = %s", report.Status)
	}
	if !strings.Contains(report.AbortReason, "harness-v1") || !strings.Contains(report.AbortReason, "harness-v2") {
		t.Fatalf("les deux empreintes doivent être consignées : %q", report.AbortReason)
	}
	campaign, _ := f.store.LoadCampaign(context.Background(), report.CampaignID)
	if campaign.Status != models.CampaignAborted {
		t.Fatalf("statut enregistré = %s", campaign.Status)
	}
	measurements, _ := f.store.LoadMeasurements(context.Background(), report.CampaignID)
	if len(measurements) == 0 {
		t.Fatal("A2 : les mesures déjà écrites sont conservées")
	}
}

func TestUC003_A3_SujetEnEchec(t *testing.T) {
	t.Parallel()
	// A3 : le sujet est consigné FAILED, la mesure se poursuit, la campagne reste COMPLETED.
	f := newCampaignFixture(t)
	fautif := f.matrix.SubjectIDs()[0]
	f.runner.measurements[fautif] = models.Measurement{
		SubjectID: fautif, Status: models.MeasurementFailed, FailureReason: "panic pendant la mesure",
	}
	report, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if report.Status != models.CampaignCompleted {
		t.Fatalf("statut = %s", report.Status)
	}
	if report.Failed != 1 || report.Measured != len(f.matrix.SubjectIDs())-1 {
		t.Fatalf("mesurés / FAILED = %d / %d", report.Measured, report.Failed)
	}
	measurements, _ := f.store.LoadMeasurements(context.Background(), report.CampaignID)
	if len(measurements) != len(f.matrix.SubjectIDs()) {
		t.Fatal("le sujet en échec est consigné, pas omis")
	}
}

func TestUC003_A3_MesureIncoherenteConsigneeCommeEchec(t *testing.T) {
	t.Parallel()
	// Une mesure qui ne porte pas exactement `count` valeurs viole NFR-003 : elle est consignée
	// FAILED plutôt qu'écrite telle quelle.
	f := newCampaignFixture(t)
	fautif := f.matrix.SubjectIDs()[0]
	f.runner.measurements[fautif] = completeMeasurementOf(fautif, models.MinCount-5, 1, 8, 1)
	report, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if report.Failed != 1 {
		t.Fatalf("%d sujets en échec, 1 attendu", report.Failed)
	}
	measurements, _ := f.store.LoadMeasurements(context.Background(), report.CampaignID)
	for _, m := range measurements {
		if m.SubjectID == fautif && (m.Status != models.MeasurementFailed || m.FailureReason == "") {
			t.Fatalf("mesure fautive = %+v", m)
		}
	}
}

func TestUC003_AucunSujetMesure(t *testing.T) {
	t.Parallel()
	// La campagne n'est COMPLETED que si au moins un sujet a été mesuré (étape 8).
	f := newCampaignFixture(t)
	for _, id := range f.matrix.SubjectIDs() {
		f.runner.measurements[id] = models.Measurement{SubjectID: id, Status: models.MeasurementFailed, FailureReason: "boom"}
	}
	report, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if err == nil {
		t.Fatal("Run aurait dû échouer")
	}
	if report.Status != models.CampaignAborted {
		t.Fatalf("statut = %s", report.Status)
	}
}

func TestUC003_A4_RepriseApresInterruption(t *testing.T) {
	t.Parallel()
	// A4 : la reprise vérifie l'empreinte, repart au premier sujet sans mesure complète et
	// n'écrit pas deux fois la même mesure.
	f := newCampaignFixture(t)
	ctx := context.Background()
	campaign := models.Campaign{
		ID: "C-2026-09-10-1", MatrixID: f.matrix.ID, HarnessDigest: "harness-v1",
		HypothesesDigest: "d", HypothesisIDs: []string{"H-001"}, Count: models.MinCount,
		Status: models.CampaignRunning, Provenance: f.provenance.provenance, StartedAt: f.clock.Now(),
	}
	if err := f.store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	deja := f.matrix.SubjectIDs()[0]
	mesure := completeMeasurementOf(deja, models.MinCount, 1, 8, 1)
	mesure.CampaignID = campaign.ID
	if err := f.store.WriteMeasurement(ctx, mesure); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}

	opts := CampaignOptions{Resume: campaign.ID, BenchTime: "250ms", CPU: 1}
	report, err := f.service.Run(ctx, opts)
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if !report.Resumed {
		t.Fatal("le rapport doit signaler une reprise")
	}
	if report.Measured != len(f.matrix.SubjectIDs()) {
		t.Fatalf("%d sujets mesurés, %d attendus", report.Measured, len(f.matrix.SubjectIDs()))
	}
	for _, id := range f.runner.order {
		if id == deja {
			t.Fatalf("le sujet déjà mesuré ne doit pas être remesuré : %v", f.runner.order)
		}
	}
	if report.Status != models.CampaignCompleted {
		t.Fatalf("statut = %s", report.Status)
	}
}

func TestUC003_A4_RepriseRefusee(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	ctx := context.Background()
	base := models.Campaign{
		ID: "C-1", MatrixID: f.matrix.ID, HarnessDigest: "harness-v1", HypothesesDigest: "d",
		Count: models.MinCount, Status: models.CampaignCompleted, Provenance: f.provenance.provenance,
	}
	if err := f.store.CreateCampaign(ctx, base); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	// Une campagne qui n'est pas RUNNING ne se reprend pas.
	if _, err := f.service.Run(ctx, CampaignOptions{Resume: "C-1"}); !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
	// Une campagne inconnue non plus.
	if _, err := f.service.Run(ctx, CampaignOptions{Resume: "C-inconnue"}); err == nil {
		t.Fatal("Run aurait dû échouer")
	}
	// Un harnais différent interdit la reprise (BR-003-1).
	running := base
	running.ID = "C-2"
	running.Status = models.CampaignRunning
	if err := f.store.CreateCampaign(ctx, running); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	f.digester.digest = "harness-v2"
	if _, err := f.service.Run(ctx, CampaignOptions{Resume: "C-2"}); !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
}

func TestUC003_BR5_CriteresGelesALaCreation(t *testing.T) {
	t.Parallel()
	// BR-003-5 : l'empreinte des critères est calculée à la création et ne dépend que des
	// hypothèses demandées. Mutation : calculer l'empreinte sur tout le catalogue ⇒ échec attendu.
	f := newCampaignFixture(t)
	opts := defaultOptions(f.matrix.ID)
	opts.HypothesisIDs = []string{"H-005", "H-001"}
	report, err := f.service.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if len(report.HypothesisIDs) != 2 || report.HypothesisIDs[0] != "H-001" {
		t.Fatalf("hypothèses = %v", report.HypothesisIDs)
	}
	attendu, err := digestOf(catalogue(), []string{"H-001", "H-005"})
	if err != nil {
		t.Fatalf("digestOf : %v", err)
	}
	if report.HypothesesDigest != attendu {
		t.Fatalf("empreinte = %q, attendue %q", report.HypothesesDigest, attendu)
	}
	complet, _ := digestOf(catalogue(), []string{"H-001", "H-002", "H-003", "H-004", "H-005", "H-006"})
	if report.HypothesesDigest == complet {
		t.Fatal("une sélection partielle ne doit pas produire l'empreinte du catalogue complet")
	}
}

func TestUC003_PreconditionVerdictsDEchappement(t *testing.T) {
	t.Parallel()
	// Précondition : les verdicts d'échappement existent pour la toolchain courante, dès lors
	// que la matrice contient au moins une Cell.
	f := newCampaignFixture(t)
	f.store.escapes = map[string][]models.EscapeReport{}
	f.store.escapePaths = map[string][]string{}
	_, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
	if !strings.Contains(err.Error(), "escapebench escape") {
		t.Fatalf("le message doit indiquer la sous-commande à exécuter : %v", err)
	}
}

func TestUC003_PreconditionInapplicableAuxSondesSeules(t *testing.T) {
	t.Parallel()
	// UC-002 ne classe jamais les Probe : une matrice de sondes seules n'a pas besoin de
	// verdicts d'échappement. Mutation : exiger les verdicts sans condition ⇒ échec attendu.
	f := newCampaignFixture(t)
	f.store.escapes = map[string][]models.EscapeReport{}
	f.store.escapePaths = map[string][]string{}
	sondesSeules := f.matrix
	sondesSeules.ID = "M-probes"
	sondesSeules.Cells = nil
	f.store.matrices[sondesSeules.ID] = sondesSeules
	report, err := f.service.Run(context.Background(), defaultOptions(sondesSeules.ID))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if report.Status != models.CampaignCompleted || report.CellCount != 0 {
		t.Fatalf("rapport = %+v", report)
	}
}

func TestUC003_VerdictsDEchappementDUneAutreToolchain(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	autre := f.provenance.provenance
	autre.GOARCH = "arm64"
	f.store.escapes[f.matrix.ID] = []models.EscapeReport{{MatrixID: f.matrix.ID, Provenance: autre}}
	_, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID))
	if !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}

func TestUC003_HarnaisDifferentDeLaMatrice(t *testing.T) {
	t.Parallel()
	// Étape 2 : l'empreinte du harnais doit être celle de la Matrix.
	f := newCampaignFixture(t)
	f.digester.digest = "harness-v2"
	if _, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID)); !errors.Is(err, ErrHarnessChanged) {
		t.Fatalf("erreur = %v, ErrHarnessChanged attendue", err)
	}
	if len(f.store.campaigns) != 0 {
		t.Fatal("aucune Campaign ne doit être créée")
	}
}

func TestUC003_VerrouExclusif(t *testing.T) {
	t.Parallel()
	// Une seule campagne à la fois : le verrou est la garde de C-005 pendant la mesure.
	f := newCampaignFixture(t)
	f.store.locked = true
	f.store.lockedBy = "C-autre"
	if _, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID)); err == nil {
		t.Fatal("Run aurait dû refuser une seconde campagne")
	}
}

func TestUC003_ErreursDesPorts(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := map[string]func(*campaignFixture){
		"matrice illisible":     func(f *campaignFixture) { delete(f.store.matrices, f.matrix.ID) },
		"provenance illisible":  func(f *campaignFixture) { f.provenance.err = boom },
		"empreinte illisible":   func(f *campaignFixture) { f.digester.err = boom },
		"hypothèses illisibles": func(f *campaignFixture) { f.hypotheses.err = boom },
		"runner en panne":       func(f *campaignFixture) { f.runner.err = boom },
		"écriture impossible":   func(f *campaignFixture) { f.store.writeErr = boom },
	}
	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newCampaignFixture(t)
			breakIt(f)
			if _, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID)); err == nil {
				t.Fatal("Run aurait dû échouer")
			}
		})
	}
}

func TestUC003_HypotheseInconnue(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	opts := defaultOptions(f.matrix.ID)
	opts.HypothesisIDs = []string{"H-999"}
	if _, err := f.service.Run(context.Background(), opts); err == nil {
		t.Fatal("une hypothèse absente du catalogue doit être refusée")
	}
}

func TestUC003_MatriceVide(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	vide := f.matrix
	vide.ID = "M-vide"
	vide.Cells = nil
	vide.Probes = nil
	f.store.matrices[vide.ID] = vide
	if _, err := f.service.Run(context.Background(), defaultOptions(vide.ID)); !errors.Is(err, ErrPrecondition) {
		t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
	}
}
