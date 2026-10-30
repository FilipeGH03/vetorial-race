package sim

import (
	"math"
	"math/rand"
	"testing"
)

// dentroDoPoligono diz se q está dentro do polígono fechado w (regra par-ímpar).
func dentroDoPoligono(q Vec2, w []Vec2) bool {
	dentro := false
	n := len(w)
	for i := 0; i < n; i++ {
		a, b := w[i], w[(i+1)%n]
		if (a.Y > q.Y) != (b.Y > q.Y) {
			x := a.X + (q.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X)
			if q.X < x {
				dentro = !dentro
			}
		}
	}
	return dentro
}

// naPista diz se q está entre as duas paredes: dentro de uma e fora da outra.
func naPista(p *Pista, q Vec2) bool {
	return dentroDoPoligono(q, p.ParedeEsq) != dentroDoPoligono(q, p.ParedeDir)
}

// contraParede leva um carro do meio da reta de baixo do oval (400,300) para
// -y, contra a parede de fora (y = 255), até bater. Devolve o carro, a posição
// do começo do tick em que bateu e quantos ticks levou.
func contraParede(t *testing.T, cfg Config) (*Carro, Vec2, int) {
	t.Helper()
	p := carregarOval(t)
	c := carroEm(Vec2{400, 300}, 0)
	ctl := seg(EixoY, -1)
	for tick := 1; tick <= 120; tick++ {
		antes := c.Pos
		c.Avancar(ctl, p, cfg)
		if c.Batido {
			return c, antes, tick
		}
		if c.Pos.Y <= 255 {
			t.Fatalf("tick %d: carro em %v passou da parede (y = 255) sem bater", tick, c.Pos)
		}
	}
	t.Fatalf("carro não bateu em 120 ticks; está em %v", c.Pos)
	return nil, Vec2{}, 0
}

func TestBaterSemMorrer(t *testing.T) {
	cfg := ConfigPadrao() // BaterMata = false (corrida)
	c, antes, tick := contraParede(t, cfg)
	t.Logf("bateu no tick %d, voltou para %v", tick, c.Pos)
	if !c.Vivo {
		t.Error("com BaterMata = false o carro morreu ao bater")
	}
	if c.Vel != (Vec2{}) {
		t.Errorf("Vel depois de bater = %v, quer zero", c.Vel)
	}
	if c.Pos != antes {
		t.Errorf("Pos depois de bater = %v, quer a do começo do tick %v", c.Pos, antes)
	}
	if c.Pos.Y <= 255 || c.Pos.Y >= 300 {
		t.Errorf("Pos depois de bater = %v, quer entre a parede (255) e a largada (300)", c.Pos)
	}
	// Batido vale só para o tick em que bateu; sem ação, o carro fica parado.
	c.Avancar(Controle{}, carregarOval(t), cfg)
	if c.Batido || c.Pos != antes || !c.Vivo {
		t.Errorf("tick seguinte sem ação: Batido = %v, Pos = %v, Vivo = %v; quer false, %v, true", c.Batido, c.Pos, c.Vivo, antes)
	}
	// E consegue sair andando para longe da parede.
	p := carregarOval(t)
	for i := 0; i < 10; i++ {
		c.Avancar(seg(EixoY, +1), p, cfg)
	}
	if c.Batido || c.Pos.Y <= antes.Y {
		t.Errorf("depois de bater o carro não saiu da parede: Pos = %v, Batido = %v", c.Pos, c.Batido)
	}
}

func TestBaterMata(t *testing.T) {
	cfg := ConfigPadrao()
	cfg.BaterMata = true // treino
	c, antes, _ := contraParede(t, cfg)
	if c.Vivo {
		t.Error("com BaterMata = true o carro continuou vivo ao bater")
	}
	if c.Pos != antes || c.Vel != (Vec2{}) {
		t.Errorf("carro morto em %v com Vel %v; quer parado na última posição válida %v", c.Pos, c.Vel, antes)
	}
	// Morto não se mexe, e Batido não se repete.
	c.Avancar(seg(EixoY, +1), carregarOval(t), cfg)
	if c.Pos != antes || c.Vel != (Vec2{}) || c.Batido || c.Vivo {
		t.Errorf("carro morto mexeu: Pos = %v, Vel = %v, Batido = %v, Vivo = %v", c.Pos, c.Vel, c.Batido, c.Vivo)
	}
}

// TestNaoAtravessaParede usa um carro tão rápido que num tick andaria a pista
// inteira: a colisão olha o caminho do tick, não só o ponto final.
func TestNaoAtravessaParede(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	cfg.VMax = 1e6
	c := carroEm(Vec2{400, 300}, 0)
	c.Vel = Vec2{0, 6e4} // 1000 px por tick para +y: atravessaria o oval todo
	c.Avancar(Controle{}, p, cfg)
	if !c.Batido || c.Pos != (Vec2{400, 300}) {
		t.Errorf("carro rápido: Batido = %v, Pos = %v; quer bater e ficar em (400,300)", c.Batido, c.Pos)
	}
}

// TestCarrosNuncaSaemDaPista solta carros com controles aleatórios (seed fixa)
// em várias pistas: com BaterMata = false nenhum sai de entre as paredes.
func TestCarrosNuncaSaemDaPista(t *testing.T) {
	cfg := ConfigPadrao()
	rng := rand.New(rand.NewSource(42))
	batidas := 0
	for _, p := range pistasDeForma(t) {
		n := len(p.Centro)
		for k := 0; k < 6; k++ {
			i := rng.Intn(n)
			c := carroEm(p.Centro[i], p.Centro[(i+1)%n].Sub(p.Centro[i]).Angle())
			if !naPista(p, c.Pos) {
				t.Fatalf("%s: largada %v fora da pista", p.Nome, c.Pos)
			}
			ctl := Controle{}
			for tick := 0; tick < 600; tick++ {
				if tick%20 == 0 { // troca de ideia a cada 1/3 s
					ctl = Controle{}
					for _, e := range []Eixo{EixoX, EixoY} {
						if rng.Intn(3) > 0 {
							ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e, Sinal: int8(2*rng.Intn(2) - 1)})
						}
					}
				}
				c.Avancar(ctl, p, cfg)
				if c.Batido {
					batidas++
				}
				if !c.Vivo || !naPista(p, c.Pos) {
					t.Fatalf("%s, tick %d: carro em %v, Vivo = %v; saiu da pista", p.Nome, tick, c.Pos, c.Vivo)
				}
				if math.IsNaN(c.Pos.X) || math.IsNaN(c.Pos.Y) {
					t.Fatalf("%s, tick %d: posição NaN", p.Nome, tick)
				}
			}
		}
	}
	if batidas == 0 {
		t.Fatal("nenhum carro bateu; o teste não exercita a colisão")
	}
	t.Logf("%d batidas", batidas)
}

func TestAvancarSemPista(t *testing.T) {
	cfg := ConfigPadrao()
	c := carroEm(Vec2{0, 0}, 0)
	c.Avancar(seg(EixoX, +1), nil, cfg)
	if c.Batido || c.Pos.X <= 0 {
		t.Errorf("sem pista: Batido = %v, Pos = %v; quer andar sem bater", c.Batido, c.Pos)
	}
}

// TestCruzaParedeComPontoRepetido: o caminho que passa exatamente pelo ponto
// repetido de uma parede (trecho de comprimento zero) bate.
func TestCruzaParedeComPontoRepetido(t *testing.T) {
	achou := false
	for _, p := range pistasDeForma(t) {
		for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
			for i := range w {
				if w[i] != w[(i+1)%len(w)] {
					continue
				}
				achou = true
				de := p.Centro[i]
				ate := de.Add(w[i].Sub(de).Scale(1.5)) // passa do ponto
				if !cruzaParede(p, de, ate) {
					t.Errorf("%s, ponto %d: caminho %v→%v pelo ponto repetido %v não bateu", p.Nome, i, de, ate, w[i])
				}
				meio := de.Add(w[i].Sub(de).Scale(0.5)) // para antes
				if cruzaParede(p, de, meio) {
					t.Errorf("%s, ponto %d: caminho %v→%v antes da parede bateu", p.Nome, i, de, meio)
				}
			}
		}
	}
	if !achou {
		t.Fatal("nenhuma parede com ponto repetido nas pistas de teste")
	}
}
