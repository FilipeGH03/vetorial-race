package jogo

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"vetorial-race/internal/sim"
)

func teclas(ks ...ebiten.Key) Teclas {
	set := map[ebiten.Key]bool{}
	for _, k := range ks {
		set[k] = true
	}
	return TeclasDe(func(k ebiten.Key) bool { return set[k] })
}

func acaoEm(ctl sim.Controle, e sim.Eixo) (sim.Acao, int) {
	var a sim.Acao
	n := 0
	for _, x := range ctl.Acoes {
		if x.Ativa && x.Eixo == e {
			a = x
			n++
		}
	}
	return a, n
}

func TestTeclasUmaPorEixo(t *testing.T) {
	casos := []struct {
		nome   string
		teclas []ebiten.Key
		eixo   sim.Eixo
		sinal  int8
	}{
		{"W", []ebiten.Key{ebiten.KeyW}, sim.EixoY, +1},
		{"seta cima", []ebiten.Key{ebiten.KeyArrowUp}, sim.EixoY, +1},
		{"S", []ebiten.Key{ebiten.KeyS}, sim.EixoY, -1},
		{"seta baixo", []ebiten.Key{ebiten.KeyArrowDown}, sim.EixoY, -1},
		{"D", []ebiten.Key{ebiten.KeyD}, sim.EixoX, +1},
		{"seta direita", []ebiten.Key{ebiten.KeyArrowRight}, sim.EixoX, +1},
		{"A", []ebiten.Key{ebiten.KeyA}, sim.EixoX, -1},
		{"seta esquerda", []ebiten.Key{ebiten.KeyArrowLeft}, sim.EixoX, -1},
	}
	for _, c := range casos {
		ctl := ControleDeTeclas(teclas(c.teclas...))
		if len(ctl.Acoes) != 1 {
			t.Fatalf("%s: %d ações, esperada 1", c.nome, len(ctl.Acoes))
		}
		a := ctl.Acoes[0]
		if !a.Ativa || a.Eixo != c.eixo || a.Sinal != c.sinal {
			t.Errorf("%s: got %+v, esperado eixo %d sinal %d", c.nome, a, c.eixo, c.sinal)
		}
	}
}

func TestTeclasJuntas(t *testing.T) {
	ctl := ControleDeTeclas(teclas(ebiten.KeyW, ebiten.KeyD))
	if len(ctl.Acoes) != 2 {
		t.Fatalf("W+D deveria dar 2 ações, deu %d", len(ctl.Acoes))
	}
	if a, n := acaoEm(ctl, sim.EixoY); n != 1 || a.Sinal != 1 {
		t.Errorf("W+D: eixo Y %+v (n=%d)", a, n)
	}
	if a, n := acaoEm(ctl, sim.EixoX); n != 1 || a.Sinal != 1 {
		t.Errorf("W+D: eixo X %+v (n=%d)", a, n)
	}
	// Tecla de WASD e seta da mesma direção juntas: continua uma ação só.
	ctl = ControleDeTeclas(teclas(ebiten.KeyW, ebiten.KeyArrowUp))
	if len(ctl.Acoes) != 1 {
		t.Errorf("W+↑ deveria dar 1 ação, deu %d", len(ctl.Acoes))
	}
	// Quatro teclas: no máximo uma ação por eixo.
	ctl = ControleDeTeclas(teclas(ebiten.KeyW, ebiten.KeyS, ebiten.KeyA, ebiten.KeyD))
	if len(ctl.Acoes) != 0 {
		t.Errorf("opostos se anulam: got %+v", ctl.Acoes)
	}
	ctl = ControleDeTeclas(teclas(ebiten.KeyS, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyD, ebiten.KeyA))
	if _, n := acaoEm(ctl, sim.EixoX); n != 0 {
		t.Errorf("A+D deveria anular o eixo X")
	}
	if a, n := acaoEm(ctl, sim.EixoY); n != 1 || a.Sinal != -1 {
		t.Errorf("S+↓: %+v (n=%d)", a, n)
	}
}

func TestSemTeclasSemAcao(t *testing.T) {
	ctl := ControleDeTeclas(teclas())
	if len(ctl.Acoes) != 0 {
		t.Fatalf("sem teclas, got %+v", ctl.Acoes)
	}
	ctl = ControleDeTeclas(teclas(ebiten.KeySpace, ebiten.KeyQ))
	if len(ctl.Acoes) != 0 {
		t.Fatalf("teclas estranhas, got %+v", ctl.Acoes)
	}
}

func novoOval(t *testing.T) *Jogo {
	t.Helper()
	p, err := sim.CarregarPista("../../pistas/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	return NovoComPista(p)
}

func TestLargadaNaPistaApontadaParaFrente(t *testing.T) {
	j := novoOval(t)
	if j.Carro.Pos != j.Pista.Centro[0] || !j.Carro.Vivo {
		t.Fatalf("carro fora da largada: %+v", j.Carro)
	}
	// oval: Centro[0] -> Centro[1] aponta para +x.
	if a := j.Carro.Angulo(); math.Abs(a) > 1e-9 {
		t.Errorf("ângulo da largada = %v, esperado 0", a)
	}
	if j.Cfg.BaterMata {
		t.Error("na corrida BaterMata deve ser false (D2)")
	}
	// Parado e sem teclas, nada muda.
	for i := 0; i < 30; i++ {
		j.Passo(Teclas{})
	}
	if j.Carro.Pos != j.Pista.Centro[0] || j.Carro.Vel != (sim.Vec2{}) {
		t.Errorf("carro parado se mexeu: %+v", j.Carro)
	}
}

func TestDirigirNoOval(t *testing.T) {
	// D: vai à direita.
	j := novoOval(t)
	ini := j.Carro.Pos
	for i := 0; i < 30; i++ {
		j.Passo(Teclas{Direita: true})
	}
	if j.Carro.Pos.X <= ini.X+10 || math.Abs(j.Carro.Pos.Y-ini.Y) > 1e-9 {
		t.Errorf("D: pos %v -> %v", ini, j.Carro.Pos)
	}

	// W: sobe (y para cima).
	j = novoOval(t)
	for i := 0; i < 10; i++ {
		j.Passo(Teclas{Cima: true})
	}
	if j.Carro.Vel.Y <= 0 || j.Carro.Pos.Y <= j.Pista.Centro[0].Y {
		t.Errorf("W: vel %v pos %v", j.Carro.Vel, j.Carro.Pos)
	}
	if a := j.Carro.Angulo(); math.Abs(a-math.Pi/2) > 1e-9 {
		t.Errorf("W: ângulo %v, esperado 90°", a)
	}

	// W+D: diagonal a 45°.
	j = novoOval(t)
	for i := 0; i < 10; i++ {
		j.Passo(Teclas{Cima: true, Direita: true})
	}
	if a := j.Carro.Angulo(); math.Abs(a-math.Pi/4) > 1e-9 {
		t.Errorf("W+D: ângulo %v, esperado 45°", a)
	}
	if math.Abs(j.Carro.Vel.X-j.Carro.Vel.Y) > 1e-9 || j.Carro.Vel.X <= 0 {
		t.Errorf("W+D: vel %v", j.Carro.Vel)
	}
}

func TestBaterNaParedeParaOCarro(t *testing.T) {
	j := novoOval(t)
	// Reta de baixo do oval: a parede de cima está em y = 300 + 45.
	j.Carro = sim.NovoCarro(0, sim.Vec2{X: 400, Y: 300})
	bateu := false
	for i := 0; i < 600 && !bateu; i++ {
		j.Passo(Teclas{Cima: true})
		bateu = j.Carro.Batido
	}
	if !bateu {
		t.Fatal("subindo reto na reta de baixo, o carro deveria bater na parede")
	}
	if j.Carro.Vel != (sim.Vec2{}) || !j.Carro.Vivo {
		t.Errorf("ao bater: vel %v vivo %v (esperado parado e vivo)", j.Carro.Vel, j.Carro.Vivo)
	}
	// Continua segurando W: não atravessa nunca.
	for i := 0; i < 600; i++ {
		j.Passo(Teclas{Cima: true})
		pos := j.Carro.Pos
		if pos.Y > 300+j.Pista.Largura/2+1e-6 {
			t.Fatalf("atravessou a parede: %v", pos)
		}
	}
}

func TestComprimentoDoCarro(t *testing.T) {
	j := novoOval(t)
	if c := ComprimentoCarro(j.Pista); math.Abs(c-36) > 1e-9 {
		t.Errorf("comprimento no oval = %v, esperado 36 (40%% de 90)", c)
	}
}
