package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/agbruneau/escapebench/internal/models"
	"github.com/agbruneau/escapebench/internal/service"
)

// RenderProvenance rend la provenance sur une ligne (NFR-001).
func RenderProvenance(p models.Provenance) string {
	return fmt.Sprintf("%s %s/%s · %s · %s", p.GoVersion, p.GOOS, p.GOARCH, p.CPUModel, p.CapturedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
}

// RenderMatrix met en forme l'étape 7 de UC-001.
func RenderMatrix(report service.MatrixReport) string {
	var b strings.Builder
	if report.AlreadyExisted {
		fmt.Fprintf(&b, "Matrice existante : %s (harnais inchangé, rien de régénéré)\n", report.MatrixID)
	} else {
		fmt.Fprintf(&b, "Matrice : %s\n", report.MatrixID)
	}
	fmt.Fprintf(&b, "TypeSpec : %d · Cell : %d · Probe : %d\n", report.TypeSpecCount, report.CellCount, report.ProbeCount)
	fmt.Fprintf(&b, "Empreinte du harnais : %s\n", report.HarnessDigest)
	if len(report.CompileErrors) > 0 {
		fmt.Fprintf(&b, "Sujets non compilables : %d\n", len(report.CompileErrors))
		for _, failure := range report.CompileErrors {
			fmt.Fprintf(&b, "  %s : %s\n", failure.SubjectID, firstLine(failure.Message))
		}
	}
	return b.String()
}

// RenderEscape met en forme l'étape 7 de UC-002.
func RenderEscape(summary service.EscapeReportSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Matrice : %s\n", summary.MatrixID)
	fmt.Fprintf(&b, "Provenance : %s\n", RenderProvenance(summary.Provenance))
	fmt.Fprintf(&b, "Fichier de verdicts : %s\n", summary.Path)
	b.WriteString("Décompte par catégorie :\n")
	for _, category := range models.EscapeCategories() {
		fmt.Fprintf(&b, "  %-16s %d\n", category, summary.Counts[category])
	}
	fmt.Fprintf(&b, "Cellules classées OTHER : %d\n", summary.OtherCount)
	fmt.Fprintf(&b, "Cellules en COMPILE_ERROR : %d\n", summary.CompileErrors)
	if check := summary.Reproducibility; check != nil {
		fmt.Fprintf(&b, "Reproductibilité (NFR-002) : comparé à %s, %d cellule(s) divergentes\n", check.ComparedTo, len(check.Differing))
		if check.Violation {
			fmt.Fprintf(&b, "  VIOLATION de NFR-002 : %s\n", strings.Join(truncateList(check.Differing, 10), ", "))
		}
	}
	return b.String()
}

// RenderCampaign met en forme les étapes 4 et 8 de UC-003.
func RenderCampaign(report service.CampaignReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Campagne : %s — statut : %s\n", report.CampaignID, report.Status)
	fmt.Fprintf(&b, "Provenance : %s\n", RenderProvenance(report.Provenance))
	fmt.Fprintf(&b, "Empreinte du harnais : %s\n", report.HarnessDigest)
	fmt.Fprintf(&b, "Empreinte des critères : %s (%s)\n", report.HypothesesDigest, strings.Join(report.HypothesisIDs, ", "))
	fmt.Fprintf(&b, "Cell : %d · Probe : %d\n", report.CellCount, report.ProbeCount)
	fmt.Fprintf(&b, "Sujets mesurés / FAILED : %d / %d\n", report.Measured, report.Failed)
	if report.Resumed {
		b.WriteString("Reprise d'une campagne interrompue (UC-003 A4)\n")
	}
	fmt.Fprintf(&b, "Durée : %s\n", report.Duration.Round(1e9))
	if report.AbortReason != "" {
		fmt.Fprintf(&b, "Abandon : %s\n", report.AbortReason)
	}
	return b.String()
}

// RenderComparison met en forme l'étape 7 de UC-004.
func RenderComparison(report service.ComparisonReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Campagne : %s\n", report.CampaignID)
	fmt.Fprintf(&b, "Fichier de comparaison : %s\n", report.Path)
	fmt.Fprintf(&b, "Méthode : %s\n\n", report.Set.Method)

	series := map[models.TippingKey][]models.Comparison{}
	for _, comparison := range report.Set.Comparisons {
		key := models.TippingKey{Profile: comparison.Profile, HasPointerField: comparison.HasPointerField}
		series[key] = append(series[key], comparison)
	}
	keys := make([]models.TippingKey, 0, len(series))
	for key := range series {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Profile != keys[j].Profile {
			return keys[i].Profile < keys[j].Profile
		}
		return !keys[i].HasPointerField && keys[j].HasPointerField
	})
	for _, key := range keys {
		pointerField := "sans champ pointeur"
		if key.HasPointerField {
			pointerField = "avec champ pointeur"
		}
		fmt.Fprintf(&b, "%s, %s\n", key.Profile, pointerField)
		fmt.Fprintf(&b, "  %-8s %12s %12s %12s %s\n", "taille", "Δ ns/op", "IC bas", "IC haut", "significatif")
		rows := series[key]
		sort.Slice(rows, func(i, j int) bool { return rows[i].SizeBytes < rows[j].SizeBytes })
		for _, comparison := range rows {
			fmt.Fprintf(&b, "  %-8d %12.3f %12.3f %12.3f %v\n",
				comparison.SizeBytes, comparison.DeltaNsPerOp, comparison.CILow, comparison.CIHigh, comparison.Significant)
		}
		tipping := report.Set.TippingPoints[key]
		if tipping == models.TippingNotObserved {
			b.WriteString("  Point de bascule : non observé\n\n")
		} else {
			fmt.Fprintf(&b, "  Point de bascule : %d octets\n\n", tipping)
		}
	}
	fmt.Fprintf(&b, "Paires exclues : %d\n", report.ExcludedCount)
	for _, excluded := range report.Set.ExcludedPairs {
		fmt.Fprintf(&b, "  %s / %s : %s\n", excluded.ValueCellID, excluded.PointerCellID, excluded.Reason)
	}
	return b.String()
}

// RenderVerdicts met en forme l'étape 8 de UC-005.
func RenderVerdicts(summary service.VerdictReportSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Campagne : %s\n", summary.CampaignID)
	fmt.Fprintf(&b, "Fichier de verdicts : %s\n", summary.Path)
	fmt.Fprintf(&b, "Tableau de bord : %s\n\n", summary.DashboardPath)
	b.WriteString("| Hypothèse | Verdict | Rationale |\n")
	b.WriteString("|---|---|---|\n")
	for _, verdict := range summary.Report.Verdicts {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", verdict.HypothesisID, verdict.Outcome, verdict.Rationale)
	}
	if len(summary.Inconclusive) > 0 {
		b.WriteString("\nHypothèses INCONCLUSIVE et cause :\n")
		for _, verdict := range summary.Inconclusive {
			fmt.Fprintf(&b, "  %s : %s\n", verdict.HypothesisID, verdict.Rationale)
		}
	} else {
		b.WriteString("\nHypothèses INCONCLUSIVE : aucune\n")
	}
	return b.String()
}

// firstLine rend la première ligne non vide d'un message.
func firstLine(message string) string {
	for _, line := range strings.Split(message, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return message
}

// truncateList borne une énumération affichée.
func truncateList(values []string, max int) []string {
	if len(values) <= max {
		return values
	}
	return append(append([]string(nil), values[:max]...), "…")
}
