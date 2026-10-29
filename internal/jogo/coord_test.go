package jogo

import "testing"

func TestMundoParaTela(t *testing.T) {
	x, y := MundoParaTela(100, 0)
	if x != 100 || y != AlturaTela {
		t.Fatalf("got %v,%v", x, y)
	}
	x, y = MundoParaTela(0, AlturaTela)
	if x != 0 || y != 0 {
		t.Fatalf("got %v,%v", x, y)
	}
	wx, wy := TelaParaMundo(MundoParaTela(37, 215))
	if wx != 37 || wy != 215 {
		t.Fatalf("ida e volta: %v,%v", wx, wy)
	}
}

func TestAnguloTela(t *testing.T) {
	if AnguloTela(0.5) != -0.5 {
		t.Fatal("deveria inverter o sinal")
	}
}
