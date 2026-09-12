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

	// A4, étape 5 : la reprise ne porte aucun paramètre de mesure ; ceux de la Campaign valent.
	opts := CampaignOptions{Resume: campaign.ID}
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
		HypothesisIDs: []string{"H-001"}, Count: models.MinCount, Status: models.CampaignCompleted,
		Provenance: f.provenance.provenance, StartedAt: f.clock.Now(), FinishedAt: f.clock.Now(),
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
	running.FinishedAt = time.Time{}
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

// TestUC003_A4_SujetEnEchecNestPasRemesure verrouille A-020, le second constat bloquant de
// l'audit : la reprise ne considérait « faite » qu'une Measurement COMPLETE. Un sujet consigné
// FAILED par A3 — le cas normal, pas l'exception — était remesuré, et son écriture refusée par
// l'immutabilité de BR-003-3 : la campagne devenait irrécupérable.
func TestUC003_A4_SujetEnEchecNestPasRemesure(t *testing.T) {
	t.Parallel()
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
	echoue := f.matrix.SubjectIDs()[0]
	if err := f.store.WriteMeasurement(ctx, models.Measurement{
		CampaignID: campaign.ID, SubjectID: echoue, Status: models.MeasurementFailed,
		FailureReason: "panic pendant la mesure",
	}); err != nil {
		t.Fatalf("WriteMeasurement : %v", err)
	}

	report, err := f.service.Run(ctx, CampaignOptions{Resume: campaign.ID})
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	for _, id := range f.runner.order {
		if id == echoue {
			t.Fatalf("un sujet FAILED ne se remesure pas dans la même campagne : %v", f.runner.order)
		}
	}
	if report.Failed != 1 {
		t.Fatalf("FAILED = %d, 1 attendu : les mesures en échec antérieures sont comptées", report.Failed)
	}
	if report.Measured != len(f.matrix.SubjectIDs())-1 {
		t.Fatalf("mesurés = %d, %d attendus", report.Measured, len(f.matrix.SubjectIDs())-1)
	}
	if report.Status != models.CampaignCompleted {
		t.Fatalf("statut = %s, COMPLETED attendu", report.Status)
	}
}

// TestUC003_A4_RepriseSurAutreToolchain verrouille A-021 : Measurement ne porte pas de provenance
// propre, c'est Campaign.provenance qui atteste NFR-001 pour toutes ses mesures. Une reprise sous
// une autre toolchain rendrait cette attestation fausse sans erreur ni trace.
func TestUC003_A4_RepriseSurAutreToolchain(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*models.Provenance){
		"version de Go":        func(p *models.Provenance) { p.GoVersion = "go1.99.0" },
		"architecture":         func(p *models.Provenance) { p.GOARCH = "arm64" },
		"modèle de processeur": func(p *models.Provenance) { p.CPUModel = "autre processeur" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
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
			mutate(&f.provenance.provenance)

			_, err := f.service.Run(ctx, CampaignOptions{Resume: campaign.ID})
			if !errors.Is(err, ErrPrecondition) {
				t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
			}
			if len(f.runner.order) != 0 {
				t.Fatalf("aucune mesure ne doit être prise : %v", f.runner.order)
			}
			after, _ := f.store.LoadCampaign(ctx, campaign.ID)
			if after.Status != models.CampaignRunning {
				t.Fatalf("statut = %s, la campagne reste reprenable", after.Status)
			}
		})
	}
}

// TestUC003_A4_RepriseRefuseDrapeauxConcurrents verrouille A-030 : --matrix, --count et
// --hypotheses étaient ignorés en silence, laissant croire au chercheur qu'il étend ou redirige
// la campagne.
func TestUC003_A4_RepriseRefuseDrapeauxConcurrents(t *testing.T) {
	t.Parallel()
	cases := map[string]CampaignOptions{
		"matrice":     {Resume: "C-1", MatrixID: "M-autre"},
		"répétitions": {Resume: "C-1", Count: models.MinCount},
		"hypothèses":  {Resume: "C-1", HypothesisIDs: []string{"H-012"}},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newCampaignFixture(t)
			_, err := f.service.Run(context.Background(), opts)
			if !errors.Is(err, ErrPrecondition) {
				t.Fatalf("erreur = %v, ErrPrecondition attendue", err)
			}
		})
	}
}

// TestUC003_A4_RestitueLesParametresDeMesure verrouille A-265 : la reprise reprenait les drapeaux
// de la ligne de commande du moment, et non ceux sous lesquels les mesures déjà écrites ont été
// prises. Les deux tronçons d'une même campagne auraient alors des benchtime différents.
func TestUC003_A4_RestitueLesParametresDeMesure(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	ctx := context.Background()
	campaign := models.Campaign{
		ID: "C-2026-09-10-1", MatrixID: f.matrix.ID, HarnessDigest: "harness-v1",
		HypothesesDigest: "d", HypothesisIDs: []string{"H-001"}, Count: models.MinCount,
		BenchTime: "500ms", CPU: 4,
		Status: models.CampaignRunning, Provenance: f.provenance.provenance, StartedAt: f.clock.Now(),
	}
	if err := f.store.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("CreateCampaign : %v", err)
	}
	if _, err := f.service.Run(ctx, CampaignOptions{Resume: campaign.ID}); err != nil {
		t.Fatalf("Run : %v", err)
	}
	if f.runner.opts.BenchTime != "500ms" || f.runner.opts.CPU != 4 || f.runner.opts.Count != models.MinCount {
		t.Fatalf("options de mesure = %+v, celles de la Campaign attendues", f.runner.opts)
	}
}

// TestUC003_MainFlow_ParametresDeMesureConsignes : sans ces champs, A4 n'a rien à restituer.
func TestUC003_MainFlow_ParametresDeMesureConsignes(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	ctx := context.Background()
	report, err := f.service.Run(ctx, defaultOptions(f.matrix.ID))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	campaign, err := f.store.LoadCampaign(ctx, report.CampaignID)
	if err != nil {
		t.Fatalf("LoadCampaign : %v", err)
	}
	if campaign.BenchTime != "250ms" || campaign.CPU != 1 {
		t.Fatalf("paramètres de mesure = %q / %d", campaign.BenchTime, campaign.CPU)
	}
}

// TestUC003_A5_InterruptionLaisseLaCampagneReprenable verrouille A-261, le troisième constat
// bloquant : une interruption était consignée comme échec de sujet, la boucle continuait, tous
// les sujets restants échouaient en chaîne et la campagne se clôturait COMPLETED — rendant A4
// structurellement inatteignable.
func TestUC003_A5_InterruptionLaisseLaCampagneReprenable(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Interruption après le premier sujet, comme un Ctrl-C en cours de campagne.
	f.runner.onRun = func(string) {
		f.clock.advance(time.Second)
		cancel()
	}

	report, err := f.service.Run(ctx, defaultOptions(f.matrix.ID))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur = %v, context.Canceled attendue", err)
	}
	if len(f.runner.order) != 1 {
		t.Fatalf("%d sujets mesurés après l'interruption, 1 attendu : %v", len(f.runner.order), f.runner.order)
	}
	campaign, err := f.store.LoadCampaign(ctx, report.CampaignID)
	if err != nil {
		t.Fatalf("LoadCampaign : %v", err)
	}
	if campaign.Status != models.CampaignRunning {
		t.Fatalf("statut = %s, RUNNING attendu pour que A4 soit atteignable", campaign.Status)
	}
	measurements, _ := f.store.LoadMeasurements(ctx, report.CampaignID)
	for _, m := range measurements {
		if m.Status == models.MeasurementFailed {
			t.Fatalf("une interruption n'est pas un échec de mesure : %s consigné FAILED", m.SubjectID)
		}
	}
	if held, _ := f.store.LockHeld(ctx); held {
		t.Fatal("le verrou doit être libéré, sinon la reprise bute dessus")
	}
}

// TestUC003_BR3_VerrouPoseAvantLaCreation verrouille A-263 : le verrou était acquis dans la boucle
// de mesure, après que CreateCampaign a écrit campaign.json au statut RUNNING. Une campagne
// orpheline subsistait quand le verrou était refusé, et deux lancements simultanés dérivaient le
// même identifiant.
func TestUC003_BR3_VerrouPoseAvantLaCreation(t *testing.T) {
	t.Parallel()
	f := newCampaignFixture(t)
	ctx := context.Background()
	if err := f.store.AcquireLock(ctx, "C-autre"); err != nil {
		t.Fatalf("AcquireLock : %v", err)
	}

	_, err := f.service.Run(ctx, defaultOptions(f.matrix.ID))
	if err == nil {
		t.Fatal("une campagne concurrente doit être refusée")
	}
	if len(f.store.campaigns) != 0 {
		t.Fatalf("aucune campagne ne doit être créée : %v", f.store.campaigns)
	}
}

// TestUC003_VerrouLibereSurCheminsDErreur verrouille A-028 : rien n'assurait la libération du
// verrou quand la mesure échoue, et un verrou orphelin bloque ensuite UC-001 comme UC-003.
func TestUC003_VerrouLibereSurCheminsDErreur(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*campaignFixture){
		"runner en panne":     func(f *campaignFixture) { f.runner.err = errors.New("runner en panne") },
		"écriture impossible": func(f *campaignFixture) { f.store.writeErr = errors.New("disque plein") },
		"harnais modifié":     func(f *campaignFixture) { f.runner.onRun = func(string) { f.digester.digest = "harness-v2" } },
	}
	for name, breaks := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newCampaignFixture(t)
			breaks(f)
			if _, err := f.service.Run(context.Background(), defaultOptions(f.matrix.ID)); err == nil {
				t.Fatal("la campagne doit échouer")
			}
			if held, _ := f.store.LockHeld(context.Background()); held {
				t.Fatal("le verrou doit être libéré sur tout chemin d'erreur")
			}
		})
	}
}
