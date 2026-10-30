package sim

import (
	"math"
	"math/rand"
	"testing"
)

// pontoNaPista sorteia um ponto dentro da pista: perto da linha central, com
// deslocamento lateral de até a meia largura (inclusive muito perto da parede).
func pontoNaPista(rng *rand.Rand, p *Pista) Vec2 {
	n := len(p.Centro)
	i := rng.Intn(n)
	a, b := p.Centro[i], p.Centro[(i+1)%n]
	q := a.Add(b.Sub(a).Scale(rng.Float64()))
	tan := b.Sub(a).Norm()
	nor := Vec2{-tan.Y, tan.X}
	return q.Add(nor.Scale((rng.Float64()*2 - 1) * p.Largura / 2 * 0.999))
}

// Sensores com o carro fora do centro: o sinal dos ângulos importa (+90 é a
// esquerda de quem anda, y para cima) e o ângulo do carro gira os raios.
func TestSensoresLadoEsquerdoDireito(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	// Reta de baixo (y = 300, +x): parede esquerda em y = 345, direita em y = 255.
	c := carroEm(Vec2{400, 320}, 0)
	l := Ler(c, p, cfg)
	if got, want := l.Sensores[6]*300, 25.0; math.Abs(got-want) > 1e-6 {
		t.Errorf("+90 graus (esquerda) = %v px, quer %v", got, want)
	}
	if got, want := l.Sensores[0]*300, 65.0; math.Abs(got-want) > 1e-6 {
		t.Errorf("-90 graus (direita) = %v px, quer %v", got, want)
	}
	// Virado para -x (pi), a esquerda de quem anda passa a ser -y (parede em 255).
	c = carroEm(Vec2{400, 320}, math.Pi)
	l = Ler(c, p, cfg)
	if got, want := l.Sensores[6]*300, 65.0; math.Abs(got-want) > 1e-6 {
		t.Errorf("virado para -x, +90 graus = %v px, quer %v", got, want)
	}
	if got, want := l.Sensores[0]*300, 25.0; math.Abs(got-want) > 1e-6 {
		t.Errorf("virado para -x, -90 graus = %v px, quer %v", got, want)
	}
	// Ângulo fora de (-pi, pi] (ex.: 2pi + 0, -4pi) dá a mesma leitura.
	ref := Ler(carroEm(Vec2{400, 320}, 0), p, cfg)
	for _, a := range []float64{2 * math.Pi, -4 * math.Pi, 10 * math.Pi} {
		l = Ler(carroEm(Vec2{400, 320}, a), p, cfg)
		for i := range l.Sensores {
			if math.Abs(l.Sensores[i]-ref.Sensores[i]) > 1e-9 {
				t.Errorf("angulo %v: sensor %d = %v, quer %v", a, i, l.Sensores[i], ref.Sensores[i])
			}
		}
	}
}

// Oráculo independente dos sensores: anda ao longo do raio em passos curtos até
// sair de entre as paredes (dentro de uma e fora da outra). Todos os raios saem
// de pontos dentro da pista.
func TestSensoresContraMarchaDoRaio(t *testing.T) {
	cfg := ConfigPadrao()
	rng := rand.New(rand.NewSource(7))
	const passo = 1.0 // passo grosso; a saída é refinada por bissecção
	for _, p := range pistasDeForma(t) {
		for k := 0; k < 60; k++ {
			c := carroEm(pontoNaPista(rng, p), (rng.Float64()*2-1)*math.Pi)
			l := Ler(c, p, cfg)
			for i := range l.Sensores {
				d := direcaoSensor(c, i)
				esperado := cfg.AlcanceSensor
				for s := 0.0; s <= cfg.AlcanceSensor; s += passo {
					if !naPista(p, c.Pos.Add(d.Scale(s))) {
						lo, hi := math.Max(s-passo, 0), s
						for it := 0; it < 40; it++ {
							m := (lo + hi) / 2
							if naPista(p, c.Pos.Add(d.Scale(m))) {
								lo = m
							} else {
								hi = m
							}
						}
						esperado = hi
						break
					}
				}
				got := l.Sensores[i] * cfg.AlcanceSensor
				if got < esperado-1e-3 && foraPertoDe(p, c.Pos, d, got) {
					continue // o raio roça um recorte fino que o passo grosso da marcha não viu
				}
				if math.Abs(got-esperado) > 1e-3 {
					t.Fatalf("%s, carro %v ang %v, sensor %d: Ler = %v px, marcha do raio = %v px", p.Nome, c.Pos, c.Angulo(), i, got, esperado)
				}
			}
		}
	}
}

// Leitura em qualquer lugar (inclusive fora da pista, sobre a parede e muito
// longe) devolve valores em [0,1], sem NaN.
func TestLerSempreEmZeroUm(t *testing.T) {
	cfg := ConfigPadrao()
	rng := rand.New(rand.NewSource(11))
	for _, p := range pistasDeForma(t) {
		pos := []Vec2{{0, 0}, {-1e5, 1e5}, p.ParedeEsq[0], p.ParedeDir[3], p.Centro[0]}
		for k := 0; k < 100; k++ {
			pos = append(pos, Vec2{(rng.Float64()*2 - 1) * 2000, (rng.Float64()*2 - 1) * 2000})
		}
		for _, q := range pos {
			c := carroEm(q, rng.Float64()*20-10)
			for i, s := range Ler(c, p, cfg).Sensores {
				if !(s >= 0 && s <= 1) {
					t.Fatalf("%s, carro em %v: sensor %d = %v fora de [0,1]", p.Nome, q, i, s)
				}
			}
			for i, r := range RaiosSensores(c, p, cfg) {
				if math.IsNaN(r.B.X) || math.IsNaN(r.B.Y) {
					t.Fatalf("%s, carro em %v: raio %d com NaN", p.Nome, q, i)
				}
			}
		}
	}
}

// Sem encostar em parede, Avancar faz exatamente o que Mover faz.
func TestAvancarIgualMoverLongeDasParedes(t *testing.T) {
	p := pistaRetangulo(t)
	cfg := ConfigPadrao()
	a := carroEm(Vec2{1500, 500}, 0)
	b := carroEm(Vec2{1500, 500}, 0)
	rng := rand.New(rand.NewSource(3))
	for tick := 0; tick < 300; tick++ {
		ctl := Controle{}
		for _, e := range []Eixo{EixoX, EixoY} {
			if rng.Intn(2) == 0 {
				ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e, Sinal: int8(2*rng.Intn(2) - 1)})
			}
		}
		a.Avancar(ctl, p, cfg)
		b.Mover(ctl, cfg)
		if a.Batido {
			t.Fatalf("tick %d: bateu longe da parede em %v", tick, a.Pos)
		}
		if *a != *b {
			t.Fatalf("tick %d: Avancar = %+v, Mover = %+v", tick, *a, *b)
		}
	}
}

// Batido é zerado no começo de cada Avancar, até para carro morto.
func TestBatidoZeraNoProximoTick(t *testing.T) {
	cfg := ConfigPadrao()
	cfg.BaterMata = true
	c, _, _ := contraParede(t, cfg)
	if !c.Batido || c.Vivo {
		t.Fatalf("pré-condição: Batido = %v, Vivo = %v", c.Batido, c.Vivo)
	}
	c.Avancar(Controle{}, carregarOval(t), cfg)
	if c.Batido {
		t.Error("carro morto continuou com Batido = true no tick seguinte")
	}
}

// Um tick a partir de qualquer ponto da pista, em qualquer direção e velocidade
// (até 5x a VMax padrão, acima do que o contrato permite): o carro termina entre
// as paredes ou bate. Nunca fica fora da pista nem com NaN.
func TestUmTickNuncaSaiDaPista(t *testing.T) {
	cfg := ConfigPadrao()
	cfg.VMax = 2000
	for _, seed := range []int64{1, 2, 3} {
		rng := rand.New(rand.NewSource(seed))
		for _, p := range pistasDeForma(t) {
			for k := 0; k < 3000; k++ {
				q := pontoNaPista(rng, p)
				c := carroEm(q, 0)
				ang := rng.Float64() * 2 * math.Pi
				c.Vel = Vec2{math.Cos(ang), math.Sin(ang)}.Scale(rng.Float64() * cfg.VMax)
				ini := c.Vel
				c.Avancar(Controle{}, p, cfg)
				if math.IsNaN(c.Pos.X) || math.IsNaN(c.Pos.Y) {
					t.Fatalf("%s: NaN saindo de %v com Vel %v", p.Nome, q, ini)
				}
				if !naPista(p, c.Pos) {
					t.Fatalf("%s: carro saiu de %v com Vel %v e parou em %v, fora da pista (Batido = %v)", p.Nome, q, ini, c.Pos, c.Batido)
				}
				if c.Batido && (c.Pos != q || c.Vel != (Vec2{})) {
					t.Fatalf("%s: bateu mas Pos = %v (quer %v) e Vel = %v", p.Nome, c.Pos, q, c.Vel)
				}
			}
		}
	}
}

// O caminho que termina exatamente no vértice da parede (e o que passa por ele
// vindo de dentro da pista) bate, e o que para antes não bate.
func TestCaminhoNoVerticeDaParede(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
			for i, v := range w {
				de := p.Centro[i]
				if !cruzaParede(p, de, v) {
					t.Errorf("%s, vértice %d: caminho da linha central até %v não bateu", p.Nome, i, v)
				}
			}
		}
	}
}

// foraPertoDe diz se, perto de s ao longo do raio de o na direção d, algum
// ponto está fora da pista (varredura fina, 0,005 px, em s ± 0,5).
func foraPertoDe(p *Pista, o, d Vec2, s float64) bool {
	for x := math.Max(s-0.5, 0); x <= s+0.5; x += 0.005 {
		if !naPista(p, o.Add(d.Scale(x))) {
			return true
		}
	}
	return false
}
