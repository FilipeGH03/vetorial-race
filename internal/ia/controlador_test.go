package ia

import (
	"math"
	"testing"

	"vetorial-race/internal/sim"
)

var _ sim.Controlador = (*Controlador)(nil)

func TestAcaoDaSaidaQuadrantes(t *testing.T) {
	casos := []struct {
		s0, s1 float64
		eixo   sim.Eixo
		sinal  int8
	}{
		{0.5, 0.5, sim.EixoX, +1},   // s0 > 0, s1 > 0: aumenta x
		{-0.5, 0.5, sim.EixoX, -1},  // s0 < 0, s1 > 0: diminui x
		{-0.5, -0.5, sim.EixoY, -1}, // s0 < 0, s1 < 0: diminui y
		{0.5, -0.5, sim.EixoY, +1},  // s0 > 0, s1 < 0: aumenta y
		{1, 1, sim.EixoX, +1},
		{-1, -1, sim.EixoY, -1},
		{0.9, 0, sim.EixoY, +1},     // s1 = 0 não é > 0: EixoY
		{0.2, 0.3, sim.EixoX, +1},   // na borda da zona morta já age
		{-0.2, -0.3, sim.EixoY, -1}, // idem
	}
	for _, c := range casos {
		a := AcaoDaSaida([]float64{c.s0, c.s1})
		want := sim.Acao{Ativa: true, Eixo: c.eixo, Sinal: c.sinal}
		if a != want {
			t.Errorf("s = (%v, %v): %+v, esperava %+v", c.s0, c.s1, a, want)
		}
	}
}

func TestAcaoDaSaidaZonaMorta(t *testing.T) {
	for _, s0 := range []float64{0, 0.1, -0.1, 0.199, -0.199, math.Nextafter(0.2, 0), math.Nextafter(-0.2, 0)} {
		for _, s1 := range []float64{-1, -0.5, 0, 0.5, 1} {
			if a := AcaoDaSaida([]float64{s0, s1}); a.Ativa {
				t.Errorf("s = (%v, %v) está na zona morta e deu %+v", s0, s1, a)
			}
		}
	}
}

func TestAcaoDaSaidaInvalida(t *testing.T) {
	for _, s := range [][]float64{nil, {}, {0.9}, {math.NaN(), 0.5}} {
		if a := AcaoDaSaida(s); a.Ativa {
			t.Errorf("saída %v deu ação ativa %+v", s, a)
		}
	}
}

func TestEntradaOrdemENormalizacao(t *testing.T) {
	l := sim.Leitura{
		Sensores: [7]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
		Vel:      sim.Vec2{X: 200, Y: -100},
		Modulo:   math.Hypot(200, -100),
		Angulo:   1.23, // não entra na rede
	}
	x := Entrada(l)
	vmax := sim.ConfigPadrao().VMax
	want := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 200 / vmax, -100 / vmax, math.Hypot(200, -100) / vmax}
	if len(x) != NEntrada {
		t.Fatalf("len(Entrada) = %d, esperava %d", len(x), NEntrada)
	}
	for i := range want {
		if !perto(x[i], want[i]) {
			t.Errorf("Entrada[%d] = %v, esperava %v", i, x[i], want[i])
		}
	}
}

// redeFixa é uma 10-2 sem camada escondida cuja saída é (tanh(b0), tanh(b1)),
// mais s0 += tanh do sensor 0 com peso w0.
func redeFixa(b0, b1, w0 float64) *Rede {
	r := vazia([]int{NEntrada, NSaida})
	r.Vieses[0][0], r.Vieses[0][1] = b0, b1
	r.Pesos[0].Set(0, 0, w0)
	return r
}

func TestControladorDecide(t *testing.T) {
	casos := []struct {
		b0, b1 float64
		want   sim.Acao
	}{
		{2, 2, sim.Acao{Ativa: true, Eixo: sim.EixoX, Sinal: +1}},
		{-2, 2, sim.Acao{Ativa: true, Eixo: sim.EixoX, Sinal: -1}},
		{-2, -2, sim.Acao{Ativa: true, Eixo: sim.EixoY, Sinal: -1}},
		{2, -2, sim.Acao{Ativa: true, Eixo: sim.EixoY, Sinal: +1}},
		{0.1, 2, sim.Acao{}}, // tanh(0.1) ≈ 0.0997: zona morta
	}
	for _, c := range casos {
		ctl := NovoControlador(redeFixa(c.b0, c.b1, 0))
		got := ctl.Decidir(sim.Leitura{})
		if len(got.Acoes) != 1 || got.Acoes[0] != c.want {
			t.Errorf("vieses (%v, %v): %+v, esperava uma ação %+v", c.b0, c.b1, got, c.want)
		}
	}
}

func TestControladorUsaAEntrada(t *testing.T) {
	// s0 = tanh(sensor 0): sensor perto da parede (0.1) cai na zona morta,
	// sensor livre (1) aumenta x.
	ctl := NovoControlador(redeFixa(0, 2, 1))
	l := sim.Leitura{}
	l.Sensores[0] = 0.1
	if a := ctl.Decidir(l).Acoes[0]; a.Ativa {
		t.Errorf("sensor 0.1 deu %+v, esperava zona morta", a)
	}
	l.Sensores[0] = 1
	if a := ctl.Decidir(l).Acoes[0]; a != (sim.Acao{Ativa: true, Eixo: sim.EixoX, Sinal: 1}) {
		t.Errorf("sensor 1 deu %+v", a)
	}
}

func TestControladorSemRedeNaoMexe(t *testing.T) {
	var nada *Controlador
	malformada := redeFixa(2, 2, 0)
	malformada.Vieses = nil
	for _, ctl := range []*Controlador{nada, NovoControlador(nil), NovoControlador(malformada)} {
		if got := ctl.Decidir(sim.Leitura{}); len(got.Acoes) != 0 {
			t.Errorf("controlador sem rede válida devolveu %+v", got)
		}
	}
}

func TestControladorRedeFunda(t *testing.T) {
	r, err := NovaRede([]int{NEntrada, 8, 6, NSaida}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := NovoControlador(r).Decidir(sim.Leitura{}); len(got.Acoes) != 1 {
		t.Fatalf("rede 10-8-6-2 devolveu %+v", got)
	}
}
