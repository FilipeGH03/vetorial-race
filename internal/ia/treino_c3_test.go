package ia

import (
	"math"
	"testing"

	"vetorial-race/internal/sim"
)

// Uso real do jogo (B5): as 6 posições de Largada(p, 6) ao mesmo tempo, todas
// com o cérebro versionado do oval, numa só sim.Corrida de 60 s. Cada carro
// completa >= 1 volta e nunca passa 2 s (120 ticks) sem a Distancia subir.
// Pulado com -short.
func TestCerebroOvalSeisCarrosJuntos(t *testing.T) {
	if testing.Short() {
		t.Skip("prova da C3")
	}
	r, err := CarregarRede("../../cerebros/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	p := pistaOval(t)
	ctls := make([]sim.Controlador, NLargadas)
	for i := range ctls {
		ctls[i] = NovoControlador(r)
	}
	cor, err := sim.NovaCorrida(p, sim.ConfigPadrao(), sim.ModoVoltas, 50, ctls, -1)
	if err != nil {
		t.Fatal(err)
	}
	ultima := make([]float64, NLargadas)
	parado := make([]int, NLargadas)
	for cor.Tick < 3600 && !cor.Terminada {
		cor.Step()
		for k, c := range cor.Carros {
			if c.Progresso.Distancia > ultima[k] {
				ultima[k], parado[k] = c.Progresso.Distancia, 0
			} else if parado[k]++; parado[k] > 120 {
				t.Fatalf("carro %d: mais de 120 ticks sem avançar (tick %d)", k, cor.Tick)
			}
		}
	}
	for k, c := range cor.Carros {
		if c.Progresso.Voltas < 1 {
			t.Errorf("carro %d: %d voltas, distância %.0f", k, c.Progresso.Voltas, c.Progresso.Distancia)
		}
	}
}

// O treino (Treino.Rodar) e a prova da C3 concordam: o cérebro do oval completa
// a volta de todas as largadas e nenhuma é encerrada por ficar parada.
func TestCerebroOvalNoTreinoNaoParaECompletaVolta(t *testing.T) {
	r, err := CarregarRede("../../cerebros/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	a := treinoOval(t).Rodar(r)
	if a.VoltasMin() < 1 || a.ComVolta() != NLargadas {
		t.Errorf("voltas mínimas %d, com volta %d/%d", a.VoltasMin(), a.ComVolta(), NLargadas)
	}
	for k, res := range a.Largadas {
		if res.Parou || res.Ticks != 3600 {
			t.Errorf("largada %d: %+v, esperava dirigir os 60 s sem parar", k, res)
		}
	}
	if math.Abs(a.Pontuacao-r.Pontuacao) > 1e-6 {
		t.Errorf("nota %v, o arquivo diz %v", a.Pontuacao, r.Pontuacao)
	}
}

// D23: Avanco acompanha o carro por mais de uma volta, passando pela emenda da
// linha central (Centro[0]) sem salto: nunca encerra, e a soma do avanço é
// ~2 voltas. Passos de 4 px sobre a linha central, a ~0,24 de VMax.
func TestAvancoDaDuasVoltasPelaEmenda(t *testing.T) {
	p := pistaOval(t)
	pt := func(s float64) sim.Vec2 {
		s = math.Mod(s, p.Comprimento)
		for i := range p.Centro {
			a, b := p.Centro[i], p.Centro[(i+1)%len(p.Centro)]
			l := b.Sub(a).Len()
			if s <= l {
				return a.Add(b.Sub(a).Scale(s / l))
			}
			s -= l
		}
		return p.Centro[0]
	}
	var av Avanco
	for s := 0.0; s < 2*p.Comprimento; s += 4 {
		if av.Tick(p, pt(s), pt(s+4)) {
			t.Fatalf("encerrado andando no sentido da pista em s = %.0f", s)
		}
	}
	if math.Abs(av.linha-2*p.Comprimento) > 30 {
		t.Errorf("avanço somado %.1f, esperava ~%.1f (2 voltas)", av.linha, 2*p.Comprimento)
	}
}

// D23: carro que não mexe em nada (rede na zona morta) é encerrado no
// TicksParado-ésimo tick, de qualquer largada, sem pontuar nem bater.
func TestRodarSemAcaoEncerraEmTicksParado(t *testing.T) {
	for k, res := range treinoOval(t).Rodar(redeFixa(0, 0, 0)).Largadas {
		if !res.Parou || res.Ticks != TicksParado || res.Distancia != 0 || res.Batidas != 0 || res.Pontuacao != 0 {
			t.Errorf("largada %d: %+v", k, res)
		}
	}
}
