package cli

import "testing"

// Paramètres des matrices de la série arm64, tels que LANCEMENT.md (section « Série arm64 ») les
// consigne. Le test prouve que chaque ligne de commande donne l'identifiant annoncé, comme le
// harnais de non-régression le prouve pour les matrices archivées (D-63).
const (
	serieClotureParams     = "sizes=8,16,24,128,1024;pointer=false,true;profiles=LOCAL,RETURNED,CAPTURED_BY_CLOSURE,SENT_ON_CHANNEL,STORED_IN_MAP,STORED_IN_SLICE,STORED_IN_STRUCT,RETURNED_ALLOCATING;modes=VALUE,POINTER;layouts=ARRAY_FILL,NAMED_FIELDS,NAMED_FIELDS_SHAM;repeats=1,4,16;payloads=1,2;probes=SEQUENTIAL_SCAN:33554432,SEQUENTIAL_SCAN:134217728,SCATTERED_SCAN:33554432,SCATTERED_SCAN:134217728,APPEND_PREALLOC:100000,APPEND_GROW:100000,POINTER_CHASE:16384,POINTER_CHASE:262144,POINTER_CHASE:268435456"
	serieReferenceParams   = "sizes=8,16,24,32,64,128,256,512,1024,2048,4096;pointer=false,true;profiles=LOCAL,RETURNED,CAPTURED_BY_CLOSURE,SENT_ON_CHANNEL,STORED_IN_MAP;modes=VALUE,POINTER;probes=SEQUENTIAL_SCAN:262144,SCATTERED_SCAN:262144,SEQUENTIAL_SCAN:4194304,SCATTERED_SCAN:4194304,SEQUENTIAL_SCAN:33554432,SCATTERED_SCAN:33554432,SEQUENTIAL_SCAN:134217728,SCATTERED_SCAN:134217728,APPEND_PREALLOC:100000,APPEND_GROW:100000"
	serieSuccesseursParams = "sizes=8,16,24,128;pointer=false,true;profiles=LOCAL;modes=VALUE,POINTER;layouts=ARRAY_FILL,NAMED_FIELDS;replicates=5;probes=APPEND_PREALLOC:100000,APPEND_GROW:100000"
)

func TestUC001_D63_MatricesDeLaSerieArm64(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, spec, want string
		cells, probes    int
	}{
		{"référence, ligne de 64", serieReferenceParams, "M-823d8b5af441", 220, 10},
		{"référence, ligne de 128", serieReferenceParams + ";cacheline=128", "M-9face550a02e", 220, 10},
		{"clôture, ligne de 64", serieClotureParams, "M-b44a93baae51", 532, 9},
		{"clôture, ligne de 128", serieClotureParams + ";cacheline=128", "M-13da29cdebb6", 532, 9},
		{"successeurs H-012, H-014 à H-016", serieSuccesseursParams, "M-ebbb95f809c8", 160, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := ParseParameters(tc.spec)
			if err != nil {
				t.Fatalf("ParseParameters : %v", err)
			}
			if got := params.Normalize().MatrixID(); got != tc.want {
				t.Fatalf("identifiant %s, LANCEMENT.md consigne %s", got, tc.want)
			}
			cells, probes, err := params.Expand()
			if err != nil {
				t.Fatalf("Expand : %v", err)
			}
			if len(cells) != tc.cells || len(probes) != tc.probes {
				t.Fatalf("%d Cell et %d Probe, %d et %d attendues", len(cells), len(probes), tc.cells, tc.probes)
			}
		})
	}
}
