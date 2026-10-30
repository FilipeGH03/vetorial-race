package sim

import (
	"math"
	"testing"
)

// carroEm cria um carro vivo em pos, virado para o ângulo a (rad).
func carroEm(pos Vec2, a float64) *Carro {
	c := NovoCarro(1, pos)
	c.DefinirAngulo(a)
	return &c
}

// pistaRetangulo é um retângulo de 3000 x 1000 com retas longas, onde o raio da
// frente, no meio da reta de baixo, não alcança parede nenhuma.
func pistaRetangulo(t *testing.T) *Pista {
	t.Helper()
	p, err := NovaPista("retangulo", 90, []Vec2{{0, 0}, {3000, 0}, {3000, 1000}, {0, 1000}})
	if err != nil {
		t.Fatalf("NovaPista: %v", err)
	}
	return p
}

// distParede é a menor distância do ponto q às duas paredes da pista.
func distParede(p *Pista, q Vec2) float64 {
	d := math.Inf(1)
	for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for i := range w {
			d = math.Min(d, distPontoSeg(q, w[i], w[(i+1)%len(w)]))
		}
	}
	return d
}

func TestSensoresLateraisNaRetaDoOval(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	// Reta de baixo do oval: de (200,300) a (600,300), no sentido +x.
	c := carroEm(Vec2{400, 300}, 0)
	l := Ler(c, p, cfg)
	want := (p.Largura / 2) / cfg.AlcanceSensor // 45/300 = 0,15
	for _, i := range []int{0, 6} {             // -90 e +90 graus
		if math.Abs(l.Sensores[i]-want) > 1e-9 {
			t.Errorf("sensor %d (%v°) = %v, quer %v", i, angulosSensores[i], l.Sensores[i], want)
		}
	}
	// ±45° batem a meia/sen(45°) do centro.
	want45 := (p.Largura / 2) / math.Sin(math.Pi/4) / cfg.AlcanceSensor
	for _, i := range []int{1, 5} {
		if math.Abs(l.Sensores[i]-want45) > 1e-9 {
			t.Errorf("sensor %d (%v°) = %v, quer %v", i, angulosSensores[i], l.Sensores[i], want45)
		}
	}
	for i, s := range l.Sensores {
		if !(s >= 0 && s <= 1) {
			t.Errorf("sensor %d = %v fora de [0,1]", i, s)
		}
	}
	// A frente (0°) bate na parede de fora da curva, antes do alcance.
	if !(l.Sensores[3] > want && l.Sensores[3] < 1) {
		t.Errorf("sensor da frente = %v, quer entre %v e 1", l.Sensores[3], want)
	}
}

func TestSensorSemParedeDaUm(t *testing.T) {
	p := pistaRetangulo(t)
	cfg := ConfigPadrao()
	c := carroEm(Vec2{1000, 0}, 0) // meio da reta de baixo, para +x
	l := Ler(c, p, cfg)
	if l.Sensores[3] != 1 {
		t.Errorf("sensor da frente sem parede = %v, quer 1", l.Sensores[3])
	}
	// Os outros batem: o mais aberto que não é lateral (±20°) alcança 45/sen(20°) ≈ 131 px.
	for _, i := range []int{0, 1, 2, 4, 5, 6} {
		if l.Sensores[i] >= 1 {
			t.Errorf("sensor %d (%v°) = %v, quer < 1", i, angulosSensores[i], l.Sensores[i])
		}
	}

	// Longe de tudo, todos os raios dão 1.
	longe := carroEm(Vec2{1500, 500}, 0.3)
	for i, s := range Ler(longe, p, cfg).Sensores {
		if s != 1 {
			t.Errorf("carro longe das paredes: sensor %d = %v, quer 1", i, s)
		}
	}
	// Sem pista também.
	for i, s := range Ler(c, nil, cfg).Sensores {
		if s != 1 {
			t.Errorf("pista nil: sensor %d = %v, quer 1", i, s)
		}
	}
}

func TestLeituraUsaAnguloDoCarro(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	// Andando para +x, mas com o ângulo guardado para +y: a leitura segue c.Angulo().
	c := carroEm(Vec2{400, 300}, math.Pi/2)
	c.Vel = Vec2{30, -40}
	l := Ler(c, p, cfg)
	if l.Angulo != c.Angulo() || l.Angulo != math.Pi/2 {
		t.Errorf("Angulo = %v, quer c.Angulo() = %v", l.Angulo, c.Angulo())
	}
	if l.Vel != c.Vel || l.Modulo != 50 {
		t.Errorf("Vel = %v, Modulo = %v; quer %v e 50", l.Vel, l.Modulo, c.Vel)
	}
	// Virado para +y, o raio da frente bate na parede de dentro (y = 345) e os
	// laterais (±90°) apontam para ±x, ao longo da reta.
	want := (p.Largura / 2) / cfg.AlcanceSensor
	if math.Abs(l.Sensores[3]-want) > 1e-9 {
		t.Errorf("sensor da frente virado para +y = %v, quer %v", l.Sensores[3], want)
	}
	if l.Sensores[0] <= want || l.Sensores[6] <= want {
		t.Errorf("laterais ao longo da reta = %v e %v, quer > %v", l.Sensores[0], l.Sensores[6], want)
	}
}

func TestRaiosSensores(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		for i := 0; i < len(p.Centro); i += 3 {
			n := len(p.Centro)
			dir := p.Centro[(i+1)%n].Sub(p.Centro[i])
			c := carroEm(p.Centro[i], dir.Angle())
			l := Ler(c, p, cfg)
			raios := RaiosSensores(c, p, cfg)
			for k, r := range raios {
				s := l.Sensores[k]
				if !(s >= 0 && s <= 1) {
					t.Fatalf("%s, ponto %d: sensor %d = %v fora de [0,1]", p.Nome, i, k, s)
				}
				if r.A != c.Pos {
					t.Fatalf("%s, ponto %d: raio %d sai de %v, quer %v", p.Nome, i, k, r.A, c.Pos)
				}
				if math.Abs(r.B.Sub(r.A).Len()-s*cfg.AlcanceSensor) > 1e-6 {
					t.Errorf("%s, ponto %d: raio %d mede %v, sensor diz %v", p.Nome, i, k, r.B.Sub(r.A).Len(), s*cfg.AlcanceSensor)
				}
				wantAng := c.Angulo() + angulosSensores[k]*math.Pi/180
				if d := r.B.Sub(r.A); math.Abs(math.Remainder(d.Angle()-wantAng, 2*math.Pi)) > 1e-9 {
					t.Errorf("%s, ponto %d: raio %d no ângulo %v, quer %v", p.Nome, i, k, d.Angle(), wantAng)
				}
				if s < 1 && distParede(p, r.B) > 1e-6 {
					t.Errorf("%s, ponto %d: raio %d termina a %v px da parede", p.Nome, i, k, distParede(p, r.B))
				}
				// Da linha central, a parede está a pelo menos meia largura.
				if s*cfg.AlcanceSensor < p.Largura/2-1e-6 {
					t.Errorf("%s, ponto %d: sensor %d vê parede a %v px, meia largura %v", p.Nome, i, k, s*cfg.AlcanceSensor, p.Largura/2)
				}
			}
		}
	}
}

// TestSensorEmPontoRepetido mira um raio exatamente num ponto que a parede
// repete (trecho de comprimento zero, onde o laço da quina foi cortado).
func TestSensorEmPontoRepetido(t *testing.T) {
	cfg := ConfigPadrao()
	achou := 0
	for _, p := range pistasDeForma(t) {
		for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
			n := len(w)
			for i := range w {
				if w[i] != w[(i+1)%n] {
					continue
				}
				achou++
				centro := p.Centro[i]
				alvo := w[i].Sub(centro)
				if alvo.Len() >= cfg.AlcanceSensor {
					continue
				}
				c := carroEm(centro, alvo.Angle()) // sensor 3 (0°) aponta para o ponto
				l := Ler(c, p, cfg)
				got := l.Sensores[3] * cfg.AlcanceSensor
				if math.IsNaN(got) || math.Abs(got-alvo.Len()) > 1e-6 {
					t.Errorf("%s, ponto %d: sensor no ponto repetido = %v px, quer %v", p.Nome, i, got, alvo.Len())
				}
			}
		}
	}
	if achou == 0 {
		t.Fatal("nenhuma pista de teste tem parede com ponto repetido; o teste não exercita o caso")
	}
}

func TestSensoresComAlcanceZero(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	cfg.AlcanceSensor = 0
	c := carroEm(Vec2{400, 300}, 0)
	for i, s := range Ler(c, p, cfg).Sensores {
		if s != 1 {
			t.Errorf("alcance 0: sensor %d = %v, quer 1", i, s)
		}
	}
	for i, r := range RaiosSensores(c, p, cfg) {
		if r.A != r.B {
			t.Errorf("alcance 0: raio %d = %v, quer de comprimento zero", i, r)
		}
	}
}
