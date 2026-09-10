package service

import (
	"strings"
	"testing"
	"time"

	"github.com/agbruneau/escapebench/internal/models"
)

// comparisonOf construit une Comparison LOCAL prête à être évaluée par H-007.
func comparisonOf(size int, hasPointerField bool, layout models.Layout, delta, ciHigh float64, significant bool) models.Comparison {
	return models.Comparison{
		CampaignID: "C-1", ValueCellID: "v", PointerCellID: "p",
		SizeBytes: size, HasPointerField: hasPointerField, Layout: layout,
		Profile: models.ProfileLocal, DeltaNsPerOp: delta, CILow: ciHigh - 0.1, CIHigh: ciHigh,
		Significant: significant, MedianValueNs: 1.0,
	}
}

// h007Evidence assemble un jeu de comparaisons complet : les deux séries, les trois petites
// tailles, leurs témoins nuls et le témoin de sensibilité.
func h007Evidence(t *testing.T, mutate func(set *models.ComparisonSet)) Evidence {
	t.Helper()
	set := models.ComparisonSet{CampaignID: "C-1"}
	for _, hasPointerField := range []bool{false, true} {
		for _, size := range []int{8, 16, 24} {
			// Par défaut la valeur tient : aucun avantage au pointeur.
			set.Comparisons = append(set.Comparisons, comparisonOf(size, hasPointerField, models.LayoutNamedFields, 0.02, 0.05, false))
			// Témoin nul : un écart de 0,03 ns entre deux binaires.
			set.Comparisons = append(set.Comparisons, comparisonOf(size, hasPointerField, models.LayoutNamedFieldsSham, 0.03, 0.04, false))
		}
		// Témoin de sensibilité : au-delà des registres d'argument, le pointeur l'emporte.
		set.Comparisons = append(set.Comparisons, comparisonOf(96, hasPointerField, models.LayoutNamedFields, -3.0, -2.5, true))
	}
	if mutate != nil {
		mutate(&set)
	}
	return Evidence{
		Campaign:       models.Campaign{ID: "C-1"},
		ComparisonSet:  &set,
		ComparisonPath: "results/campaigns/C-1/comparison-x.json",
	}
}

// setComparison remplace la comparaison d'une taille, d'une série et d'une disposition données.
func setComparison(set *models.ComparisonSet, size int, hasPointerField bool, layout models.Layout, delta, ciHigh float64, significant bool) {
	for i, c := range set.Comparisons {
		if c.SizeBytes == size && c.HasPointerField == hasPointerField && c.Layout == layout {
			set.Comparisons[i] = comparisonOf(size, hasPointerField, layout, delta, ciHigh, significant)
			return
		}
	}
}

func TestUC005_H007(t *testing.T) {
	t.Parallel()
	t.Run("la valeur tient sur les trois petites tailles", func(t *testing.T) {
		t.Parallel()
		result := evaluateH007(h007Evidence(t, nil))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("une seule taille ne suffit pas", func(t *testing.T) {
		t.Parallel()
		result := evaluateH007(h007Evidence(t, func(set *models.ComparisonSet) {
			setComparison(set, 24, false, models.LayoutNamedFields, -1.0, -0.9, true)
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("deux tailles dans une série infirment", func(t *testing.T) {
		t.Parallel()
		// Mutation : exiger trois tailles sur trois ⇒ échec attendu.
		result := evaluateH007(h007Evidence(t, func(set *models.ComparisonSet) {
			setComparison(set, 16, false, models.LayoutNamedFields, -1.0, -0.9, true)
			setComparison(set, 24, false, models.LayoutNamedFields, -1.0, -0.9, true)
		}))
		if result.Outcome != models.OutcomeRefuted {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
		if !strings.Contains(result.Rationale, seriesLabel(false)) {
			t.Fatalf("le rationale doit nommer la série fautive : %q", result.Rationale)
		}
	})
	t.Run("un écart sous le plancher de bruit ne compte pas", func(t *testing.T) {
		t.Parallel()
		// Le témoin nul mesure 0,03 ns d'écart entre binaires ; un avantage de 0,02 ns est en
		// deçà et ne peut pas porter un verdict.
		// Mutation : retirer le plancher mesuré ⇒ échec attendu.
		result := evaluateH007(h007Evidence(t, func(set *models.ComparisonSet) {
			setComparison(set, 16, false, models.LayoutNamedFields, -0.02, -0.02, true)
			setComparison(set, 24, false, models.LayoutNamedFields, -0.02, -0.02, true)
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("la barrière relative s'applique quand le témoin est muet", func(t *testing.T) {
		t.Parallel()
		// Sans écart mesuré sur le témoin, la barrière relative de 10 % de la médiane prend le
		// relais : un avantage de 0,05 ns sur une médiane de 1 ns reste sous le seuil.
		result := evaluateH007(h007Evidence(t, func(set *models.ComparisonSet) {
			for i, c := range set.Comparisons {
				if c.Layout == models.LayoutNamedFieldsSham {
					set.Comparisons[i].DeltaNsPerOp = 0
				}
			}
			setComparison(set, 16, false, models.LayoutNamedFields, -0.05, -0.05, true)
			setComparison(set, 24, false, models.LayoutNamedFields, -0.05, -0.05, true)
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
}

func TestUC005_H007_Gardes(t *testing.T) {
	t.Parallel()
	cases := map[string]func(set *models.ComparisonSet){
		"témoin de sensibilité absent": func(set *models.ComparisonSet) {
			var kept []models.Comparison
			for _, c := range set.Comparisons {
				if c.SizeBytes < RegisterArgumentBytes {
					kept = append(kept, c)
				}
			}
			set.Comparisons = kept
		},
		"taille manquante en champs nommés": func(set *models.ComparisonSet) {
			var kept []models.Comparison
			for _, c := range set.Comparisons {
				if !(c.SizeBytes == 16 && !c.HasPointerField && c.Layout == models.LayoutNamedFields) {
					kept = append(kept, c)
				}
			}
			set.Comparisons = kept
		},
		"témoin nul manquant": func(set *models.ComparisonSet) {
			var kept []models.Comparison
			for _, c := range set.Comparisons {
				if !(c.SizeBytes == 8 && c.Layout == models.LayoutNamedFieldsSham) {
					kept = append(kept, c)
				}
			}
			set.Comparisons = kept
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := evaluateH007(h007Evidence(t, mutate))
			if result.Outcome != models.OutcomeInconclusive {
				t.Fatalf("verdict = %s, INCONCLUSIVE attendu — %s", result.Outcome, result.Rationale)
			}
		})
	}
	t.Run("sans fichier de comparaison", func(t *testing.T) {
		t.Parallel()
		if got := evaluateH007(Evidence{Campaign: models.Campaign{ID: "C-1"}}).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s", got)
		}
	})
}

// h008Evidence assemble une campagne dont la provenance porte les tailles de cache et deux sondes
// à chaîne dépendante, l'une résidente en L1, l'autre hors de tout cache.
func h008Evidence(l1, llc int64, residentNs, nonResidentNs float64) Evidence {
	const residentParam = 16384
	const nonResidentParam = 150994944
	resident := models.Probe{Kind: models.ProbePointerChase, Parameter: residentParam}.ID()
	nonResident := models.Probe{Kind: models.ProbePointerChase, Parameter: nonResidentParam}.ID()
	e := Evidence{
		Campaign: models.Campaign{ID: "C-1", Provenance: models.Provenance{
			GoVersion: "go1.27.0", GOOS: "windows", GOARCH: "amd64", CPUModel: "cpu",
			L1DataCacheBytes: l1, LastLevelCacheBytes: llc, CapturedAt: time.Unix(1, 0),
		}},
		Measurements:     map[string]models.Measurement{},
		MeasurementPaths: map[string]string{},
	}
	for id, ns := range map[string]float64{resident: residentNs, nonResident: nonResidentNs} {
		m := completeMeasurementOf(id, models.MinCount, ns, 0, 0)
		e.Measurements[id] = m
		e.MeasurementPaths[id] = "results/campaigns/C-1/measurements/" + models.SubjectDir(id) + ".json"
	}
	return e
}

func TestUC005_H008(t *testing.T) {
	t.Parallel()
	const l1 = 49152
	const llc = 36 << 20
	cases := []struct {
		name                      string
		residentNs, nonResidentNs float64
		want                      models.Outcome
		contains                  string
	}{
		{"rapport dans l'intervalle", 0.78, 131.5, models.OutcomeConfirmed, ""},
		{"borne basse atteinte", 12.0, 120.0, models.OutcomeConfirmed, ""},
		{"latence non résidente trop faible", 0.78, 90.0, models.OutcomeRefuted, "au plus 100"},
		{"rapport trop faible", 20.0, 150.0, models.OutcomeRefuted, "sous 10"},
		{"rapport trop élevé", 0.5, 150.0, models.OutcomeRefuted, "au-dessus de 200"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := evaluateH008(h008Evidence(l1, llc, tc.residentNs, tc.nonResidentNs))
			if result.Outcome != tc.want {
				t.Fatalf("verdict = %s, %s attendu — %s", result.Outcome, tc.want, result.Rationale)
			}
			if tc.contains != "" && !strings.Contains(result.Rationale, tc.contains) {
				t.Fatalf("rationale = %q", result.Rationale)
			}
			if tc.want != models.OutcomeInconclusive && len(result.Files) != 2 {
				t.Fatalf("BR-005-2 : deux fichiers de mesure attendus, %v", result.Files)
			}
		})
	}
}

func TestUC005_H008_Gardes(t *testing.T) {
	t.Parallel()
	t.Run("provenance sans tailles de cache", func(t *testing.T) {
		t.Parallel()
		// Mutation : substituer une constante aux tailles relevées ⇒ échec attendu, et H-008
		// répéterait la faute de H-004, dont le seuil de 32 Mo tenait dans le cache de la machine.
		result := evaluateH008(h008Evidence(0, 0, 0.78, 131.5))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("bande non peuplée", func(t *testing.T) {
		t.Parallel()
		// Avec un cache de dernier niveau démesuré, aucune sonde n'atteint la bande non résidente.
		result := evaluateH008(h008Evidence(49152, 1<<40, 0.78, 131.5))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("latence résidente nulle", func(t *testing.T) {
		t.Parallel()
		result := evaluateH008(h008Evidence(49152, 36<<20, 0, 131.5))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
}

// h009Matrix construit une matrice portant les trois conteneurs et le témoin local.
func h009Matrix(t *testing.T) models.Matrix {
	t.Helper()
	params := models.MatrixParameters{
		Sizes:                []int{24, 64},
		PointerFieldVariants: []bool{false},
		Profiles: []models.LifetimeProfile{models.ProfileLocal, models.ProfileStoredInMap,
			models.ProfileStoredInSlice, models.ProfileStoredInStruct},
		PassingModes: models.PassingModes(),
	}
	matrix, err := models.NewMatrix(params, "digest", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	return matrix
}

// h009Evidence associe à chaque cellule POINTER un verdict d'échappement.
func h009Evidence(t *testing.T, escapes func(cell models.Cell) bool) Evidence {
	t.Helper()
	matrix := h009Matrix(t)
	report := models.EscapeReport{MatrixID: matrix.ID}
	for _, cell := range matrix.Cells {
		if cell.PassingMode != models.PassingPointer {
			continue
		}
		verdict := models.EscapeVerdict{CellID: cell.ID(), Status: models.EscapeStatusOK, Category: models.CategoryNone}
		if escapes(cell) {
			verdict.Escapes = true
			verdict.Category = models.CategoryContainerStore
			verdict.CompilerReason = "moved to heap: t"
		}
		report.Verdicts = append(report.Verdicts, verdict)
	}
	return Evidence{
		Campaign:     models.Campaign{ID: "C-1"},
		Matrix:       matrix,
		EscapeReport: &report,
		EscapePath:   "results/escape/" + matrix.ID + "/x.json",
	}
}

func TestUC005_H009(t *testing.T) {
	t.Parallel()
	t.Run("les trois conteneurs font échapper", func(t *testing.T) {
		t.Parallel()
		result := evaluateH009(h009Evidence(t, func(cell models.Cell) bool {
			return cell.Profile != models.ProfileLocal
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("un conteneur ne fait pas échapper", func(t *testing.T) {
		t.Parallel()
		// Mutation : ne regarder que STORED_IN_MAP ⇒ échec attendu, la généralisation du livre
		// ne serait plus éprouvée.
		result := evaluateH009(h009Evidence(t, func(cell models.Cell) bool {
			return cell.Profile != models.ProfileLocal && cell.Profile != models.ProfileStoredInStruct
		}))
		if result.Outcome != models.OutcomeRefuted {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
		if !strings.Contains(result.Rationale, string(models.ProfileStoredInStruct)) {
			t.Fatalf("le rationale doit nommer le conteneur fautif : %q", result.Rationale)
		}
	})
	t.Run("une seule taille ne suffit pas à infirmer", func(t *testing.T) {
		t.Parallel()
		result := evaluateH009(h009Evidence(t, func(cell models.Cell) bool {
			if cell.Profile == models.ProfileLocal {
				return false
			}
			return !(cell.Profile == models.ProfileStoredInStruct && cell.TypeSpec.SizeBytes == 24)
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
}

func TestUC005_H009_Gardes(t *testing.T) {
	t.Parallel()
	t.Run("sans fichier de verdicts", func(t *testing.T) {
		t.Parallel()
		if got := evaluateH009(Evidence{Campaign: models.Campaign{ID: "C-1"}}).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s", got)
		}
	})
	t.Run("témoin local manquant", func(t *testing.T) {
		t.Parallel()
		// Sans cellules locales qui n'échappent pas, un échappement partout ne prouverait rien.
		// Mutation : retirer la garde du témoin ⇒ échec attendu.
		result := evaluateH009(h009Evidence(t, func(models.Cell) bool { return true }))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
		if !strings.Contains(result.Rationale, "témoin") {
			t.Fatalf("rationale = %q", result.Rationale)
		}
	})
	t.Run("un conteneur sans deux tailles", func(t *testing.T) {
		t.Parallel()
		evidence := h009Evidence(t, func(cell models.Cell) bool { return cell.Profile != models.ProfileLocal })
		var kept []models.EscapeVerdict
		for _, v := range evidence.EscapeReport.Verdicts {
			cell, _ := evidence.Matrix.Cell(v.CellID)
			if cell.Profile == models.ProfileStoredInSlice && cell.TypeSpec.SizeBytes == 64 {
				continue
			}
			kept = append(kept, v)
		}
		evidence.EscapeReport.Verdicts = kept
		if got := evaluateH009(evidence).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s", got)
		}
	})
}

// h010Evidence construit une matrice de retours qui allouent, et leurs mesures.
func h010Evidence(t *testing.T, repeats []int, allocs func(repeat int, pointer bool) int64) Evidence {
	t.Helper()
	params := models.MatrixParameters{
		Sizes:                []int{24},
		PointerFieldVariants: []bool{true},
		Profiles:             []models.LifetimeProfile{models.ProfileReturnedAlloc},
		PassingModes:         models.PassingModes(),
		Repeats:              repeats,
	}
	matrix, err := models.NewMatrix(params, "digest", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewMatrix : %v", err)
	}
	e := Evidence{
		Campaign:         models.Campaign{ID: "C-1"},
		Matrix:           matrix,
		Measurements:     map[string]models.Measurement{},
		MeasurementPaths: map[string]string{},
	}
	for _, cell := range matrix.Cells {
		count := allocs(cell.Repetitions(), cell.PassingMode == models.PassingPointer)
		m := completeMeasurementOf(cell.ID(), models.MinCount, 10, 24, count)
		m.CampaignID = "C-1"
		e.Measurements[cell.ID()] = m
		e.MeasurementPaths[cell.ID()] = "results/campaigns/C-1/measurements/" + models.SubjectDir(cell.ID()) + ".json"
	}
	return e
}

func TestUC005_H010(t *testing.T) {
	t.Parallel()
	repeats := []int{1, 2, 4, 16}
	t.Run("le pointeur double exactement", func(t *testing.T) {
		t.Parallel()
		result := evaluateH010(h010Evidence(t, repeats, func(repeat int, pointer bool) int64 {
			if pointer {
				return int64(2 * repeat)
			}
			return int64(repeat)
		}))
		if result.Outcome != models.OutcomeConfirmed {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("le surcoût est additif", func(t *testing.T) {
		t.Parallel()
		// Le livre parle d'un doublement : un terme constant additionnel l'infirme.
		// Mutation : accepter un rapport ≥ 2 au lieu du double exact ⇒ échec attendu.
		result := evaluateH010(h010Evidence(t, repeats, func(repeat int, pointer bool) int64 {
			if pointer {
				return int64(repeat + 1)
			}
			return int64(repeat)
		}))
		if result.Outcome != models.OutcomeRefuted {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
}

func TestUC005_H010_Gardes(t *testing.T) {
	t.Parallel()
	doubling := func(repeat int, pointer bool) int64 {
		if pointer {
			return int64(2 * repeat)
		}
		return int64(repeat)
	}
	t.Run("base nulle du côté valeur", func(t *testing.T) {
		t.Parallel()
		// Une base nulle relève de H-003 : elle n'est jamais éligible ici.
		result := evaluateH010(h010Evidence(t, []int{1, 2, 4, 16}, func(repeat int, pointer bool) int64 {
			if pointer {
				return 1
			}
			return 0
		}))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("trop peu de paires", func(t *testing.T) {
		t.Parallel()
		if got := evaluateH010(h010Evidence(t, []int{1, 4}, doubling)).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s", got)
		}
	})
	t.Run("une seule base", func(t *testing.T) {
		t.Parallel()
		result := evaluateH010(h010Evidence(t, []int{1, 2, 4, 16}, func(int, bool) int64 { return 1 }))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
	})
	t.Run("étalement insuffisant", func(t *testing.T) {
		t.Parallel()
		// Des bases de 1 à 2 ne distinguent pas un doublement d'un terme additif.
		// Mutation : retirer l'exigence d'étalement ⇒ échec attendu.
		result := evaluateH010(h010Evidence(t, []int{1, 2, 4, 16}, func(repeat int, pointer bool) int64 {
			base := int64(1)
			if repeat >= 4 {
				base = 2
			}
			if pointer {
				return 2 * base
			}
			return base
		}))
		if result.Outcome != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s — %s", result.Outcome, result.Rationale)
		}
		if !strings.Contains(result.Rationale, "étalement") {
			t.Fatalf("rationale = %q", result.Rationale)
		}
	})
	t.Run("compte d'allocations instable", func(t *testing.T) {
		t.Parallel()
		evidence := h010Evidence(t, []int{1, 2, 4, 16}, doubling)
		for id, m := range evidence.Measurements {
			m.AllocsPerOp[0]++
			evidence.Measurements[id] = m
		}
		if got := evaluateH010(evidence).Outcome; got != models.OutcomeInconclusive {
			t.Fatalf("verdict = %s : un compte qui varie n'est pas un entier exact", got)
		}
	})
}

func TestReturnedFamily(t *testing.T) {
	t.Parallel()
	// H-010 porte sur la famille des retours ; le profil d'origine n'y produit jamais de base
	// non nulle, seule la variante qui alloue fournit des paires éligibles.
	if !returnedFamily(models.ProfileReturned) || !returnedFamily(models.ProfileReturnedAlloc) {
		t.Fatal("les deux profils de retour appartiennent à la famille")
	}
	for _, profile := range []models.LifetimeProfile{models.ProfileLocal, models.ProfileStoredInMap, models.ProfileStoredInSlice} {
		if returnedFamily(profile) {
			t.Fatalf("%s n'appartient pas à la famille des retours", profile)
		}
	}
}

func TestConstantAllocsEtAbs(t *testing.T) {
	t.Parallel()
	if !constantAllocs(models.Measurement{AllocsPerOp: []int64{2, 2, 2}}) {
		t.Fatal("un compte constant doit être reconnu")
	}
	if constantAllocs(models.Measurement{AllocsPerOp: []int64{2, 3}}) {
		t.Fatal("un compte qui varie n'est pas constant")
	}
	if constantAllocs(models.Measurement{}) {
		t.Fatal("une mesure vide n'a pas de compte constant")
	}
	if abs(-2.5) != 2.5 || abs(2.5) != 2.5 {
		t.Fatal("abs")
	}
	if suffixList(nil) != "" || suffixList([]string{"a", "b"}) != " (a, b)" {
		t.Fatalf("suffixList = %q", suffixList([]string{"a", "b"}))
	}
}
