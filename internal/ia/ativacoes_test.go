package ia

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestAtivacoesUltimaIgualForward(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 9))
	formas := [][]int{{14, 12, 2}, {14, 2}, {14, 8, 6, 2}}
	for _, cam := range formas {
		for i := 0; i < 50; i++ {
			r, err := NovaRede(cam, rng)
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range r.Vieses {
				for j := range b {
					b[j] = rng.Float64()*2 - 1
				}
			}
			x := make([]float64, NEntrada)
			for j := range x {
				x[j] = rng.Float64()*2 - 1
			}
			ativ, err := r.Ativacoes(x)
			if err != nil {
				t.Fatal(err)
			}
			if len(ativ) != len(r.Camadas) {
				t.Fatalf("len = %d, quer %d", len(ativ), len(r.Camadas))
			}
			for k, a := range ativ {
				if len(a) != r.Camadas[k] {
					t.Fatalf("camada %d tem %d valores, quer %d", k, len(a), r.Camadas[k])
				}
			}
			f, _ := r.Forward(x)
			ult := ativ[len(ativ)-1]
			for j := range f {
				if math.Float64bits(f[j]) != math.Float64bits(ult[j]) {
					t.Fatalf("saída %d difere: %v x %v", j, ult[j], f[j])
				}
			}
			ativ[0][0] = 99 // não pode alterar x
			if x[0] == 99 {
				t.Fatal("ativ[0] compartilha memória com x")
			}
		}
	}
}

func TestAtivacoesEntradaErrada(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, nil)
	for _, n := range []int{0, 13, 15} {
		x := make([]float64, n)
		_, e1 := r.Ativacoes(x)
		_, e2 := r.Forward(x)
		if e1 == nil || e2 == nil || e1.Error() != e2.Error() {
			t.Fatalf("n=%d: Ativacoes %v, Forward %v", n, e1, e2)
		}
	}
}
