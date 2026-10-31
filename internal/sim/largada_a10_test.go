package sim

import (
	"fmt"
	"testing"
)

// A10: cada fila da largada tem no máximo floor(largura livre da fila / (0,2·
// largura)) carros. Filas ficam a ≥ 0,5 largura uma da outra, então dois carros
// a menos de 0,2 largura de distância são da mesma fila e estão apertados.
func TestLargadaDistanciaMinimaEntreCarrosDaFila(t *testing.T) {
	retangulo := []Vec2{{30, 30}, {1250, 30}, {1250, 690}, {30, 690}}
	base := pistasDeForma(t)
	pr, err := NovaPista("retangulo 1280x720", 60, retangulo)
	if err != nil {
		t.Fatalf("NovaPista(retângulo): %v", err)
	}
	base = append(base, pr)
	for _, b := range base {
		for _, k := range append([]int{0}, deslocamentos(len(b.Centro))...) {
			p, err := NovaPista(fmt.Sprintf("%s +%d", b.Nome, k), b.Largura, girar(b.Centro, k))
			if err != nil {
				t.Fatalf("%s +%d: NovaPista: %v", b.Nome, k, err)
			}
			for _, n := range []int{2, 3, 6, 8} {
				carros := Largada(p, n)
				conferirLargada(t, p, n, carros)
				minimo := espacoLateralLargada * p.Largura
				for i := range carros {
					for j := i + 1; j < len(carros); j++ {
						if d := carros[i].Pos.Sub(carros[j].Pos).Len(); d < minimo-1e-9 {
							t.Errorf("%s, %d carros: carros %d e %d a %.2f px um do outro, quer ≥ %.2f",
								p.Nome, n, i, j, d, minimo)
						}
					}
				}
			}
		}
	}
}
