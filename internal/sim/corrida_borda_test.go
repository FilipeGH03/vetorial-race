package sim

import (
	"slices"
	"testing"
)

// Depois de Terminada, Step não mexe em nada (carros congelados, Tick fixo).
func TestCorridaStepDepoisDeTerminadaNaoFazNada(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, &seguidor{p: p, vAlvo: 150})
	for i := 0; i < 3000 && !r.Terminada; i++ {
		r.Step()
	}
	if !r.Terminada {
		t.Fatal("corrida não terminou")
	}
	tick, venc := r.Tick, r.Vencedor
	carros := slices.Clone(r.Carros)
	for i := 0; i < 50; i++ {
		r.Step()
	}
	if r.Tick != tick || r.Vencedor != venc || !slices.Equal(carros, r.Carros) {
		t.Errorf("Step depois de Terminada mudou o estado")
	}
}

// Controlador nil vale como carro parado e não causa pânico.
func TestCorridaControladorNil(t *testing.T) {
	p := carregarOval(t)
	r, err := NovaCorrida(p, ConfigPadrao(), ModoVoltas, 1, []Controlador{nil, reta{}}, -1)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 120; i++ {
		r.Step()
	}
	if r.Carros[0].Vel != (Vec2{}) {
		t.Errorf("carro com controlador nil se mexeu: %v", r.Carros[0].Vel)
	}
}

// Laser e faixa fora da Eliminação; Chegada devolve cópia; laser anda pela pista.
func TestCorridaLaserForaDaEliminacaoEChegadaCopia(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoVoltas, 1, &seguidor{p: p, vAlvo: 200}, parado{})
	r.Tick = 100000
	if _, ativo := r.Laser(); ativo {
		t.Error("laser ativo no modo Voltas")
	}
	r.Tick = 0
	for i := 0; i < 20000 && !r.Terminada; i++ {
		r.Step()
	}
	if !r.Terminada || r.Vencedor != 0 {
		t.Fatalf("terminada=%v vencedor=%d", r.Terminada, r.Vencedor)
	}
	ch := r.Chegada()
	if len(ch) == 0 || ch[0] != 0 {
		t.Fatalf("Chegada = %v", ch)
	}
	ch[0] = 99
	if r.Chegada()[0] != 0 {
		t.Error("Chegada não devolve cópia")
	}
	if c := r.Classificacao(); c[0] != 0 || len(c) != 2 {
		t.Errorf("Classificacao = %v", c)
	}
}

// LaserPos e LaserFaixa ficam dentro da pista durante toda a Eliminação.
func TestCorridaLaserFaixaDentroDaPista(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{}, parado{})
	for _, tick := range []int{0, 179, 180, 600, 3000, 20000, 100000} {
		r.Tick = tick
		f := r.LaserFaixa()
		pos := r.LaserPos()
		if !naPista(p, pos) {
			t.Errorf("tick %d: LaserPos %v fora da pista", tick, pos)
		}
		if f.A == f.B {
			t.Errorf("tick %d: faixa degenerada", tick)
		}
	}
}

// Classificação na Eliminação com todos mortos pela parede (BaterMata) não pânica.
func TestCorridaEliminacaoBaterMata(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	cfg.BaterMata = true
	r, err := NovaCorrida(p, cfg, ModoEliminacao, 0, []Controlador{reta{}, reta{}, parado{}}, -1)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3000 && !r.Terminada; i++ {
		r.Step()
	}
	if !r.Terminada || r.Vencedor != 2 {
		t.Errorf("terminada=%v vencedor=%d, quer o parado (2) vivo", r.Terminada, r.Vencedor)
	}
	if len(r.Classificacao()) != 3 {
		t.Error("Classificacao com tamanho errado")
	}
}
