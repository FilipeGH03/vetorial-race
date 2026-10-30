package ia

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"
)

func TestForwardNaoMexeNaEntradaNemNaRede(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 4)))
	cp := r.Copia()
	x := []float64{.1, .2, .3, .4, .5, .6, .7, .8, .9, 1}
	orig := append([]float64(nil), x...)
	a, _ := r.Forward(x)
	b, _ := r.Forward(x)
	for i := range x {
		if x[i] != orig[i] {
			t.Fatal("Forward alterou a entrada")
		}
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("Forward não é determinístico")
		}
	}
	for k := range r.Pesos {
		for i := range r.Pesos[k].D {
			if r.Pesos[k].D[i] != cp.Pesos[k].D[i] {
				t.Fatal("Forward alterou pesos")
			}
		}
	}
}

func TestForwardSaidaSempreEmMenos1Mais1(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 9))
	r, _ := NovaRede(CamadasPadrao, rng)
	for n := 0; n < 500; n++ {
		x := make([]float64, 10)
		for i := range x {
			x[i] = rng.Float64()*2 - 1
		}
		y, err := r.Forward(x)
		if err != nil || len(y) != 2 {
			t.Fatalf("err=%v len=%d", err, len(y))
		}
		for _, v := range y {
			if math.IsNaN(v) || v < -1 || v > 1 {
				t.Fatalf("saída fora de [-1,1]: %v", v)
			}
		}
	}
}

func TestSalvarCarregarMetadadosEProfundidade(t *testing.T) {
	r, err := NovaRede([]int{10, 7, 5, 2}, rand.New(rand.NewPCG(5, 6)))
	if err != nil {
		t.Fatal(err)
	}
	r.Pista, r.Geracao, r.Pontuacao = "oval", 140, 5321.7
	caminho := filepath.Join(t.TempDir(), "sub", "pasta", "c.json")
	if err := r.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	g, err := CarregarRede(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if g.Pista != "oval" || g.Geracao != 140 || g.Pontuacao != 5321.7 {
		t.Errorf("metadados perdidos: %+v", g)
	}
	x := []float64{1, -1, .5, 0, .25, -.25, .75, 0, 1, -1}
	a, _ := r.Forward(x)
	b, _ := g.Forward(x)
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			t.Errorf("saída difere: %v vs %v", a, b)
		}
	}
}

func TestForwardEntradaNilEVazia(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, nil)
	for _, x := range [][]float64{nil, {}, make([]float64, 11)} {
		if _, err := r.Forward(x); err == nil {
			t.Errorf("entrada com %d valores deveria dar erro", len(x))
		}
	}
}
