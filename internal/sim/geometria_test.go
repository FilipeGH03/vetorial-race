package sim

import (
	"math"
	"math/rand"
	"testing"
)

func TestDistanciaRaio(t *testing.T) {
	o := Vec2{0, 0}
	px := Vec2{1, 0} // raio para +x
	casos := []struct {
		nome   string
		d      Vec2
		a, b   Vec2
		want   float64
		wantOK bool
	}{
		{"perpendicular à frente", px, Vec2{10, -5}, Vec2{10, 5}, 10, true},
		{"perpendicular atrás", px, Vec2{-10, -5}, Vec2{-10, 5}, 0, false},
		{"ao lado, sem tocar", px, Vec2{10, 1}, Vec2{10, 5}, 0, false},
		{"na ponta do segmento", px, Vec2{10, 0}, Vec2{10, 5}, 10, true},
		{"inclinado", px, Vec2{5, -5}, Vec2{15, 5}, 10, true},
		{"paralelo fora da reta", px, Vec2{5, 1}, Vec2{15, 1}, 0, false},
		{"colinear à frente", px, Vec2{15, 0}, Vec2{5, 0}, 5, true},
		{"colinear atrás", px, Vec2{-15, 0}, Vec2{-5, 0}, 0, false},
		{"colinear com a origem dentro", px, Vec2{-5, 0}, Vec2{5, 0}, 0, true},
		{"comprimento zero sobre o raio", px, Vec2{7, 0}, Vec2{7, 0}, 7, true},
		{"comprimento zero fora do raio", px, Vec2{7, 1}, Vec2{7, 1}, 0, false},
		{"comprimento zero atrás", px, Vec2{-7, 0}, Vec2{-7, 0}, 0, false},
		{"comprimento zero na origem", px, Vec2{0, 0}, Vec2{0, 0}, 0, true},
		{"raio diagonal", Vec2{1, 1}.Norm(), Vec2{0, 10}, Vec2{10, 0}, 5 * math.Sqrt2, true},
		{"raio para -y", Vec2{0, -1}, Vec2{-3, -4}, Vec2{3, -4}, 4, true},
	}
	for _, c := range casos {
		got, ok := distanciaRaio(o, c.d, c.a, c.b)
		if ok != c.wantOK || (ok && math.Abs(got-c.want) > 1e-9) {
			t.Errorf("%s: distanciaRaio = (%v, %v), quer (%v, %v)", c.nome, got, ok, c.want, c.wantOK)
		}
		if math.IsNaN(got) || math.IsInf(got, 0) {
			t.Errorf("%s: distância não finita: %v", c.nome, got)
		}
	}
}

func TestSegmentosSeTocamComprimentoZero(t *testing.T) {
	casos := []struct {
		nome           string
		p1, p2, q1, q2 Vec2
		want           bool
	}{
		{"ponto no meio do segmento", Vec2{0, 0}, Vec2{10, 0}, Vec2{5, 0}, Vec2{5, 0}, true},
		{"ponto na ponta", Vec2{0, 0}, Vec2{10, 0}, Vec2{10, 0}, Vec2{10, 0}, true},
		{"ponto fora, colinear", Vec2{0, 0}, Vec2{10, 0}, Vec2{11, 0}, Vec2{11, 0}, false},
		{"ponto fora da reta", Vec2{0, 0}, Vec2{10, 0}, Vec2{5, 1}, Vec2{5, 1}, false},
		{"dois pontos iguais", Vec2{3, 3}, Vec2{3, 3}, Vec2{3, 3}, Vec2{3, 3}, true},
		{"dois pontos diferentes", Vec2{3, 3}, Vec2{3, 3}, Vec2{3, 4}, Vec2{3, 4}, false},
		{"cruz", Vec2{0, 0}, Vec2{10, 10}, Vec2{0, 10}, Vec2{10, 0}, true},
		{"paralelos", Vec2{0, 0}, Vec2{10, 0}, Vec2{0, 1}, Vec2{10, 1}, false},
	}
	for _, c := range casos {
		if got := segmentosSeTocam(c.p1, c.p2, c.q1, c.q2); got != c.want {
			t.Errorf("%s: segmentosSeTocam = %v, quer %v", c.nome, got, c.want)
		}
		if got := segmentosSeTocam(c.q1, c.q2, c.p1, c.p2); got != c.want {
			t.Errorf("%s (trocados): segmentosSeTocam = %v, quer %v", c.nome, got, c.want)
		}
	}
}

func TestCaixasSeparadas(t *testing.T) {
	lo, hi := Vec2{0, 0}, Vec2{10, 10}
	if caixasSeparadas(Vec2{5, 5}, Vec2{5, 5}, lo, hi) {
		t.Error("ponto dentro da caixa foi dado como separado")
	}
	if caixasSeparadas(Vec2{-5, 5}, Vec2{15, 5}, lo, hi) {
		t.Error("segmento atravessando a caixa foi dado como separado")
	}
	if !caixasSeparadas(Vec2{11, 0}, Vec2{20, 10}, lo, hi) {
		t.Error("segmento à direita da caixa não foi dado como separado")
	}
}

// TestVerticeNaoVaza: caminhos e raios mirados exatamente num vértice comum a
// dois trechos (coordenadas "tortas") tocam pelo menos um dos dois.
func TestVerticeNaoVaza(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	pt := func() Vec2 { return Vec2{rng.Float64()*1000 - 500, rng.Float64()*1000 - 500} }
	for k := 0; k < 20000; k++ {
		a, v, b, o := pt(), pt(), pt(), pt()
		if o == v {
			continue
		}
		ate := o.Add(v.Sub(o).Scale(1 + rng.Float64())) // passa do vértice
		if !cruzaTrecho(o, ate, a, v) && !cruzaTrecho(o, ate, v, b) {
			t.Fatalf("caso %d: caminho %v→%v pelo vértice %v escapou dos trechos %v e %v", k, o, ate, v, a, b)
		}
		d := v.Sub(o).Norm()
		t1, ok1 := distanciaRaio(o, d, a, v)
		t2, ok2 := distanciaRaio(o, d, v, b)
		if !ok1 && !ok2 {
			t.Fatalf("caso %d: raio de %v para o vértice %v escapou dos trechos %v e %v", k, o, v, a, b)
		}
		want := v.Sub(o).Len()
		for _, r := range []struct {
			t  float64
			ok bool
		}{{t1, ok1}, {t2, ok2}} {
			if r.ok && r.t > want*(1+1e-6)+1e-9 {
				t.Fatalf("caso %d: raio bate depois do vértice: %v > %v", k, r.t, want)
			}
		}
	}
}
