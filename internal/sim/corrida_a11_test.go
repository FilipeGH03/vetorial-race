package sim

import (
	"errors"
	"testing"
)

func ctlsN(n int) []Controlador {
	c := make([]Controlador, n)
	for i := range c {
		c[i] = parado{}
	}
	return c
}

// D19: 2 a 8 carros e humano em [-1, n).
func TestNovaCorridaA11Validacao(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	casos := []struct {
		nome   string
		n      int
		humano int
		ok     bool
	}{
		{"1 controlador", 1, -1, false},
		{"9 controladores", 9, -1, false},
		{"humano -2", 3, -2, false},
		{"humano = n", 3, 3, false},
		{"2 sem humano", 2, -1, true},
		{"8 com humano", 8, 7, true},
		{"humano 0", 4, 0, true},
	}
	for _, c := range casos {
		r, err := NovaCorrida(p, cfg, ModoVoltas, 1, ctlsN(c.n), c.humano)
		if c.ok {
			if err != nil || r == nil || r.Humano != c.humano {
				t.Errorf("%s: err = %v", c.nome, err)
			}
		} else if !errors.Is(err, ErrCorridaInvalida) || r != nil {
			t.Errorf("%s: err = %v, quer ErrCorridaInvalida", c.nome, err)
		}
	}
}

// Humano parado na Eliminação morre no tick 180 (laser sai com Distancia 0) e a
// partida termina com as 2 IAs vivas.
func TestCorridaA11HumanoMorreTermina(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, &seguidor{p: p, vAlvo: 150}, &seguidor{p: p, vAlvo: 150})
	r.Humano = 0
	for i := 0; i < 179; i++ {
		r.Step()
	}
	if r.Terminada {
		t.Fatalf("terminou antes do tick 180 (tick %d)", r.Tick)
	}
	r.Step()
	if !r.Terminada || r.Tick != 180 || r.Carros[0].Vivo {
		t.Fatalf("terminada=%v tick=%d humano vivo=%v", r.Terminada, r.Tick, r.Carros[0].Vivo)
	}
	if !r.Carros[1].Vivo || !r.Carros[2].Vivo {
		t.Fatal("as IAs deviam estar vivas")
	}
	if r.Vencedor != 1 && r.Vencedor != 2 {
		t.Errorf("Vencedor = %d, quer uma IA viva", r.Vencedor)
	}
	if cl := r.Classificacao(); cl[0] != r.Vencedor || cl[2] != 0 {
		t.Errorf("classificação %v", cl)
	}
}

// Parados morrendo juntos com a mesma Distancia: sem vencedor.
func TestCorridaA11EmpateSemVencedor(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{}, parado{})
	for i := 0; i < 180; i++ {
		r.Step()
	}
	if !r.Terminada || r.Vencedor != -1 {
		t.Errorf("terminada=%v vencedor=%d, quer fim com -1", r.Terminada, r.Vencedor)
	}
}

// Últimos morrendo juntos: o mais adiantado vence.
func TestCorridaA11MaisAdiantadoVence(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{}, parado{})
	r.Tick = 599 // laser em ~1036 px no próximo tick
	r.Carros[0].Progresso.Distancia = 100
	r.Carros[1].Progresso.Distancia = 300
	r.Carros[2].Progresso.Distancia = 200
	r.Step()
	if !r.Terminada || r.Vencedor != 1 {
		t.Errorf("terminada=%v vencedor=%d, quer 1", r.Terminada, r.Vencedor)
	}
}
