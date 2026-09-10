// Package harness contient le code de mesure commun à toutes les cellules (C-005) : la boucle
// de benchmark, les profils de durée de vie et les fonctions puits. Son empreinte SHA-256 est
// enregistrée avec chaque Matrix et chaque Campaign (BR-003-1) ; il est figé pendant une
// campagne (hook guard-paths, results/.campaign-lock).
package harness
