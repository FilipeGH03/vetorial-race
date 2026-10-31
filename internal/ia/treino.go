package ia

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"

	"vetorial-race/internal/sim"
)

// Avaliação de uma rede no treino (D5, D21, D22): o indivíduo dirige sozinho
// NLargadas vezes, uma de cada posição de sim.Largada(p, NLargadas), com a
// Config da corrida (BaterMata = false: bater zera a velocidade e devolve o
// carro à última posição válida), cada vez até TempoMaxTreino ou até passar
// TicksParado ticks sem a Distancia aumentar; a nota é a média das NLargadas
// notas. A simulação não sorteia nada, então a nota só depende da rede e da
// pista: avaliar em goroutines dá o mesmo resultado que em série.

// Parâmetros da pontuação do treino (D21).
const (
	TempoMaxTreino = 60.0   // s que cada indivíduo dirige, no máximo (D5)
	TicksParado    = 60     // ticks (1 s) sem a Distancia aumentar que encerram o indivíduo (D21)
	BonusVolta     = 1000.0 // pontos por volta completa
	BonusTempo     = 20.0   // pontos por segundo que sobram de TempoMaxTreino quando completa a 1ª volta
	PenaBatida     = 5.0    // pontos tirados por batida na parede (D21)
	NLargadas      = 6      // posições de sim.Largada(p, NLargadas) de que cada indivíduo larga (D22)
)

// ErrTreino é devolvido por NovoTreino quando a pista não serve para treinar.
var ErrTreino = errors.New("treino inválido")

// Resultado é o que um indivíduo fez na pista partindo de uma das largadas.
type Resultado struct {
	Pontuacao  float64 // Pontuacao(Distancia, Voltas, TempoVolta, Batidas)
	Distancia  float64 // Progresso.Distancia final (px)
	Voltas     int     // voltas completas
	TempoVolta float64 // s até completar a 1ª volta; 0 se não completou
	Batidas    int     // ticks em que bateu na parede
	Ticks      int     // ticks que dirigiu
	Parou      bool    // encerrado por TicksParado sem avançar (senão, acabou o tempo)
}

// Pontuacao calcula a nota do treino (D21): Distancia + BonusVolta·voltas +
// BonusTempo·(TempoMaxTreino − tempo da 1ª volta) − PenaBatida·batidas. O
// bônus de tempo só vale quando completa a volta (tempo menor vale mais).
func Pontuacao(distancia float64, voltas int, tempoVolta float64, batidas int) float64 {
	p := distancia + BonusVolta*float64(voltas) - PenaBatida*float64(batidas)
	if voltas >= 1 {
		p += BonusTempo * max(0, TempoMaxTreino-tempoVolta)
	}
	return p
}

// Avaliacao é o que um indivíduo fez nas NLargadas largadas (D22).
type Avaliacao struct {
	Pontuacao float64              // média das Pontuacao de Largadas: a nota do indivíduo
	Largadas  [NLargadas]Resultado // Largadas[k]: partindo de sim.Largada(p, NLargadas)[k]
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
	Cfg      sim.Config // ConfigPadrao, BaterMata = false como na corrida (D21)
	MaxTicks int        // TempoMaxTreino em ticks

	largadas [NLargadas]sim.Carro // sim.Largada(p, NLargadas), calculada uma vez (D22)
}

// NovoTreino prepara o treino na pista p: a Config da corrida (D21) e as
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
// até MaxTicks, ou até passar TicksParado ticks sem a Distancia aumentar
// (D21), e devolve o que ela fez. Cada tick: Ler, Decidir (uma vez) e Avancar,
// como na Corrida; cada tick com Batido conta uma batida. Não altera r.
func (t *Treino) RodarDe(r *Rede, k int) Resultado {
	c := t.largadas[k] // cópia
	ctl := NovoControlador(r)
	res := Resultado{}
	ultima, parado := c.Progresso.Distancia, 0
	for res.Ticks < t.MaxTicks && c.Vivo {
		c.Avancar(ctl.Decidir(sim.Ler(&c, t.Pista, t.Cfg)), t.Pista, t.Cfg)
		res.Ticks++
		if c.Batido {
			res.Batidas++
		}
		if res.TempoVolta == 0 && c.Progresso.Voltas >= 1 {
			res.TempoVolta = float64(res.Ticks) * t.Cfg.DT
		}
		if c.Progresso.Distancia > ultima {
			ultima, parado = c.Progresso.Distancia, 0
			continue
		}
		parado++
		if parado >= TicksParado {
			res.Parou = true
			break
		}
	}
	res.Distancia = c.Progresso.Distancia
	res.Voltas = c.Progresso.Voltas
	res.Pontuacao = Pontuacao(res.Distancia, res.Voltas, res.TempoVolta, res.Batidas)
	return res
}

// AvaliarPopulacao roda todas as redes da população em até workers goroutines
// (workers ≤ 0 usa GOMAXPROCS), guarda as notas com Definir e devolve o
// Avaliacao de cada rede, na ordem de pop.Redes. As NLargadas corridas de cada
// indivíduo também são repartidas entre os workers; nenhuma divide estado com
// outra, então o resultado não depende de workers.
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

	notas := make([]float64, len(res))
	for i := range res {
		soma := 0.0 // mesma ordem de soma que Rodar: mesma nota, bit a bit
		for _, r := range res[i].Largadas {
			soma += r.Pontuacao
		}
		res[i].Pontuacao = soma / NLargadas
		notas[i] = res[i].Pontuacao
	}
	if err := pop.Definir(notas); err != nil {
		return nil, err
	}
	return res, nil
}
