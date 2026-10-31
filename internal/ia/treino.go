package ia

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"

	"vetorial-race/internal/sim"
)

// Avaliação de uma rede no treino (D5): o indivíduo larga sozinho de
// sim.Largada(p, 1)[0] (D18) e dirige com BaterMata = true (D2) até bater ou
// até TempoMaxTreino. A simulação não sorteia nada, então a nota só depende da
// rede e da pista: avaliar em goroutines dá o mesmo resultado que em série.

// Parâmetros da pontuação do treino.
const (
	TempoMaxTreino = 60.0   // s que cada indivíduo dirige, no máximo (D5)
	BonusVolta     = 1000.0 // pontos por volta completa
	BonusTempo     = 20.0   // pontos por segundo que sobram de TempoMaxTreino quando completa a 1ª volta
)

// ErrTreino é devolvido por NovoTreino quando a pista não serve para treinar.
var ErrTreino = errors.New("treino inválido")

// Resultado é o que um indivíduo fez na pista.
type Resultado struct {
	Pontuacao  float64 // Distancia + BonusVolta·Voltas + BonusTempo·(TempoMaxTreino - TempoVolta)
	Distancia  float64 // Progresso.Distancia final (px)
	Voltas     int     // voltas completas
	TempoVolta float64 // s até completar a 1ª volta; 0 se não completou
	Ticks      int     // ticks que dirigiu
	Bateu      bool    // terminou batendo (senão, acabou o tempo)
}

// Pontuacao calcula a nota do treino (D5): a Distancia, mais BonusVolta por
// volta completa, mais BonusTempo por segundo poupado na 1ª volta (tempo menor
// vale mais). Quem não cruza a largada fica com 0.
func Pontuacao(distancia float64, voltas int, tempoVolta float64) float64 {
	p := distancia + BonusVolta*float64(voltas)
	if voltas >= 1 {
		p += BonusTempo * max(0, TempoMaxTreino-tempoVolta)
	}
	return p
}

// Treino roda os indivíduos de uma pista. Só lê a pista: o mesmo Treino pode
// avaliar várias redes ao mesmo tempo.
type Treino struct {
	Pista    *sim.Pista
	Cfg      sim.Config // ConfigPadrao com BaterMata = true (D2, D13)
	MaxTicks int        // TempoMaxTreino em ticks

	largada sim.Carro // sim.Largada(p, 1)[0], calculada uma vez (D18)
}

// NovoTreino prepara o treino na pista p: Config do jogo com BaterMata e a
// largada de um carro só, calculada uma vez e copiada para cada indivíduo.
func NovoTreino(p *sim.Pista) (*Treino, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: pista nil", ErrTreino)
	}
	carros := sim.Largada(p, 1)
	if len(carros) != 1 {
		return nil, fmt.Errorf("%w: pista %q sem largada", ErrTreino, p.Nome)
	}
	cfg := sim.ConfigPadrao()
	cfg.BaterMata = true
	return &Treino{
		Pista:    p,
		Cfg:      cfg,
		MaxTicks: int(math.Round(TempoMaxTreino / cfg.DT)),
		largada:  carros[0],
	}, nil
}

// Rodar põe a rede r para dirigir um carro da largada até bater ou até
// MaxTicks e devolve o que ela fez. Cada tick: Ler, Decidir (uma vez) e
// Avancar, como na Corrida. Não altera r.
func (t *Treino) Rodar(r *Rede) Resultado {
	c := t.largada // cópia
	ctl := NovoControlador(r)
	res := Resultado{}
	for res.Ticks < t.MaxTicks && c.Vivo {
		c.Avancar(ctl.Decidir(sim.Ler(&c, t.Pista, t.Cfg)), t.Pista, t.Cfg)
		res.Ticks++
		if res.TempoVolta == 0 && c.Progresso.Voltas >= 1 {
			res.TempoVolta = float64(res.Ticks) * t.Cfg.DT
		}
	}
	res.Bateu = !c.Vivo
	res.Distancia = c.Progresso.Distancia
	res.Voltas = c.Progresso.Voltas
	res.Pontuacao = Pontuacao(res.Distancia, res.Voltas, res.TempoVolta)
	return res
}

// AvaliarPopulacao roda todas as redes da população em até workers goroutines
// (workers ≤ 0 usa GOMAXPROCS), guarda as notas com Definir e devolve o
// Resultado de cada rede, na ordem de pop.Redes. Os indivíduos não dividem
// estado, então o resultado não depende de workers.
func (t *Treino) AvaliarPopulacao(pop *Populacao, workers int) ([]Resultado, error) {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	res := make([]Resultado, len(pop.Redes))
	fila := make(chan int)
	var wg sync.WaitGroup
	for range min(workers, len(pop.Redes)) {
		wg.Go(func() {
			for i := range fila {
				res[i] = t.Rodar(pop.Redes[i])
			}
		})
	}
	for i := range pop.Redes {
		fila <- i
	}
	close(fila)
	wg.Wait()

	notas := make([]float64, len(res))
	for i, r := range res {
		notas[i] = r.Pontuacao
	}
	if err := pop.Definir(notas); err != nil {
		return nil, err
	}
	return res, nil
}
