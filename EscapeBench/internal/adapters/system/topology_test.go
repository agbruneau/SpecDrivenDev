package system

import "testing"

func TestTopologyFrom(t *testing.T) {
	t.Parallel()
	levels := []CacheLevel{
		{Level: 1, Data: false, SizeBytes: 32768},  // cache d'instructions : ignoré
		{Level: 1, Data: true, SizeBytes: 49152},   // L1 de données d'un cœur performance
		{Level: 1, Data: true, SizeBytes: 32768},   // L1 de données d'un cœur efficacité
		{Level: 2, Data: true, SizeBytes: 3 << 20}, // L2 par cluster
		{Level: 3, Data: true, SizeBytes: 36 << 20},
		{Level: 3, Data: true, SizeBytes: 36 << 20},
		{Level: 3, Data: true, SizeBytes: 0}, // instance sans taille : ignorée
	}
	got := topologyFrom(levels)
	// Le L1 de données retenu est le plus grand, celui qu'un cœur performance voit.
	// Mutation : retenir le plus petit ⇒ échec attendu, et les bandes de H-008 se déplaceraient.
	if got.L1DataCacheBytes != 49152 {
		t.Fatalf("L1 de données = %d", got.L1DataCacheBytes)
	}
	if got.LastLevelCacheBytes != 36<<20 {
		t.Fatalf("dernier niveau = %d", got.LastLevelCacheBytes)
	}
}

func TestTopologyFromSansDonnees(t *testing.T) {
	t.Parallel()
	// Sans relevé exploitable, la topologie reste vide : H-008 se déclare non concluante plutôt
	// que de supposer une taille.
	cases := map[string][]CacheLevel{
		"aucune instance":          nil,
		"que des instructions":     {{Level: 1, Data: false, SizeBytes: 32768}},
		"que des tailles absentes": {{Level: 1, Data: true, SizeBytes: 0}, {Level: 3, Data: true}},
	}
	for name, levels := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := topologyFrom(levels)
			if got.L1DataCacheBytes != 0 || got.LastLevelCacheBytes != 0 {
				t.Fatalf("topologie = %+v, vide attendue", got)
			}
		})
	}
}

func TestTopologySansL1MaisAvecDernierNiveau(t *testing.T) {
	t.Parallel()
	got := topologyFrom([]CacheLevel{{Level: 2, Data: true, SizeBytes: 1 << 20}})
	if got.L1DataCacheBytes != 0 {
		t.Fatalf("L1 de données = %d, zéro attendu", got.L1DataCacheBytes)
	}
	if got.LastLevelCacheBytes != 1<<20 {
		t.Fatalf("dernier niveau = %d", got.LastLevelCacheBytes)
	}
}

func TestSortedLevels(t *testing.T) {
	t.Parallel()
	got := sortedLevels([]CacheLevel{
		{Level: 3, SizeBytes: 100}, {Level: 1, SizeBytes: 20}, {Level: 1, SizeBytes: 10},
	})
	if got[0].SizeBytes != 10 || got[1].SizeBytes != 20 || got[2].Level != 3 {
		t.Fatalf("ordre = %+v", got)
	}
}

func TestDetectionReelleSurCetteMachine(t *testing.T) {
	t.Parallel()
	// La détection ne doit jamais paniquer, quelle que soit la plateforme. Quand elle rend des
	// tailles, elles doivent être cohérentes entre elles.
	got := detectTopology()
	if got.L1DataCacheBytes < 0 || got.LastLevelCacheBytes < 0 {
		t.Fatalf("tailles négatives : %+v", got)
	}
	if got.L1DataCacheBytes > 0 && got.LastLevelCacheBytes > 0 && got.L1DataCacheBytes >= got.LastLevelCacheBytes {
		t.Fatalf("le L1 de données ne peut pas égaler ou dépasser le dernier niveau : %+v", got)
	}
}
