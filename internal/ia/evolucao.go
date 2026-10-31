package ia

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

// Neuroevolução (D5): algoritmo genético sobre os pesos e vieses da rede.
// Não usa gradiente. Tudo sorteia de um único rand.Rand semeado por
// Params.Seed, então a mesma seed e a mesma função de pontuação dão sempre as
// mesmas redes. O pacote não conhece a simulação: quem avalia (C3) entrega
// uma pontuação por rede.

// ErrParams é devolvido quando os parâmetros da evolução não fazem sentido.
var ErrParams = errors.New("parâmetros de evolução inválidos")

// Params são os ajustes da evolução.
type Params struct {
	Camadas     []int   // forma das redes (padrão 14-12-2)
	Tamanho     int     // indivíduos por geração (padrão 50)
	Elite       int     // os N melhores passam iguais (padrão 5)
	Torneio     int     // candidatos sorteados por torneio (padrão 3)
	TaxaMutacao float64 // chance de cada peso/viés mutar, em [0,1] (padrão 0.1)
	Sigma       float64 // desvio da mutação gaussiana (padrão 0.3)
	Seed        uint64  // seed do sorteio (padrão 1)
}

// ParamsPadrao devolve os valores da D5.
func ParamsPadrao() Params {
	return Params{
		Camadas:     append([]int(nil), CamadasPadrao...),
		Tamanho:     50,
		Elite:       5,
		Torneio:     3,
		TaxaMutacao: 0.1,
		Sigma:       0.3,
		Seed:        1,
	}
}

func (p Params) validar() error {
	if err := validarCamadas(p.Camadas); err != nil {
		return err
	}
	switch {
	case p.Tamanho < 2:
		return fmt.Errorf("%w: tamanho %d (mínimo 2)", ErrParams, p.Tamanho)
	case p.Elite < 0 || p.Elite >= p.Tamanho:
		return fmt.Errorf("%w: elite %d com população de %d", ErrParams, p.Elite, p.Tamanho)
	case p.Torneio < 1:
		return fmt.Errorf("%w: torneio com %d candidatos", ErrParams, p.Torneio)
	case !(p.TaxaMutacao >= 0 && p.TaxaMutacao <= 1):
		return fmt.Errorf("%w: taxa de mutação %v fora de [0,1]", ErrParams, p.TaxaMutacao)
	case !(p.Sigma >= 0) || math.IsInf(p.Sigma, 0):
		return fmt.Errorf("%w: sigma %v", ErrParams, p.Sigma)
	}
	return nil
}

// Populacao é uma geração de redes com as pontuações da última avaliação.
type Populacao struct {
	Params     Params
	Redes      []*Rede
	Pontuacoes []float64 // Pontuacoes[i] é a nota de Redes[i]; vale depois de Avaliar/Definir
	Geracao    int       // 0 é a população inicial

	rng *rand.Rand
}

// NovaPopulacao sorteia a geração 0 (pesos Xavier) com a seed dos parâmetros.
func NovaPopulacao(p Params) (*Populacao, error) {
	p.Camadas = append([]int(nil), p.Camadas...)
	if err := p.validar(); err != nil {
		return nil, err
	}
	pop := &Populacao{
		Params:     p,
		Redes:      make([]*Rede, p.Tamanho),
		Pontuacoes: make([]float64, p.Tamanho),
		rng:        rand.New(rand.NewPCG(p.Seed, p.Seed^0x9e3779b97f4a7c15)),
	}
	for i := range pop.Redes {
		r, err := NovaRede(p.Camadas, pop.rng)
		if err != nil {
			return nil, err
		}
		pop.Redes[i] = r
	}
	return pop, nil
}

// Avaliar chama pontuar para cada rede, em ordem, e guarda as notas.
// Maior é melhor. NaN conta como o pior possível.
func (p *Populacao) Avaliar(pontuar func(*Rede) float64) {
	for i, r := range p.Redes {
		p.Pontuacoes[i] = limpar(pontuar(r))
		r.Pontuacao = p.Pontuacoes[i]
	}
}

// Definir guarda notas já calculadas (uma por rede, na ordem de Redes). É a
// porta para quem avalia por fora, por exemplo em paralelo.
func (p *Populacao) Definir(notas []float64) error {
	if len(notas) != len(p.Redes) {
		return fmt.Errorf("%w: %d notas para %d redes", ErrParams, len(notas), len(p.Redes))
	}
	for i, n := range notas {
		p.Pontuacoes[i] = limpar(n)
		p.Redes[i].Pontuacao = p.Pontuacoes[i]
	}
	return nil
}

func limpar(x float64) float64 {
	if math.IsNaN(x) {
		return math.Inf(-1)
	}
	return x
}

// Ranking devolve os índices de Redes do melhor para o pior (empate: o de
// menor índice primeiro, para o resultado não depender do algoritmo de ordenação).
func (p *Populacao) Ranking() []int {
	idx := make([]int, len(p.Redes))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return p.Pontuacoes[idx[a]] > p.Pontuacoes[idx[b]] })
	return idx
}

// Melhor devolve a melhor rede da avaliação atual e sua nota.
func (p *Populacao) Melhor() (*Rede, float64) {
	i := p.Ranking()[0]
	return p.Redes[i], p.Pontuacoes[i]
}

// Proxima monta a geração seguinte a partir das notas atuais: os Elite
// melhores passam iguais (cópias) e o resto nasce de cruzamento uniforme entre
// pais escolhidos por torneio, seguido de mutação gaussiana. A elite chega com
// a nota de antes; as notas dos filhos ficam zeradas até a próxima avaliação.
func (p *Populacao) Proxima() error {
	if len(p.Redes) != p.Params.Tamanho || len(p.Pontuacoes) != p.Params.Tamanho {
		return fmt.Errorf("%w: população com %d redes, o esperado é %d", ErrParams, len(p.Redes), p.Params.Tamanho)
	}
	rank := p.Ranking()
	prox := make([]*Rede, 0, p.Params.Tamanho)
	notas := make([]float64, p.Params.Tamanho)
	for n, i := range rank[:p.Params.Elite] {
		prox = append(prox, p.Redes[i].Copia())
		notas[n] = p.Pontuacoes[i]
	}
	for len(prox) < p.Params.Tamanho {
		a := p.Redes[p.sortearTorneio()]
		b := p.Redes[p.sortearTorneio()]
		filho, err := Cruzar(a, b, p.rng)
		if err != nil {
			return err
		}
		Mutar(filho, p.rng, p.Params.TaxaMutacao, p.Params.Sigma)
		prox = append(prox, filho)
	}
	p.Geracao++
	for _, r := range prox {
		r.Geracao = p.Geracao
	}
	p.Redes, p.Pontuacoes = prox, notas
	return nil
}

// sortearTorneio sorteia Torneio candidatos (com reposição) e devolve o
// índice do de maior nota; empate fica com o primeiro sorteado.
func (p *Populacao) sortearTorneio() int {
	melhor := p.rng.IntN(len(p.Redes))
	for k := 1; k < p.Params.Torneio; k++ {
		c := p.rng.IntN(len(p.Redes))
		if p.Pontuacoes[c] > p.Pontuacoes[melhor] {
			melhor = c
		}
	}
	return melhor
}

// Cruzar faz o cruzamento uniforme: cada peso e cada viés do filho vem do pai
// a ou do b, com chance meio a meio. Erro se as formas forem diferentes. Os
// pais não são alterados; o filho nasce sem metadados.
func Cruzar(a, b *Rede, rng *rand.Rand) (*Rede, error) {
	if err := a.conferir(); err != nil {
		return nil, err
	}
	if err := b.conferir(); err != nil {
		return nil, err
	}
	if len(a.Camadas) != len(b.Camadas) {
		return nil, fmt.Errorf("%w: pais com formas diferentes %v e %v", ErrParams, a.Camadas, b.Camadas)
	}
	for k := range a.Camadas {
		if a.Camadas[k] != b.Camadas[k] {
			return nil, fmt.Errorf("%w: pais com formas diferentes %v e %v", ErrParams, a.Camadas, b.Camadas)
		}
	}
	f := a.Copia()
	f.Pista, f.Geracao, f.Pontuacao = "", 0, 0
	for k := range f.Pesos {
		for i := range f.Pesos[k].D {
			if rng.IntN(2) == 1 {
				f.Pesos[k].D[i] = b.Pesos[k].D[i]
			}
		}
		for i := range f.Vieses[k] {
			if rng.IntN(2) == 1 {
				f.Vieses[k][i] = b.Vieses[k][i]
			}
		}
	}
	return f, nil
}

// Mutar soma ruído gaussiano N(0, sigma) a cada peso e viés com chance taxa
// (em média taxa*100% dos parâmetros mudam). Altera r no lugar e devolve
// quantos parâmetros mutaram.
func Mutar(r *Rede, rng *rand.Rand, taxa, sigma float64) int {
	n := 0
	for k := range r.Pesos {
		for i := range r.Pesos[k].D {
			if rng.Float64() < taxa {
				r.Pesos[k].D[i] += rng.NormFloat64() * sigma
				n++
			}
		}
		for i := range r.Vieses[k] {
			if rng.Float64() < taxa {
				r.Vieses[k][i] += rng.NormFloat64() * sigma
				n++
			}
		}
	}
	return n
}
