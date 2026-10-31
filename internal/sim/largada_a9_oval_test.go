package sim

import (
	"fmt"
	"math"
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

// O oval de exemplo larga no meio da reta de baixo (D23): Largada(oval, 6) põe
// todas as filas na reta (y entre as paredes, x antes da largada e depois do
// fim da curva anterior, apontados para +x), e o trecho da linha central mais
// perto de cada carro é a reta.
func TestLargadaOvalNoMeioDaRetaDeBaixo(t *testing.T) {
	p := carregarOval(t)
	cp0 := p.Checkpoints[0]
	if m := cp0.A.Add(cp0.B).Scale(0.5); m.Sub(Vec2{400, 300}).Len() > 1e-6 {
		t.Fatalf("largada em %v, quer o meio da reta de baixo (400,300)", m)
	}
	carros := Largada(p, 6)
	conferirLargada(t, p, 6, carros)
	for i, c := range carros {
		if c.Pos.X <= 200+FolgaLargada || c.Pos.X >= 400 || math.Abs(c.Pos.Y-300) > p.Largura/2-FolgaLargada {
			t.Errorf("carro %d em %v: fora da reta de baixo antes da largada", i, c.Pos)
		}
		if a := c.Angulo(); math.Abs(a) > 1e-9 {
			t.Errorf("carro %d aponta para %v, quer 0 (+x)", i, a)
		}
		if d := Ler(&carros[i], p, ConfigPadrao()).DirPista[0]; d.Sub(Vec2{1, 0}).Len() > 1e-9 {
			t.Errorf("carro %d: DirPista[0] = %v, quer (1,0)", i, d)
		}
	}
}
