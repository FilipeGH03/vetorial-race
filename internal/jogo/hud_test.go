package jogo

import (
	"math"
	"strings"
	"testing"

	"vetorial-race/internal/sim"
)

func perto(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSetasHUDInverteY(t *testing.T) {
	pos := sim.Vec2{X: 100, Y: 200}
	x, y, r := SetasHUD(pos, sim.Vec2{X: 40, Y: 80}, 0.5)
	// carro na tela: (100, 720-200)
	if x.X0 != 100 || x.Y0 != 520 || !perto(x.X1, 120) || !perto(x.Y1, 520) {
		t.Errorf("seta x = %+v", x)
	}
	// vy > 0 sobe na tela: Y1 < Y0
	if !perto(y.X1, 100) || !perto(y.Y1, 480) {
		t.Errorf("seta y = %+v", y)
	}
	if !perto(r.X1, 120) || !perto(r.Y1, 480) {
		t.Errorf("resultante = %+v", r)
	}
}

func TestSetasHUDSoDireitaAmarelaIgualVermelha(t *testing.T) {
	j := novoOval(t)
	for i := 0; i < 20; i++ {
		j.Passo(Teclas{Direita: true})
	}
	x, y, r := SetasHUD(j.Carro.Pos, j.Carro.Vel, EscalaSeta)
	if x != r {
		t.Errorf("amarela %+v != vermelha %+v", r, x)
	}
	if y.X0 != y.X1 || y.Y0 != y.Y1 {
		t.Errorf("seta y deveria ter comprimento zero: %+v", y)
	}
}

func TestSetasHUDParado(t *testing.T) {
	x, y, r := SetasHUD(sim.Vec2{X: 5, Y: 5}, sim.Vec2{}, EscalaSeta)
	for _, s := range []Seta{x, y, r} {
		if s.X0 != s.X1 || s.Y0 != s.Y1 {
			t.Errorf("seta com vel zero: %+v", s)
		}
	}
}

func TestTextoPainel(t *testing.T) {
	s := TextoPainel(sim.Vec2{X: 3, Y: 4})
	for _, esp := range []string{"3.0", "4.0", "5.0", "53.1"} {
		if !strings.Contains(s, esp) {
			t.Errorf("painel %q sem %q", s, esp)
		}
	}
	if s := TextoPainel(sim.Vec2{X: 0, Y: -10}); !strings.Contains(s, "270.0") {
		t.Errorf("-y deveria dar 270: %q", s)
	}
	if s := TextoPainel(sim.Vec2{}); !strings.Contains(s, "ang:    0.0") {
		t.Errorf("parado: %q", s)
	}
}

func TestPainelBateComVel(t *testing.T) {
	j := novoOval(t)
	for i := 0; i < 10; i++ {
		j.Passo(Teclas{Cima: true, Direita: true})
	}
	s := TextoPainel(j.Carro.Vel)
	if !strings.Contains(s, "45.0") {
		t.Errorf("W+D deveria dar 45 graus: %q", s)
	}
}
