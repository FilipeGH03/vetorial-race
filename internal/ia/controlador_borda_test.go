package ia

import (
	"math"
	"testing"

	"vetorial-race/internal/sim"
)

// Entrada de uma Leitura real do oval (a que o treino e o jogo produzem):
// todos os 10 valores ficam em [-1, 1] e a ordem bate com o contrato 1.3.
func TestEntradaDeLeituraRealFicaEmMenos1a1(t *testing.T) {
	tr := treinoOval(t)
	c := tr.largada
	ctl := NovoControlador(redeFixa(2, 2, 0))
	for i := 0; i < 300 && c.Vivo; i++ {
		l := sim.Ler(&c, tr.Pista, tr.Cfg)
		x := Entrada(l)
		if len(x) != NEntrada {
			t.Fatalf("len = %d", len(x))
		}
		for k, v := range x {
			if math.IsNaN(v) || v < -1-1e-9 || v > 1+1e-9 {
				t.Fatalf("tick %d: Entrada[%d] = %v fora de [-1,1]", i, k, v)
			}
		}
		if x[7] != l.Vel.X/tr.Cfg.VMax || x[8] != l.Vel.Y/tr.Cfg.VMax || x[9] != l.Modulo/tr.Cfg.VMax {
			t.Fatalf("tick %d: velocidades fora de ordem: %v", i, x[7:])
		}
		c.Avancar(ctl.Decidir(l), tr.Pista, tr.Cfg)
	}
}

// Velocidade máxima e vetor zero nos extremos.
func TestEntradaExtremos(t *testing.T) {
	vmax := sim.ConfigPadrao().VMax
	x := Entrada(sim.Leitura{Vel: sim.Vec2{X: -vmax, Y: 0}, Modulo: vmax})
	if x[7] != -1 || x[8] != 0 || x[9] != 1 {
		t.Errorf("Vel = (-VMax, 0): %v", x[7:])
	}
	for _, v := range Entrada(sim.Leitura{}) {
		if v != 0 {
			t.Errorf("Leitura zero deu entrada %v", v)
		}
	}
}

// Cada Decidir devolve exatamente uma Acao com Sinal +1/-1 (contrato 1.1: no
// máximo uma por eixo) em uma corrida de verdade, sem BaterMata (jogo livre).
func TestControladorNaCorridaDoJogo(t *testing.T) {
	p := pistaOval(t)
	r, err := NovaRede(CamadasPadrao, nil)
	if err != nil {
		t.Fatal(err)
	}
	espia := &espiao{Controlador: NovoControlador(r)}
	cor, err := sim.NovaCorrida(p, sim.ConfigPadrao(), sim.ModoVoltas, 1, []sim.Controlador{espia})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1200 && !cor.Terminada; i++ {
		cor.Step()
	}
	if espia.chamadas == 0 {
		t.Fatal("Decidir nunca foi chamado")
	}
	if espia.ruim != "" {
		t.Fatal(espia.ruim)
	}
	if !cor.Carros[0].Vivo {
		t.Error("sem BaterMata o carro nunca morre")
	}
}

type espiao struct {
	*Controlador
	chamadas int
	ruim     string
}

func (e *espiao) Decidir(l sim.Leitura) sim.Controle {
	e.chamadas++
	c := e.Controlador.Decidir(l)
	if len(c.Acoes) != 1 {
		e.ruim = "Controle sem exatamente uma Acao"
	}
	for _, a := range c.Acoes {
		if a.Ativa && a.Sinal != 1 && a.Sinal != -1 {
			e.ruim = "Sinal fora de +1/-1"
		}
		if !a.Ativa && (a.Sinal != 0 || a.Eixo != 0) {
			e.ruim = "Acao inativa com lixo"
		}
	}
	return c
}
