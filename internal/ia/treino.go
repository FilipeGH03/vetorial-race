package ia

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"

	"vetorial-race/internal/sim"
)

// Avaliação de uma rede no treino (D5, D22, D23): o indivíduo dirige sozinho
// NLargadas vezes, uma de cada posição de sim.Largada(p, NLargadas), com a
// Config da corrida (BaterMata = false: bater devolve o carro ao começo do
// tick e o faz deslizar na parede com metade da velocidade ao longo dela, D28),
// cada vez até TempoMaxTreino ou até passar
// TicksParado ticks sem avançar no sentido da pista, medido por
// sim.AvancoNaLinha (conta também antes de cruzar a largada); a nota é a
// média das NLargadas notas. A simulação não sorteia nada, então a nota só depende da rede e da
// pista: avaliar em goroutines dá o mesmo resultado que em série.

// Parâmetros da pontuação do treino (D23, D26, D27).
const (
	TempoMaxTreino = 60.0   // s que cada indivíduo dirige, no máximo (D5)
	TicksParado    = 60     // ticks (1 s) sem avançar no sentido da pista (sim.AvancoNaLinha) que encerram o indivíduo (D23)
	BonusVolta     = 1000.0 // pontos por volta completa
	BonusTempo     = 20.0   // pontos por segundo que sobram de TempoMaxTreino quando completa a 1ª volta
	PenaBatidaNova = 300.0  // pontos tirados por batida nova: tick batido depois de um tick sem bater, ou o 1º tick (D26)
	PenaBatida     = 50.0   // pontos tirados por tick com batida na parede (D25, mantida pela D26)
	NLargadas      = 6      // posições de sim.Largada(p, NLargadas) de que cada indivíduo larga (D22)
)

// Rampa das penas de batida no treino (D27): até a geração RampaInicio − 1 o
// treino usa PenasLeves (as da D23); de RampaInicio a RampaFim − 1 as duas
// penas sobem linearmente até PenasCheias; de RampaFim em diante, a D26 cheia.
const (
	RampaInicio  = 100 // 1ª geração em que as penas começam a subir
	RampaFim     = 300 // 1ª geração com as penas cheias (D26)
	PenaTickLeve = 5.0 // pena por tick batido antes da rampa (D23)
	PenaNovaLeve = 0.0 // pena por batida nova antes da rampa (D23 não tinha)
)

// Penas são os pontos tirados por batida na nota do treino.
type Penas struct {
	BatidaNova float64 // por batida nova (D26)
	Tick       float64 // por tick batido
}

var (
	PenasLeves  = Penas{BatidaNova: PenaNovaLeve, Tick: PenaTickLeve} // gerações 0 .. RampaInicio−1 (D27)
	PenasCheias = Penas{BatidaNova: PenaBatidaNova, Tick: PenaBatida} // D26: a nota gravada no cérebro
)

// PenasDaGeracao devolve as penas que o treino aplica na geração g (D27):
// PenasLeves antes de RampaInicio, PenasCheias de RampaFim em diante e, entre
// as duas, cada pena na reta que liga uma à outra.
func PenasDaGeracao(g int) Penas {
	switch {
	case g < RampaInicio:
		return PenasLeves
	case g >= RampaFim:
		return PenasCheias
	}
	f := float64(g-RampaInicio) / float64(RampaFim-RampaInicio)
	return Penas{
		BatidaNova: PenasLeves.BatidaNova + f*(PenasCheias.BatidaNova-PenasLeves.BatidaNova),
		Tick:       PenasLeves.Tick + f*(PenasCheias.Tick-PenasLeves.Tick),
	}
}

// Nota é a fórmula da D26 com as penas pn no lugar das cheias: Distancia +
// BonusVolta·voltas + BonusTempo·(TempoMaxTreino − tempo da 1ª volta) −
// pn.BatidaNova·batidas novas − pn.Tick·ticks batidos.
func (pn Penas) Nota(distancia float64, voltas int, tempoVolta float64, batidasNovas, ticksBatidos int) float64 {
	p := distancia + BonusVolta*float64(voltas) -
		pn.BatidaNova*float64(batidasNovas) - pn.Tick*float64(ticksBatidos)
	if voltas >= 1 {
		p += BonusTempo * max(0, TempoMaxTreino-tempoVolta)
	}
	return p
}

// ErrTreino é devolvido por NovoTreino quando a pista não serve para treinar.
var ErrTreino = errors.New("treino inválido")

// Resultado é o que um indivíduo fez na pista partindo de uma das largadas.
type Resultado struct {
	Pontuacao    float64 // Pontuacao(Distancia, Voltas, TempoVolta, BatidasNovas, Batidas)
	Distancia    float64 // Progresso.Distancia final (px)
	Voltas       int     // voltas completas
	TempoVolta   float64 // s até completar a 1ª volta; 0 se não completou
	Batidas      int     // ticks em que bateu na parede
	BatidasNovas int     // batidas novas (D26): ticks batidos logo depois de um tick sem bater (ou no 1º tick)
	Ticks        int     // ticks que dirigiu
	Parou        bool    // encerrado por TicksParado sem avançar (senão, acabou o tempo)
}

// Pontuacao calcula a nota do treino (D26): Distancia + BonusVolta·voltas +
// BonusTempo·(TempoMaxTreino − tempo da 1ª volta) − PenaBatidaNova·batidas
// novas − PenaBatida·ticks batidos. O bônus de tempo só vale quando completa a
// volta (tempo menor vale mais). Cada batida custa PenaBatidaNova uma vez, mais
// PenaBatida por tick batido (o 1º incluído) enquanto o carro fica na parede.
//
// É a nota gravada no cérebro; o treino seleciona com PenasDaGeracao (D27).
func Pontuacao(distancia float64, voltas int, tempoVolta float64, batidasNovas, ticksBatidos int) float64 {
	return PenasCheias.Nota(distancia, voltas, tempoVolta, batidasNovas, ticksBatidos)
}

// Nota é a nota deste resultado com as penas pn (D27); com PenasCheias é a
// Pontuacao.
func (r Resultado) Nota(pn Penas) float64 {
	return pn.Nota(r.Distancia, r.Voltas, r.TempoVolta, r.BatidasNovas, r.Batidas)
}

// Avaliacao é o que um indivíduo fez nas NLargadas largadas (D22).
type Avaliacao struct {
	Pontuacao float64              // média das Pontuacao de Largadas: a nota do indivíduo (D26, a gravada no cérebro)
	Largadas  [NLargadas]Resultado // Largadas[k]: partindo de sim.Largada(p, NLargadas)[k]
}

// Nota é a média das notas das largadas com as penas pn (D27), somadas na
// mesma ordem que Pontuacao: com PenasCheias dá a Pontuacao, bit a bit.
func (a Avaliacao) Nota(pn Penas) float64 {
	soma := 0.0
	for _, r := range a.Largadas {
		soma += r.Nota(pn)
	}
	return soma / NLargadas
}

// VoltasMin é o menor número de voltas entre as largadas: ≥ 1 quando o
// indivíduo completou a volta de todas as posições.
func (a Avaliacao) VoltasMin() int {
	v := a.Largadas[0].Voltas
	for _, r := range a.Largadas[1:] {
		v = min(v, r.Voltas)
	}
	return v
}

// ComVolta conta de quantas largadas o indivíduo completou ao menos 1 volta.
func (a Avaliacao) ComVolta() int {
	n := 0
	for _, r := range a.Largadas {
		if r.Voltas >= 1 {
			n++
		}
	}
	return n
}

// Treino roda os indivíduos de uma pista. Só lê a pista: o mesmo Treino pode
// avaliar várias redes ao mesmo tempo.
type Treino struct {
	Pista    *sim.Pista
	Cfg      sim.Config // ConfigPadrao, BaterMata = false como na corrida (D23)
	MaxTicks int        // TempoMaxTreino em ticks

	largadas [NLargadas]sim.Carro // sim.Largada(p, NLargadas), calculada uma vez (D22)
}

// NovoTreino prepara o treino na pista p: a Config da corrida (D23) e as
// NLargadas posições de sim.Largada(p, NLargadas), calculadas uma vez e
// copiadas para cada indivíduo (D22).
func NovoTreino(p *sim.Pista) (*Treino, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: pista nil", ErrTreino)
	}
	carros := sim.Largada(p, NLargadas)
	if len(carros) != NLargadas {
		return nil, fmt.Errorf("%w: pista %q sem largada", ErrTreino, p.Nome)
	}
	cfg := sim.ConfigPadrao()
	cfg.BaterMata = false
	t := &Treino{
		Pista:    p,
		Cfg:      cfg,
		MaxTicks: int(math.Round(TempoMaxTreino / cfg.DT)),
	}
	copy(t.largadas[:], carros)
	return t, nil
}

// Largada devolve a posição k (0 ≤ k < NLargadas) de onde o treino larga: uma
// cópia de sim.Largada(p, NLargadas)[k].
func (t *Treino) Largada(k int) sim.Carro {
	return t.largadas[k]
}

// Rodar põe a rede r para dirigir de cada uma das NLargadas largadas (D22) e
// devolve o que ela fez em cada uma, com a nota = média das notas. Não altera r.
func (t *Treino) Rodar(r *Rede) Avaliacao {
	var a Avaliacao
	soma := 0.0
	for k := range a.Largadas {
		a.Largadas[k] = t.RodarDe(r, k)
		soma += a.Largadas[k].Pontuacao
	}
	a.Pontuacao = soma / NLargadas
	return a
}

// RodarDe põe a rede r para dirigir um carro da largada k (0 ≤ k < NLargadas)
// até MaxTicks, ou até passar TicksParado ticks sem avançar no sentido da
// pista (D23), e devolve o que ela fez. Cada tick: Ler, Decidir (uma vez) e
// Avancar, como na Corrida; cada tick com Batido conta uma batida, e a batida é
// nova quando o tick anterior não tinha Batido (D26). Não altera r.
// O fim por ficar parado é o de Avanco.
func (t *Treino) RodarDe(r *Rede, k int) Resultado {
	c := t.largadas[k] // cópia
	ctl := NovoControlador(r)
	res := Resultado{}
	var av Avanco
	for res.Ticks < t.MaxTicks && c.Vivo {
		if t.Passo(&c, ctl, &av, &res) {
			break
		}
	}
	res.Distancia = c.Progresso.Distancia
	res.Voltas = c.Progresso.Voltas
	res.Pontuacao = Pontuacao(res.Distancia, res.Voltas, res.TempoVolta, res.BatidasNovas, res.Batidas)
	return res
}

// Passo é o tick do treino, a regra num lugar só (usada por RodarDe e pela tela
// de treino): Ler, Decidir (uma vez) e Avancar do carro c; conta o tick, a
// batida (e se é nova, D26), o tempo da 1ª volta e o "1 s sem avançar" (av) em res, e mantém
// res.Distancia/Voltas em dia. Devolve true quando o indivíduo terminou (parado,
// MaxTicks ou carro morto); nesse tick res.Pontuacao também é preenchida.
func (t *Treino) Passo(c *sim.Carro, ctl *Controlador, av *Avanco, res *Resultado) bool {
	antes, batidoAntes := c.Pos, c.Batido // Batido do tick anterior (false na largada)
	c.Avancar(ctl.Decidir(sim.Ler(c, t.Pista, t.Cfg)), t.Pista, t.Cfg)
	res.Ticks++
	if c.Batido {
		res.Batidas++
		if !batidoAntes {
			res.BatidasNovas++
		}
	}
	if res.TempoVolta == 0 && c.Progresso.Voltas >= 1 {
		res.TempoVolta = float64(res.Ticks) * t.Cfg.DT
	}
	fim := false
	if av.Tick(t.Pista, antes, c.Pos) {
		res.Parou = true
		fim = true
	}
	if res.Ticks >= t.MaxTicks || !c.Vivo {
		fim = true
	}
	res.Distancia, res.Voltas = c.Progresso.Distancia, c.Progresso.Voltas
	if fim {
		res.Pontuacao = Pontuacao(res.Distancia, res.Voltas, res.TempoVolta, res.BatidasNovas, res.Batidas)
	}
	return fim
}

// Avanco aplica a um carro a regra do treino de "1 s sem avançar" (D23).
// "Avançar" é passar do ponto mais adiante já alcançado na linha central,
// medido pela soma de sim.AvancoNaLinha(p, posição antes, posição depois) de
// cada tick desde a largada. Assim vale também atrás da largada (onde a
// Distancia ainda é 0), e ir e voltar no mesmo lugar não conta como avanço.
// O valor zero é o de um carro na largada.
type Avanco struct {
	linha       float64 // soma dos avanços na linha central desde a largada (px)
	maisAdiante float64 // maior valor de linha até agora
	parado      int     // ticks seguidos sem passar de maisAdiante
}

// Tick registra o tick em que o carro foi de "de" para "ate" e diz se ele deve
// ser encerrado: TicksParado ticks seguidos sem avançar.
func (a *Avanco) Tick(p *sim.Pista, de, ate sim.Vec2) bool {
	if ate != de {
		a.linha += sim.AvancoNaLinha(p, de, ate)
	}
	if a.linha > a.maisAdiante {
		a.maisAdiante, a.parado = a.linha, 0
		return false
	}
	a.parado++
	return a.parado >= TicksParado
}

// AvaliarPopulacao roda todas as redes da população, elite inclusive, em até
// workers goroutines (workers ≤ 0 usa GOMAXPROCS) e devolve o Avaliacao de cada
// rede, na ordem de pop.Redes. A nota de seleção, guardada com Definir em
// pop.Pontuacoes, usa as penas da geração, PenasDaGeracao(pop.Geracao) (D27);
// Rede.Pontuacao fica com a Pontuacao (fórmula cheia, D26), a que vai para o
// arquivo do cérebro. As NLargadas corridas de cada indivíduo também são
// repartidas entre os workers; nenhuma divide estado com outra, então o
// resultado não depende de workers.
func (t *Treino) AvaliarPopulacao(pop *Populacao, workers int) ([]Avaliacao, error) {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	res := make([]Avaliacao, len(pop.Redes))
	total := len(pop.Redes) * NLargadas
	fila := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, total) {
		wg.Go(func() {
			for j := range fila {
				i, k := j/NLargadas, j%NLargadas
				res[i].Largadas[k] = t.RodarDe(pop.Redes[i], k)
			}
		})
	}
	for j := range total {
		fila <- j
	}
	close(fila)
	wg.Wait()

	pn := PenasDaGeracao(pop.Geracao)
	notas := make([]float64, len(res))
	for i := range res {
		soma := 0.0 // mesma ordem de soma que Rodar: mesma nota, bit a bit
		for _, r := range res[i].Largadas {
			soma += r.Pontuacao
		}
		res[i].Pontuacao = soma / NLargadas
		notas[i] = res[i].Nota(pn)
	}
	if err := pop.Definir(notas); err != nil {
		return nil, err
	}
	for i, r := range pop.Redes {
		r.Pontuacao = res[i].Pontuacao // D27: o cérebro guarda a fórmula cheia
	}
	return res, nil
}
