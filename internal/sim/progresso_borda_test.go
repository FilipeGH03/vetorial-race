package sim

import (
	"math"
	"math/rand"
	"testing"
)

// Controle aleatório (com seed) dirigindo com Avancar e batendo nas paredes: o
// progresso nunca regride, Proximo fica no intervalo e Voltas é coerente com
// Distancia, em todas as formas de pista.
func TestProgressoInvariantesComColisao(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		n := len(p.Checkpoints)
		rng := rand.New(rand.NewSource(7))
		for k := 0; k < 4; k++ {
			c := carroEm(p.Centro[n-3], 0)
			c.DefinirAngulo(p.Centro[n-2].Sub(p.Centro[n-3]).Angle())
			ultD, ultV := 0.0, 0
			for tick := 0; tick < 3000; tick++ {
				var ctl Controle
				for _, e := range []Eixo{EixoX, EixoY} {
					if rng.Intn(3) > 0 {
						s := int8(1)
						if rng.Intn(2) == 0 {
							s = -1
						}
						ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e, Sinal: s})
					}
				}
				de := c.Pos
				c.Avancar(ctl, p, cfg)
				c.AtualizarProgresso(de, p)
				pr := c.Progresso
				if pr.Proximo < 0 || pr.Proximo >= n {
					t.Fatalf("%s: Proximo %d fora de [0,%d)", p.Nome, pr.Proximo, n)
				}
				if pr.Distancia < ultD || pr.Voltas < ultV {
					t.Fatalf("%s tick %d: progresso regrediu %+v (antes D=%.2f V=%d)", p.Nome, tick, pr, ultD, ultV)
				}
				if pr.Voltas > 0 && pr.Distancia < float64(pr.Voltas)*p.Comprimento-p.Comprimento/2 {
					t.Fatalf("%s: Voltas %d incoerente com Distancia %.1f", p.Nome, pr.Voltas, pr.Distancia)
				}
				if math.IsNaN(pr.Distancia) || math.IsInf(pr.Distancia, 0) {
					t.Fatalf("%s: Distancia %v", p.Nome, pr.Distancia)
				}
				ultD, ultV = pr.Distancia, pr.Voltas
			}
		}
	}
}

// Bater e voltar ao começo do tick (de == Pos) não muda o progresso.
func TestProgressoPararNaoMuda(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-3], 0)
	andar(c, p, voltaPor(p.Centro, n-3, 6, 1)...)
	antes := c.Progresso
	for i := 0; i < 5; i++ {
		c.AtualizarProgresso(c.Pos, p)
	}
	if c.Progresso != antes {
		t.Errorf("parado: %+v, quer %+v", c.Progresso, antes)
	}
}

// Sentado em cima da linha de largada e andando de ré e para frente várias vezes
// não gera volta.
func TestOscilarNaLargadaNaoDaVolta(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-1], 0)
	for i := 0; i < 20; i++ {
		andar(c, p, p.Centro[1], p.Centro[n-1])
	}
	if c.Progresso.Voltas != 0 {
		t.Errorf("oscilar na largada: Voltas = %d", c.Progresso.Voltas)
	}
	if c.Progresso.Distancia > 2*p.Passo {
		t.Errorf("Distancia = %.1f depois de oscilar", c.Progresso.Distancia)
	}
}

// Um caminho de um tick que cruza vários checkpoints em ordem conta todos; um
// que cruza o seguinte sem o atual (pulo) não conta.
func TestSaltoLongoEmOrdem(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-1], 0)
	de := c.Pos
	c.Pos = p.Centro[4].Add(p.Centro[5].Sub(p.Centro[4]).Scale(0.5))
	c.AtualizarProgresso(de, p)
	if c.Progresso.Proximo != 5 {
		t.Errorf("salto de n-1 a 4: Proximo = %d, quer 5", c.Progresso.Proximo)
	}
	// Pulo: Proximo = 0, mas o carro aparece lá na frente sem cruzar a largada.
	c2 := carroEm(p.Centro[2], 0)
	de = c2.Pos
	c2.Pos = p.Centro[4]
	c2.AtualizarProgresso(de, p)
	if c2.Progresso.Proximo != 0 || c2.Progresso.Distancia != 0 {
		t.Errorf("pulo: %+v, quer Proximo 0 e Distancia 0", c2.Progresso)
	}
}

func TestClassificarBorda(t *testing.T) {
	if r := Classificar([]Carro{}); len(r) != 0 {
		t.Errorf("vazio: %v", r)
	}
	um := []Carro{NovoCarro(0, Vec2{})}
	if r := Classificar(um); len(r) != 1 || r[0] != 0 {
		t.Errorf("um carro: %v", r)
	}
	// Todos empatados: ordem original.
	cs := make([]Carro, 6)
	for i := range cs {
		cs[i] = NovoCarro(i, Vec2{})
	}
	for i, v := range Classificar(cs) {
		if v != i {
			t.Fatalf("empate total desordenou: %v", Classificar(cs))
		}
	}
	// Voltas pesa mais que qualquer Distancia, inclusive enorme.
	cs[3].Progresso = Progresso{Voltas: 1}
	cs[5].Progresso = Progresso{Voltas: 0, Distancia: 1e9}
	if r := Classificar(cs); r[0] != 3 || r[1] != 5 {
		t.Errorf("Voltas deve vencer Distancia: %v", r)
	}
}
