package jogo

import (
	"math"
	"path/filepath"
	"testing"

	"vetorial-race/internal/sim"
)

// Toda pista real do repositorio: o carro nasce em sim.Largada(p, 1)[0], vivo,
// com o angulo dado pela largada.
func TestLargadaEmTodasAsPistas(t *testing.T) {
	arqs, _ := filepath.Glob("../../pistas/*.json")
	if len(arqs) == 0 {
		t.Skip("sem pistas")
	}
	for _, a := range arqs {
		p, err := sim.CarregarPista(a)
		if err != nil {
			t.Errorf("%s: %v", a, err)
			continue
		}
		j := NovoComPista(p)
		ref := sim.Largada(p, 1)[0]
		if !j.Carro.Vivo || j.Carro.Pos != ref.Pos {
			t.Errorf("%s: carro fora da largada %+v", a, j.Carro)
		}
		esp := ref.Angulo()
		if d := math.Abs(j.Carro.Angulo() - esp); d > 1e-9 {
			t.Errorf("%s: angulo %v esperado %v", a, j.Carro.Angulo(), esp)
		}
	}
}

// Segurar W+D (diagonal) contra a parede: nunca sai do asfalto e nunca morre.
func TestDiagonalContraParedeNaoAtravessa(t *testing.T) {
	j := novoOval(t)
	j.Carro = sim.NovoCarro(0, sim.Vec2{X: 400, Y: 400})
	for i := 0; i < 3000; i++ {
		j.Passo(Teclas{Cima: true, Direita: true})
		if !j.Carro.Vivo {
			t.Fatal("carro morreu na corrida (BaterMata deve ser false)")
		}
		if j.Carro.Pos.Y > 500+j.Pista.Largura/2+1e-6 || j.Carro.Pos.X > 800 {
			t.Fatalf("tick %d: saiu da pista %v", i, j.Carro.Pos)
		}
	}
}

// Depois de bater e soltar as teclas, o carro fica parado e o angulo e mantido.
func TestAnguloMantidoAposBater(t *testing.T) {
	j := novoOval(t)
	j.Carro = sim.NovoCarro(0, sim.Vec2{X: 400, Y: 300})
	for i := 0; i < 600; i++ {
		j.Passo(Teclas{Cima: true})
	}
	a := j.Carro.Angulo()
	for i := 0; i < 60; i++ {
		j.Passo(Teclas{})
	}
	if j.Carro.Angulo() != a {
		t.Errorf("angulo mudou parado: %v -> %v", a, j.Carro.Angulo())
	}
	if math.IsNaN(j.Carro.Pos.X) || math.IsNaN(j.Carro.Pos.Y) {
		t.Errorf("pos NaN: %v", j.Carro.Pos)
	}
}

// Soltar as teclas faz o atrito reduzir a velocidade.
func TestAtritoAoSoltar(t *testing.T) {
	j := novoOval(t)
	for i := 0; i < 20; i++ {
		j.Passo(Teclas{Direita: true})
	}
	v := j.Carro.Vel.X
	for i := 0; i < 30; i++ {
		j.Passo(Teclas{})
	}
	if !(j.Carro.Vel.X > 0 && j.Carro.Vel.X < v) {
		t.Errorf("atrito: %v -> %v", v, j.Carro.Vel.X)
	}
}
