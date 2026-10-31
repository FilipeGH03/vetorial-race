package ia

import (
	"errors"
	"reflect"
	"testing"

	"vetorial-race/internal/sim"
)

func pistaOval(t *testing.T) *sim.Pista {
	t.Helper()
	p, err := sim.CarregarPista("../../pistas/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func treinoOval(t *testing.T) *Treino {
	t.Helper()
	tr, err := NovoTreino(pistaOval(t))
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func TestNovoTreinoConfigELargada(t *testing.T) {
	tr := treinoOval(t)
	want := sim.ConfigPadrao() // D21: BaterMata = false, como na corrida
	if tr.Cfg != want || tr.Cfg.BaterMata {
		t.Errorf("Cfg = %+v, esperava ConfigPadrao sem BaterMata: %+v", tr.Cfg, want)
	}
	if tr.MaxTicks != 3600 {
		t.Errorf("MaxTicks = %d, esperava 60 s = 3600", tr.MaxTicks)
	}
	l := sim.Largada(tr.Pista, 1)[0]
	if tr.largada.Pos != l.Pos || tr.largada.Angulo() != l.Angulo() || !tr.largada.Vivo {
		t.Errorf("largada = %+v, esperava sim.Largada(p, 1)[0] = %+v", tr.largada, l)
	}
}

func TestNovoTreinoSemPista(t *testing.T) {
	if _, err := NovoTreino(nil); !errors.Is(err, ErrTreino) {
		t.Fatalf("pista nil: err = %v", err)
	}
	if _, err := NovoTreino(&sim.Pista{Nome: "vazia"}); !errors.Is(err, ErrTreino) {
		t.Fatalf("pista sem largada: err = %v", err)
	}
}

func TestPontuacao(t *testing.T) {
	if got := Pontuacao(500, 0, 0, 0); got != 500 {
		t.Errorf("sem volta: %v", got)
	}
	if got := Pontuacao(1500, 1, 30, 0); got != 1500+BonusVolta+BonusTempo*30 {
		t.Errorf("1 volta em 30 s: %v", got)
	}
	if Pontuacao(1500, 1, 20, 0) <= Pontuacao(1500, 1, 30, 0) {
		t.Error("volta mais rápida deveria valer mais")
	}
	if got := Pontuacao(0, 0, 0, 0); got != 0 {
		t.Errorf("parado: %v", got)
	}
	// D21: Distancia + 1000·voltas + 20·(60 − tempo da 1ª volta) − 5·batidas.
	if got := Pontuacao(2000, 1, 40, 3); got != 2000+1000+20*20-5*3 {
		t.Errorf("1 volta em 40 s com 3 batidas: %v", got)
	}
	if got := Pontuacao(300, 0, 0, 4); got != 300-20 {
		t.Errorf("4 batidas sem volta: %v", got)
	}
}

// Rede parada (zona morta sempre) não avança: é encerrada depois de
// TicksParado ticks (D21), sem batida e com nota 0.
func TestRodarParadoEncerraEm60Ticks(t *testing.T) {
	tr := treinoOval(t)
	res := tr.Rodar(redeFixa(0, 0, 0)) // saída (0, 0): zona morta sempre
	want := Resultado{Ticks: TicksParado, Parou: true}
	if res != want {
		t.Fatalf("rede parada: %+v, esperava %+v", res, want)
	}
}

// Sempre aumenta x: cruza a largada, segue a reta e bate na quina. Não morre
// (D21): fica empurrando a parede, cada tick conta uma batida, e é encerrada
// TicksParado ticks depois de a Distancia parar de subir.
func TestRodarRetoBateEParaSemMorrer(t *testing.T) {
	tr := treinoOval(t)
	res := tr.Rodar(redeFixa(2, 2, 0))
	if !res.Parou || res.Ticks >= tr.MaxTicks || res.Batidas == 0 {
		t.Fatalf("deveria bater e ser encerrada por ficar parada: %+v", res)
	}
	if res.Distancia <= 0 || res.Voltas != 0 || res.Pontuacao != res.Distancia-PenaBatida*float64(res.Batidas) {
		t.Fatalf("resultado %+v", res)
	}
}

// Sempre diminui x: nunca cruza a largada, bate atrás dela e só perde pontos.
func TestRodarDeReNaoPontua(t *testing.T) {
	res := treinoOval(t).Rodar(redeFixa(-2, 2, 0))
	if !res.Parou || res.Distancia != 0 || res.Batidas == 0 || res.Pontuacao != -PenaBatida*float64(res.Batidas) {
		t.Fatalf("de ré: %+v", res)
	}
}

// Com a Config do treino bater não mata (D21): quem bate e sai da parede segue
// dirigindo e a Distancia volta a subir.
func TestTreinoBaterNaoMata(t *testing.T) {
	tr := treinoOval(t)
	ctl := &alterna{}
	c := tr.largada
	bateu, subiuDepois := false, false
	var dist float64
	for n := 0; n < 600; n++ {
		c.Avancar(ctl.Decidir(sim.Ler(&c, tr.Pista, tr.Cfg)), tr.Pista, tr.Cfg)
		if c.Batido {
			bateu, dist = true, c.Progresso.Distancia
			ctl.bateu = true
		} else if bateu && c.Progresso.Distancia > dist {
			subiuDepois = true
			break
		}
	}
	if !bateu || !subiuDepois || !c.Vivo {
		t.Fatalf("bateu %v, subiu depois %v, vivo %v", bateu, subiuDepois, c.Vivo)
	}
}

// alterna sobe y e x até bater na parede de dentro e, depois, desce y e
// acelera em +x, saindo da parede.
type alterna struct{ bateu bool }

func (a *alterna) Decidir(sim.Leitura) sim.Controle {
	if !a.bateu {
		return sim.Controle{Acoes: []sim.Acao{{Ativa: true, Eixo: sim.EixoY, Sinal: 1}, {Ativa: true, Eixo: sim.EixoX, Sinal: 1}}}
	}
	return sim.Controle{Acoes: []sim.Acao{{Ativa: true, Eixo: sim.EixoX, Sinal: 1}, {Ativa: true, Eixo: sim.EixoY, Sinal: -1}}}
}

func TestRodarNaoAlteraARede(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, nil)
	antes := r.Copia()
	treinoOval(t).Rodar(r)
	if !reflect.DeepEqual(r, antes) {
		t.Fatal("Rodar alterou a rede")
	}
}

func TestAvaliarPopulacaoIgualEmSerieEEmParalelo(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 12
	params.Elite = 2
	nova := func() *Populacao {
		pop, err := NovaPopulacao(params)
		if err != nil {
			t.Fatal(err)
		}
		return pop
	}
	a, b := nova(), nova()
	resA, err := tr.AvaliarPopulacao(a, 1)
	if err != nil {
		t.Fatal(err)
	}
	resB, err := tr.AvaliarPopulacao(b, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resA, resB) || !reflect.DeepEqual(a.Pontuacoes, b.Pontuacoes) {
		t.Fatalf("1 worker e 8 workers deram resultados diferentes:\n%v\n%v", resA, resB)
	}
	for i, r := range a.Redes {
		if want := tr.Rodar(r); resA[i] != want {
			t.Errorf("rede %d: %+v, Rodar dá %+v", i, resA[i], want)
		}
		if a.Pontuacoes[i] != resA[i].Pontuacao || r.Pontuacao != resA[i].Pontuacao {
			t.Errorf("rede %d: nota %v / %v, resultado %v", i, a.Pontuacoes[i], r.Pontuacao, resA[i].Pontuacao)
		}
	}
}

func TestTreinoCurtoMelhorNaoCai(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 16
	params.Elite = 2
	params.Camadas = []int{NEntrada, 6, 4, NSaida} // o treino não fixa 14-12-2
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	anterior := -1.0
	for g := 0; g < 5; g++ {
		if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
			t.Fatal(err)
		}
		_, nota := pop.Melhor()
		if nota < anterior {
			t.Fatalf("geração %d: melhor caiu de %v para %v (elite e simulação são determinísticas)", g, anterior, nota)
		}
		anterior = nota
		if err := pop.Proxima(); err != nil {
			t.Fatal(err)
		}
	}
}
