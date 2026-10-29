package sim

import "testing"

func TestBordaAcoesOpostasNoMesmoEixo(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{})
	c.Vel = Vec2{100, 0}
	ctl := Controle{Acoes: []Acao{{true, EixoX, +1}, {true, EixoX, -1}}}
	c.Mover(ctl, cfg)
	if !perto(c.Vel.X, 100) {
		t.Errorf("acoes opostas: Vel.X = %v, want 100 (sem decair)", c.Vel.X)
	}
}

func TestBordaModuloPreservaDirecao(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{})
	c.Vel = Vec2{400, 400}
	c.Mover(Controle{Acoes: []Acao{{true, EixoX, +1}, {true, EixoY, +1}}}, cfg)
	if !perto(c.Vel.Len(), cfg.VMax) || !perto(c.Vel.X, c.Vel.Y) {
		t.Errorf("Vel = %v: modulo deve ser VMax e direcao 45 graus", c.Vel)
	}
}

func TestBordaDeterministico(t *testing.T) {
	cfg := ConfigPadrao()
	roda := func() Carro {
		c := NovoCarro(1, Vec2{10, 20})
		for i := 0; i < 500; i++ {
			c.Mover(Controle{Acoes: []Acao{{true, Eixo(i % 2), int8(1 - 2*((i/50)%2))}}}, cfg)
		}
		return c
	}
	if a, b := roda(), roda(); a != b {
		t.Errorf("simulacao nao deterministica: %v vs %v", a, b)
	}
}

func TestBordaFreiaAteInverter(t *testing.T) {
	cfg := ConfigPadrao()
	c := NovoCarro(1, Vec2{})
	for i := 0; i < 120; i++ {
		c.Mover(Controle{Acoes: []Acao{{true, EixoX, -1}}}, cfg)
	}
	if !perto(c.Vel.X, -cfg.VMax) {
		t.Errorf("Vel.X = %v, want -VMax", c.Vel.X)
	}
	if !perto(c.Angulo(), 3.141592653589793) {
		t.Errorf("Angulo = %v, want π", c.Angulo())
	}
}

func TestBordaVec2Zero(t *testing.T) {
	if n := (Vec2{}).Norm(); n != (Vec2{}) {
		t.Errorf("Norm de zero = %v", n)
	}
	if (Vec2{}).Len() != 0 {
		t.Error("Len de zero")
	}
}
