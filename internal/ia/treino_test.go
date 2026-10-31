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
	want := sim.ConfigPadrao() // D23: BaterMata = false, como na corrida
	if tr.Cfg != want || tr.Cfg.BaterMata {
		t.Errorf("Cfg = %+v, esperava ConfigPadrao sem BaterMata: %+v", tr.Cfg, want)
	}
	if tr.MaxTicks != 3600 {
		t.Errorf("MaxTicks = %d, esperava 60 s = 3600", tr.MaxTicks)
	}
	ls := sim.Largada(tr.Pista, NLargadas) // D22: as 6 posições, calculadas uma vez
	for k, l := range ls {
		c := tr.Largada(k)
		if c.Pos != l.Pos || c.Angulo() != l.Angulo() || !c.Vivo || c.Vel != (sim.Vec2{}) {
			t.Errorf("largada %d = %+v, esperava sim.Largada(p, %d)[%d] = %+v", k, c, NLargadas, k, l)
		}
	}
	if tr.Largada(0).Pos == tr.Largada(NLargadas-1).Pos {
		t.Error("largadas repetidas")
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
	// D23: Distancia + 1000·voltas + 20·(60 − tempo da 1ª volta) − 5·batidas.
	if got := Pontuacao(2000, 1, 40, 3); got != 2000+1000+20*20-5*3 {
		t.Errorf("1 volta em 40 s com 3 batidas: %v", got)
	}
	if got := Pontuacao(300, 0, 0, 4); got != 300-20 {
		t.Errorf("4 batidas sem volta: %v", got)
	}
}

// Rede parada (zona morta sempre) não avança: é encerrada depois de
// TicksParado ticks (D23), sem batida e com nota 0.
func TestRodarParadoEncerraEm60Ticks(t *testing.T) {
	tr := treinoOval(t)
	a := tr.Rodar(redeFixa(0, 0, 0)) // saída (0, 0): zona morta sempre
	want := Resultado{Ticks: TicksParado, Parou: true}
	for k, res := range a.Largadas {
		if res != want {
			t.Fatalf("rede parada, largada %d: %+v, esperava %+v", k, res, want)
		}
	}
	if a.Pontuacao != 0 || a.VoltasMin() != 0 || a.ComVolta() != 0 {
		t.Fatalf("rede parada: %+v", a)
	}
}

// D22: a nota é a média das notas das NLargadas largadas, e cada largada é a
// corrida de RodarDe daquela posição.
func TestRodarNotaEAMediaDasLargadas(t *testing.T) {
	tr := treinoOval(t)
	r, _ := NovaRede(CamadasPadrao, nil)
	a := tr.Rodar(r)
	soma := 0.0
	for k := range NLargadas {
		if got := tr.RodarDe(r, k); got != a.Largadas[k] {
			t.Errorf("largada %d: %+v, RodarDe dá %+v", k, a.Largadas[k], got)
		}
		soma += a.Largadas[k].Pontuacao
	}
	if a.Pontuacao != soma/NLargadas {
		t.Errorf("nota %v, média das largadas %v", a.Pontuacao, soma/NLargadas)
	}
}

func TestAvaliacaoVoltas(t *testing.T) {
	var a Avaliacao
	for k := range a.Largadas {
		a.Largadas[k].Voltas = 2
	}
	if a.VoltasMin() != 2 || a.ComVolta() != NLargadas {
		t.Fatalf("todas com 2 voltas: min %d, com volta %d", a.VoltasMin(), a.ComVolta())
	}
	a.Largadas[3].Voltas = 0
	if a.VoltasMin() != 0 || a.ComVolta() != NLargadas-1 {
		t.Fatalf("uma sem volta: min %d, com volta %d", a.VoltasMin(), a.ComVolta())
	}
}

// Sempre aumenta x: da 1ª fila cruza a largada, segue a reta e bate na quina
// (das de trás pode bater antes da linha). Não morre (D23): fica empurrando a
// parede, cada tick conta uma batida, e é encerrada TicksParado ticks depois
// de parar de avançar na linha central.
func TestRodarRetoBateEParaSemMorrer(t *testing.T) {
	tr := treinoOval(t)
	a := tr.Rodar(redeFixa(2, 2, 0))
	for k, res := range a.Largadas {
		if !res.Parou || res.Ticks >= tr.MaxTicks || res.Batidas == 0 {
			t.Fatalf("largada %d: deveria bater e ser encerrada por ficar parada: %+v", k, res)
		}
		if res.Voltas != 0 || res.Pontuacao != res.Distancia-PenaBatida*float64(res.Batidas) {
			t.Fatalf("largada %d: resultado %+v", k, res)
		}
	}
	if a.Largadas[0].Distancia <= 0 {
		t.Fatalf("da 1ª largada deveria cruzar a linha: %+v", a.Largadas[0])
	}
}

// Sempre diminui x: nunca cruza a largada, bate atrás dela e só perde pontos.
// (Com a largada no meio da reta de baixo (D23), de ré em 1 s o carro pode nem
// chegar à parede: Batidas pode ser 0.)
func TestRodarDeReNaoPontua(t *testing.T) {
	for k, res := range treinoOval(t).Rodar(redeFixa(-2, 2, 0)).Largadas {
		if !res.Parou || res.Distancia != 0 || res.Pontuacao != -PenaBatida*float64(res.Batidas) {
			t.Fatalf("de ré, largada %d: %+v", k, res)
		}
	}
}

// D23: o "1 s sem avançar" é medido por sim.AvancoNaLinha, que conta também
// atrás da largada (onde a Distancia ainda é 0). Um carro 190 px atrás da linha
// acelerando em +x leva mais de 1 s para cruzá-la (150 px no 1º segundo): não
// pode ser encerrado antes, e chega a pontuar.
func TestRodarAvancoAtrasDaLargadaNaoEncerra(t *testing.T) {
	tr := treinoOval(t)
	c := sim.NovoCarro(0, sim.Vec2{X: 210, Y: 300}) // reta de baixo; a largada é x = 400
	tr.largadas[0] = c
	res := tr.RodarDe(redeFixa(2, 2, 0), 0)
	if res.Ticks <= TicksParado || res.Distancia <= 0 {
		t.Fatalf("avançando atrás da largada foi encerrado: %+v", res)
	}
	if !res.Parou || res.Batidas == 0 {
		t.Fatalf("deveria seguir até bater na quina e parar lá: %+v", res)
	}
}

// D23: ir e voltar no mesmo lugar não é avançar. De ré o carro nunca passa do
// ponto de largada: é encerrado em exatamente TicksParado ticks.
func TestRodarDeReEncerraEmTicksParado(t *testing.T) {
	for k, res := range treinoOval(t).Rodar(redeFixa(-2, 2, 0)).Largadas {
		if res.Ticks != TicksParado || !res.Parou {
			t.Fatalf("de ré, largada %d: %+v, esperava %d ticks e Parou", k, res, TicksParado)
		}
	}
}

// Avanco (D23): andar para a frente zera a contagem; voltar e ir de novo até
// onde já tinha chegado não é avançar; ficar parado conta; o fim vem no
// TicksParado-ésimo tick seguido sem avançar.
func TestAvancoIrEVoltarNaoConta(t *testing.T) {
	p := pistaOval(t)
	a, b, c := sim.Vec2{X: 300, Y: 300}, sim.Vec2{X: 305, Y: 300}, sim.Vec2{X: 310, Y: 300}
	var av Avanco
	if av.Tick(p, a, b) || av.Tick(p, b, c) {
		t.Fatal("andando para a frente foi encerrado")
	}
	for n := 1; n <= TicksParado; n++ {
		de, ate := c, b // vai e volta entre b e c, sem passar de c
		if n%2 == 0 {
			de, ate = b, c
		}
		if got := av.Tick(p, de, ate); got != (n == TicksParado) {
			t.Fatalf("tick %d sem passar de c: encerrado = %v", n, got)
		}
	}
	av = Avanco{}
	for n := 1; n < TicksParado; n++ {
		if av.Tick(p, a, a) {
			t.Fatalf("parado, tick %d: encerrado cedo", n)
		}
	}
	if av.Tick(p, a, b) {
		t.Fatal("avançou no último tick e foi encerrado")
	}
}

// Com a Config do treino bater não mata (D23): quem bate e sai da parede segue
// dirigindo e a Distancia volta a subir.
func TestTreinoBaterNaoMata(t *testing.T) {
	tr := treinoOval(t)
	ctl := &alterna{}
	c := tr.Largada(0)
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
