package sim

import (
	"errors"
	"math"
	"sort"
)

// Modo é o tipo de partida.
type Modo int8

const (
	// ModoVoltas: vence quem completa Corrida.Voltas voltas primeiro.
	ModoVoltas Modo = iota
	// ModoEliminacao: um laser anda pela linha central e mata quem ficar para
	// trás (D6); o último vivo vence.
	ModoEliminacao
)

// O laser da Eliminação (D6): é um ponto que anda pela linha central a partir
// da largada, LaserAtraso segundos depois do início, com velocidade que começa
// em LaserVelInicial e sobe LaserAceleracao px/s a cada segundo depois que sai.
const (
	LaserAtraso     = 3.0   // s desde o início da corrida até o laser sair
	LaserVelInicial = 120.0 // px/s quando sai
	LaserAceleracao = 8.0   // px/s a cada segundo depois que sai
)

// ErrCorridaInvalida é o erro de NovaCorrida para argumentos que não formam uma
// partida (pista nil ou sem largada, nenhum controlador, modo ou voltas inválidos).
var ErrCorridaInvalida = errors.New("corrida inválida")

// Corrida é uma partida: carros e quem os controla, rodando passo a passo na
// mesma simulação do treino. Nada nela depende do relógio de parede nem de
// sorteio: com os mesmos controladores o resultado é o mesmo.
type Corrida struct {
	Pista  *Pista
	Cfg    Config
	Modo   Modo
	Voltas int // meta do ModoVoltas (ignorada na Eliminação)

	Carros        []Carro       // Carros[i] é controlado por Controladores[i]
	Controladores []Controlador // nil = carro sem controle (não mexe em nada)

	Tick      int  // ticks já rodados; o tempo é Tick*Cfg.DT
	Terminada bool // Step não faz mais nada
	Vencedor  int  // índice em Carros; -1 se não há vencedor (ainda em jogo ou ninguém)

	chegada []int // índices de quem completou a meta, em ordem de chegada (Voltas)
	morte   []int // tick em que cada carro deixou de estar vivo; -1 se vivo
}

// NovaCorrida monta uma partida com um carro por controlador, postos por
// Largada(p, len(ctls)) (D18: uma vez, no início). Controlador nil vale como
// carro parado. Em ModoVoltas, voltas ≥ 1 é a meta; na Eliminação voltas é
// ignorado. cfg é usado como veio (corrida: BaterMata = false).
func NovaCorrida(p *Pista, cfg Config, modo Modo, voltas int, ctls []Controlador) (*Corrida, error) {
	if p == nil || len(ctls) == 0 || cfg.DT <= 0 {
		return nil, ErrCorridaInvalida
	}
	if modo != ModoVoltas && modo != ModoEliminacao {
		return nil, ErrCorridaInvalida
	}
	if modo == ModoVoltas && voltas < 1 {
		return nil, ErrCorridaInvalida
	}
	carros := Largada(p, len(ctls))
	if len(carros) != len(ctls) {
		return nil, ErrCorridaInvalida
	}
	r := &Corrida{
		Pista:         p,
		Cfg:           cfg,
		Modo:          modo,
		Voltas:        voltas,
		Carros:        carros,
		Controladores: append([]Controlador(nil), ctls...),
		Vencedor:      -1,
		morte:         make([]int, len(carros)),
	}
	for i := range r.morte {
		r.morte[i] = -1
	}
	return r, nil
}

// Tempo é o tempo de corrida em segundos.
func (r *Corrida) Tempo() float64 { return float64(r.Tick) * r.Cfg.DT }

// Step roda um tick: cada carro vivo recebe a Leitura, o Controlador decide e o
// carro faz Avancar (com colisão e progresso, nunca Mover). Depois, no modo
// Voltas, quem completou a meta é registrado em ordem de chegada (empate no
// mesmo tick: quem passou mais da linha primeiro) e a corrida termina com o
// primeiro deles como vencedor; na Eliminação o laser mata quem tem Distancia
// ≤ à dele e a corrida termina quando sobra no máximo um vivo. Depois de
// Terminada, Step não faz nada (os carros ficam congelados).
func (r *Corrida) Step() {
	if r.Terminada {
		return
	}
	r.Tick++
	for i := range r.Carros {
		c := &r.Carros[i]
		if !c.Vivo {
			continue
		}
		var ctl Controle
		if i < len(r.Controladores) && r.Controladores[i] != nil {
			ctl = r.Controladores[i].Decidir(Ler(c, r.Pista, r.Cfg))
		}
		c.Avancar(ctl, r.Pista, r.Cfg)
	}
	if r.Modo == ModoVoltas {
		r.fimDoTickVoltas()
	} else {
		r.fimDoTickEliminacao()
	}
}

// marcarMortes anota o tick de morte de quem deixou de estar vivo neste tick
// (por parede com BaterMata ou pelo laser) e devolve quantos estão vivos e o
// índice do último vivo.
func (r *Corrida) marcarMortes() (vivos, ultimo int) {
	ultimo = -1
	for i := range r.Carros {
		if r.Carros[i].Vivo {
			vivos++
			ultimo = i
		} else if r.morte[i] < 0 {
			r.morte[i] = r.Tick
		}
	}
	return vivos, ultimo
}

// fimDoTickVoltas registra quem terminou neste tick e fecha a corrida.
func (r *Corrida) fimDoTickVoltas() {
	var novos []int
	for i := range r.Carros {
		if c := &r.Carros[i]; c.Vivo && c.Progresso.Voltas >= r.Voltas {
			novos = append(novos, i)
		}
	}
	vivos, _ := r.marcarMortes()
	if len(novos) > 0 {
		sort.SliceStable(novos, func(a, b int) bool {
			return r.Carros[novos[a]].Progresso.MaisAdiante(r.Carros[novos[b]].Progresso)
		})
		r.chegada = append(r.chegada, novos...)
		r.Vencedor = r.chegada[0]
		r.Terminada = true
	} else if vivos == 0 {
		r.Terminada = true // ninguém vivo e ninguém terminou: sem vencedor
	}
}

// fimDoTickEliminacao aplica o laser e decide se a corrida acabou.
func (r *Corrida) fimDoTickEliminacao() {
	if d, ativo := r.Laser(); ativo {
		for i := range r.Carros {
			if c := &r.Carros[i]; c.Vivo && c.Progresso.Distancia <= d {
				c.Vivo = false
				c.Vel = Vec2{}
			}
		}
	}
	vivos, ultimo := r.marcarMortes()
	switch {
	case vivos == 1 && len(r.Carros) > 1:
		r.Vencedor, r.Terminada = ultimo, true
	case vivos == 0:
		// Os últimos morreram juntos: vence quem estava mais adiante.
		r.Terminada = true
		melhor := -1
		for i := range r.Carros {
			if r.morte[i] != r.Tick {
				continue
			}
			if melhor < 0 || r.Carros[i].Progresso.MaisAdiante(r.Carros[melhor].Progresso) {
				melhor = i
			}
		}
		r.Vencedor = melhor
	}
}

// Laser devolve a Distancia do laser da Eliminação e se ele já saiu (D6). É
// função só do Tick: depois de LaserAtraso s, d = v0·τ + a·τ²/2 com τ os
// segundos desde que saiu. Fora do ModoEliminacao nunca está ativo.
func (r *Corrida) Laser() (distancia float64, ativo bool) {
	if r.Modo != ModoEliminacao {
		return 0, false
	}
	atraso := int(math.Round(LaserAtraso / r.Cfg.DT))
	if r.Tick < atraso {
		return 0, false
	}
	tau := float64(r.Tick-atraso) * r.Cfg.DT
	return LaserVelInicial*tau + LaserAceleracao*tau*tau/2, true
}

// LaserPos é o ponto da linha central onde o laser está (antes de sair, é a
// largada).
func (r *Corrida) LaserPos() Vec2 {
	d, _ := r.Laser()
	j, f := posicaoNaLinha(r.Pista, d)
	if j < 0 {
		return Vec2{}
	}
	n := len(r.Pista.Centro)
	a, b := r.Pista.Centro[j], r.Pista.Centro[(j+1)%n]
	return a.Add(b.Sub(a).Scale(f))
}

// LaserFaixa é o segmento de parede a parede onde o laser está, para desenhar
// a faixa vermelha atravessando a pista: interpola os checkpoints vizinhos.
func (r *Corrida) LaserFaixa() Segmento {
	d, _ := r.Laser()
	j, f := posicaoNaLinha(r.Pista, d)
	if j < 0 {
		return Segmento{}
	}
	cps := r.Pista.Checkpoints
	a, b := cps[j], cps[(j+1)%len(cps)]
	return Segmento{
		A: a.A.Add(b.A.Sub(a.A).Scale(f)),
		B: a.B.Add(b.B.Sub(a.B).Scale(f)),
	}
}

// posicaoNaLinha converte uma Distancia (somando voltas) no trecho j da linha
// central e na fração f de 0 a 1 dele. j = -1 se a pista não tem linha.
func posicaoNaLinha(p *Pista, d float64) (j int, f float64) {
	if p == nil || len(p.Centro) == 0 || len(p.Checkpoints) != len(p.Centro) || p.Passo <= 0 || p.Comprimento <= 0 {
		return -1, 0
	}
	m := math.Mod(max(d, 0), p.Comprimento)
	j = min(int(m/p.Passo), len(p.Centro)-1)
	return j, max(0, min(1, (m-float64(j)*p.Passo)/p.Passo))
}

// Chegada são os índices de quem completou a meta, em ordem de chegada (só
// ModoVoltas). O slice devolvido é uma cópia.
func (r *Corrida) Chegada() []int { return append([]int(nil), r.chegada...) }

// Classificacao devolve os índices dos carros do primeiro ao último, a qualquer
// momento da corrida. Classificar (A4) é só progresso bruto; aqui:
//   - Voltas: quem completou a meta vem primeiro, em ordem de chegada; o resto
//     por progresso (Voltas e Distancia), estável.
//   - Eliminação: os vivos vêm acima dos eliminados, por progresso; os
//     eliminados, do que morreu mais tarde ao que morreu primeiro (empate no
//     mesmo tick: por progresso), estável.
func (r *Corrida) Classificacao() []int {
	n := len(r.Carros)
	ordem := make([]int, 0, n)
	feito := make([]bool, n)
	if r.Modo == ModoVoltas {
		for _, i := range r.chegada {
			ordem = append(ordem, i)
			feito[i] = true
		}
	}
	var resto []int
	for i := 0; i < n; i++ {
		if !feito[i] {
			resto = append(resto, i)
		}
	}
	elim := r.Modo == ModoEliminacao
	sort.SliceStable(resto, func(a, b int) bool {
		i, j := resto[a], resto[b]
		if elim {
			if vi, vj := r.Carros[i].Vivo, r.Carros[j].Vivo; vi != vj {
				return vi // vivo acima de eliminado
			}
			if !r.Carros[i].Vivo && r.morte[i] != r.morte[j] {
				return r.morte[i] > r.morte[j]
			}
		}
		return r.Carros[i].Progresso.MaisAdiante(r.Carros[j].Progresso)
	})
	return append(ordem, resto...)
}
