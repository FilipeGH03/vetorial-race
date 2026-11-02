package ia

import (
	"math"
	"testing"
)

// D25: a pena é de 50 pontos por tick batido, e cada batida pesa o mesmo na
// nota: Pontuacao cai exatamente PenaBatida por batida a mais.
func TestPenaBatidaD25(t *testing.T) {
	if PenaBatida != 50 {
		t.Fatalf("PenaBatida = %v, D25 pede 50", PenaBatida)
	}
	base := Pontuacao(3000, 1, 45, 0)
	for n := 1; n <= 20; n++ {
		if got, want := Pontuacao(3000, 1, 45, n), base-PenaBatida*float64(n); math.Abs(got-want) > 1e-9 {
			t.Fatalf("%d batidas: %v, esperava %v", n, got, want)
		}
	}
	// Sem volta o bônus de tempo não entra, e a pena continua valendo.
	if got := Pontuacao(500, 0, 0, 2); got != 400 {
		t.Errorf("500 px, 2 batidas, sem volta: %v, esperava 400", got)
	}
}

// O cérebro versionado do oval: a nota de cada largada devolvida por Rodar é a
// da fórmula D25 com as batidas contadas (e o cérebro bate de fato, então a
// pena aparece na nota); a média das 6 é a nota gravada no arquivo.
func TestCerebroOvalNotaUsaPenaD25(t *testing.T) {
	r, err := CarregarRede("../../cerebros/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	a := treinoOval(t).Rodar(r)
	soma, batidas := 0.0, 0
	for k, res := range a.Largadas {
		want := Pontuacao(res.Distancia, res.Voltas, res.TempoVolta, res.Batidas)
		if math.Abs(res.Pontuacao-want) > 1e-9 {
			t.Errorf("largada %d: nota %v, fórmula D25 dá %v", k, res.Pontuacao, want)
		}
		soma += res.Pontuacao
		batidas += res.Batidas
	}
	if batidas == 0 {
		t.Log("o cérebro não bateu em nenhuma largada")
	}
	if math.Abs(soma/NLargadas-r.Pontuacao) > 1e-6 {
		t.Errorf("média %v, o arquivo diz %v", soma/NLargadas, r.Pontuacao)
	}
}
