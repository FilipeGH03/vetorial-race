package sim

import (
	"math"
	"testing"
)

func seg(eixo Eixo, sinal int8) Controle {
	return Controle{Acoes: []Acao{{Ativa: true, Eixo: eixo, Sinal: sinal}}}
}

func diagonal() Controle {
	return Controle{Acoes: []Acao{
		{Ativa: true, Eixo: EixoX, Sinal: +1},
		{Ativa: true, Eixo: EixoY, Sinal: +1},
	}}
}

// roda n ticks com o mesmo controle.
func roda(c *Carro, ctl Controle, cfg Config, n int) {
	for i := 0; i < n; i++ {
		c.Mover(ctl, cfg)
	}
}

func TestAceleracao(t *testing.T) {
	cfg := ConfigPadrao()
	casos := []struct {
		nome  string
		ctl   Controle
		ticks int
		want  Vec2
	}{
		{"+X por 1 s", seg(EixoX, +1), 60, Vec2{300, 0}},
		{"-X por 1 s", seg(EixoX, -1), 60, Vec2{-300, 0}},
		{"+Y por 1 s", seg(EixoY, +1), 60, Vec2{0, 300}},
		{"-Y por 1 s", seg(EixoY, -1), 60, Vec2{0, -300}},
		{"+X por 1 tick", seg(EixoX, +1), 1, Vec2{5, 0}},
		{"W e D juntas por 0,5 s", diagonal(), 30, Vec2{150, 150}},
	}
	for _, tc := range casos {
		c := NovoCarro(1, Vec2{})
		roda(&c, tc.ctl, cfg, tc.ticks)
		if math.Abs(c.Vel.X-tc.want.X) > 1e-6 || math.Abs(c.Vel.Y-tc.want.Y) > 1e-6 {
			t.Errorf("%s: Vel = %v, want %v", tc.nome, c.Vel, tc.want)
		}
	}
}

func TestNuncaPassaDeVMax(t *testing.T) {
	cfg := ConfigPadrao()
	casos := []struct {
		nome string
		ctl  Controle
	}{
		{"+X", seg(EixoX, +1)},
		{"-Y", seg(EixoY, -1)},
		{"+X e +Y", diagonal()},
	}
	for _, tc := range casos {
		c := NovoCarro(1, Vec2{})
		for i := 0; i < 600; i++ {
			c.Mover(tc.ctl, cfg)
			if c.Vel.Len() > cfg.VMax+1e-9 || math.Abs(c.Vel.X) > cfg.VMax+1e-9 || math.Abs(c.Vel.Y) > cfg.VMax+1e-9 {
				t.Fatalf("%s: tick %d: Vel = %v passou de VMax", tc.nome, i, c.Vel)
			}
		}
	}
	// Um eixo satura no componente; dois eixos saturam no módulo.
	c := NovoCarro(1, Vec2{})
	roda(&c, seg(EixoX, +1), cfg, 600)
	if !perto(c.Vel.X, cfg.VMax) {
		t.Errorf("+X por 10 s: Vel.X = %v, want %v", c.Vel.X, cfg.VMax)
	}
	d := NovoCarro(2, Vec2{})
	roda(&d, diagonal(), cfg, 600)
	if !perto(d.Vel.Len(), cfg.VMax) {
		t.Errorf("diagonal: |Vel| = %v, want %v", d.Vel.Len(), cfg.VMax)
	}
}

func TestSemAcaoDecai(t *testing.T) {
	cfg := ConfigPadrao()
	casos := []struct {
		nome string
		ctl  Controle
	}{
		{"controle vazio", Controle{}},
		{"acao inativa", Controle{Acoes: []Acao{{Ativa: false, Eixo: EixoX, Sinal: +1}}}},
	}
	for _, tc := range casos {
		c := NovoCarro(1, Vec2{})
		c.Vel = Vec2{200, -100}
		c.Mover(tc.ctl, cfg)
		f := 1 - cfg.Atrito*cfg.DT
		if !perto(c.Vel.X, 200*f) || !perto(c.Vel.Y, -100*f) {
			t.Errorf("%s: Vel = %v, want (%v, %v)", tc.nome, c.Vel, 200*f, -100*f)
		}
		roda(&c, tc.ctl, cfg, 600)
		if c.Vel.Len() >= 200*f {
			t.Errorf("%s: depois de 10 s Vel = %v, deveria ter decaído mais", tc.nome, c.Vel)
		}
	}
}

func TestDecaiSoNoEixoSemAcao(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{})
	c.Vel = Vec2{0, 200}
	c.Mover(seg(EixoX, +1), cfg)
	if !perto(c.Vel.X, 5) {
		t.Errorf("Vel.X = %v, want 5", c.Vel.X)
	}
	if want := 200 * (1 - cfg.Atrito*cfg.DT); !perto(c.Vel.Y, want) {
		t.Errorf("Vel.Y = %v, want %v (decai)", c.Vel.Y, want)
	}
}

func TestPosicao(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{10, 20})
	c.Vel = Vec2{60, -120}
	c.Mover(seg(EixoX, +1), cfg) // X: 60+5 = 65; Y decai
	vy := -120 * (1 - cfg.Atrito*cfg.DT)
	if !perto(c.Pos.X, 10+65.0/60) || !perto(c.Pos.Y, 20+vy/60) {
		t.Errorf("Pos = %v", c.Pos)
	}
}

func TestAngulo(t *testing.T) {
	cfg := ConfigPadrao()

	// Ângulo de (0,1) é π/2.
	c := NovoCarro(1, Vec2{})
	c.Vel = Vec2{0, 100}
	c.Mover(seg(EixoY, +1), cfg)
	if !perto(c.Angulo(), math.Pi/2) {
		t.Errorf("Angulo = %v, want π/2", c.Angulo())
	}

	// Parado mantém o ângulo anterior.
	p := NovoCarro(2, Vec2{})
	p.DefinirAngulo(1.234)
	p.Mover(Controle{}, cfg)
	if p.Angulo() != 1.234 {
		t.Errorf("parado: Angulo = %v, want 1.234", p.Angulo())
	}

	// Freia até |Vel| < 1: o ângulo fica no último valor (direção -x, ±π).
	q := NovoCarro(3, Vec2{})
	q.Vel = Vec2{-50, 0}
	q.Mover(Controle{}, cfg)
	roda(&q, Controle{}, cfg, 600)
	if q.Vel.Len() >= 1 {
		t.Fatalf("Vel = %v, deveria ter caído abaixo de 1", q.Vel)
	}
	ant := q.Angulo()
	if !perto(math.Abs(ant), math.Pi) {
		t.Fatalf("Angulo = %v, want ±π", ant)
	}
	roda(&q, Controle{}, cfg, 100)
	if q.Angulo() != ant {
		t.Errorf("Angulo mudou com o carro quase parado: %v -> %v", ant, q.Angulo())
	}
}

func TestCarroMortoNaoSeMexe(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{5, 5})
	c.Vivo = false
	c.Vel = Vec2{100, 0}
	c.Mover(seg(EixoX, +1), cfg)
	if c.Pos != (Vec2{5, 5}) || c.Vel != (Vec2{100, 0}) {
		t.Errorf("carro morto mexeu: Pos=%v Vel=%v", c.Pos, c.Vel)
	}
}
