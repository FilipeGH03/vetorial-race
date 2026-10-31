package sim

import (
	"fmt"
	"testing"
)

// A pista real do repositório (oval.json), com a largada deslocada e com as
// larguras extremas do contrato (40 e 200): a largada segue válida.
func TestLargadaOvalRealDeslocadaLarguras(t *testing.T) {
	cfg := ConfigPadrao()
	base, err := CarregarPista(caminhoOval)
	if err != nil {
		t.Fatalf("CarregarPista: %v", err)
	}
	for _, larg := range []float64{40, base.Largura, 200} {
		for _, k := range append([]int{0}, deslocamentos(len(base.Centro))...) {
			p, err := NovaPista(fmt.Sprintf("oval l%.0f +%d", larg, k), larg, girar(base.Centro, k))
			if err != nil {
				continue // largura grande demais para a curva: recusada pelo contrato, não é largada
			}
			for _, n := range []int{1, 4, 8} {
				carros := Largada(p, n)
				conferirLargada(t, p, n, carros)
				conferirDirecaoLargada(t, p, n, carros)
				for i := range carros {
					c := &carros[i]
					for tick := 0; tick < 300 && c.Progresso.Proximo == 0; tick++ {
						c.Avancar(motorista(p, c), p, cfg)
					}
					if c.Progresso.Proximo != 1 || c.Progresso.Voltas != 0 {
						t.Errorf("%s, %d carros: carro %d ficou com %+v", p.Nome, n, i, c.Progresso)
					}
				}
			}
		}
	}
}
