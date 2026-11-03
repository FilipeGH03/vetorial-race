package ia

import (
	"math"
	"testing"
)

// D27: gerações 0–99 com a pena da D23 (0 por batida nova, 5 por tick); de
// 100 a 299 as duas sobem linearmente até 300 e 50; da 300 em diante, a D26
// cheia.
func TestPenasDaGeracao(t *testing.T) {
	if PenasLeves != (Penas{BatidaNova: 0, Tick: 5}) || PenasCheias != (Penas{BatidaNova: 300, Tick: 50}) {
		t.Fatalf("PenasLeves = %+v, PenasCheias = %+v; D27 pede {0 5} e {300 50}", PenasLeves, PenasCheias)
	}
	casos := []struct {
		g    int
		want Penas
	}{
		{0, PenasLeves}, {50, PenasLeves}, {99, PenasLeves}, {100, PenasLeves},
		{150, Penas{75, 16.25}}, {200, Penas{150, 27.5}}, {250, Penas{225, 38.75}},
		{299, Penas{298.5, 49.775}},
		{300, PenasCheias}, {500, PenasCheias}, {10000, PenasCheias},
	}
	for _, c := range casos {
		got := PenasDaGeracao(c.g)
		if math.Abs(got.BatidaNova-c.want.BatidaNova) > 1e-9 || math.Abs(got.Tick-c.want.Tick) > 1e-9 {
			t.Errorf("geração %d: %+v, esperava %+v", c.g, got, c.want)
		}
	}
	// As penas nunca descem de uma geração para a seguinte.
	ant := PenasDaGeracao(0)
	for g := 1; g <= 400; g++ {
		p := PenasDaGeracao(g)
		if p.BatidaNova < ant.BatidaNova || p.Tick < ant.Tick {
			t.Fatalf("geração %d: %+v desceu de %+v", g, p, ant)
		}
		ant = p
	}
}

// Com PenasCheias a nota é a Pontuacao (D26) e o resto da fórmula não muda com
// as penas: só as batidas pesam diferente.
func TestNotaComPenas(t *testing.T) {
	for _, c := range []struct {
		dist         float64
		voltas       int
		tempo        float64
		novas, ticks int
	}{{500, 0, 0, 0, 0}, {2000, 1, 40, 2, 3}, {300, 0, 0, 1, 4}, {9000, 3, 18.5, 4, 11}} {
		if got, want := PenasCheias.Nota(c.dist, c.voltas, c.tempo, c.novas, c.ticks), Pontuacao(c.dist, c.voltas, c.tempo, c.novas, c.ticks); got != want {
			t.Errorf("%+v: PenasCheias.Nota %v, Pontuacao %v", c, got, want)
		}
		leve := PenasLeves.Nota(c.dist, c.voltas, c.tempo, c.novas, c.ticks)
		if want := Pontuacao(c.dist, c.voltas, c.tempo, c.novas, c.ticks) + 300*float64(c.novas) + 45*float64(c.ticks); math.Abs(leve-want) > 1e-9 {
			t.Errorf("%+v: PenasLeves.Nota %v, esperava %v", c, leve, want)
		}
	}
	a := treinoOval(t).Rodar(redeFixa(2, 2, 0)) // bate na parede: as penas pesam
	if a.Nota(PenasCheias) != a.Pontuacao {
		t.Errorf("Avaliacao.Nota(PenasCheias) = %v, Pontuacao = %v", a.Nota(PenasCheias), a.Pontuacao)
	}
	if !(a.Nota(PenasLeves) > a.Pontuacao) {
		t.Errorf("com batidas, a nota leve (%v) deveria passar da cheia (%v)", a.Nota(PenasLeves), a.Pontuacao)
	}
}

// D27: a população inteira, elite inclusive, é reavaliada a cada geração com
// as penas daquela geração; Rede.Pontuacao (a do arquivo) fica com a fórmula
// cheia.
func TestAvaliarPopulacaoPenasDaGeracao(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 12
	params.Elite = 3
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []int{0, 99, 199, 299} {
		pop.Geracao = g // pula para a geração; Proxima soma 1
		if err := pop.Proxima(); err != nil {
			t.Fatal(err)
		}
		pn := PenasDaGeracao(pop.Geracao)
		res, err := tr.AvaliarPopulacao(pop, 0)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range pop.Redes {
			if want := res[i].Nota(pn); pop.Pontuacoes[i] != want {
				t.Errorf("geração %d, rede %d (elite: %v): nota de seleção %v, esperava %v", pop.Geracao, i, i < params.Elite, pop.Pontuacoes[i], want)
			}
			if want := tr.Rodar(r).Pontuacao; r.Pontuacao != want || res[i].Pontuacao != want {
				t.Errorf("geração %d, rede %d: Rede.Pontuacao %v, Avaliacao.Pontuacao %v, fórmula cheia %v", pop.Geracao, i, r.Pontuacao, res[i].Pontuacao, want)
			}
		}
	}
}
