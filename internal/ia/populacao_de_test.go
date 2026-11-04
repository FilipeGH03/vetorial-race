package ia

import (
	"errors"
	"math/rand/v2"
	"testing"
)

func baseTreinada(t *testing.T, camadas []int) *Rede {
	t.Helper()
	r, err := NovaRede(camadas, rand.New(rand.NewPCG(7, 8)))
	if err != nil {
		t.Fatal(err)
	}
	r.Geracao, r.Pontuacao, r.Pista = 140, 5321.7, "oval"
	return r
}

func TestNovaPopulacaoDe(t *testing.T) {
	base := baseTreinada(t, []int{14, 9, 7, 2})
	antes := base.Copia()
	p := ParamsPadrao() // Camadas padrão diferente da base: a base manda
	pop, err := NovaPopulacaoDe(p, base)
	if err != nil {
		t.Fatal(err)
	}
	if !iguais(base, antes) || base.Geracao != 140 {
		t.Fatal("base foi alterada")
	}
	if len(pop.Redes) != p.Tamanho || len(pop.Pontuacoes) != p.Tamanho {
		t.Fatalf("tamanho %d/%d", len(pop.Redes), len(pop.Pontuacoes))
	}
	if !iguais(pop.Redes[0], base) {
		t.Fatal("Redes[0] difere da base")
	}
	if pop.Redes[0] == base {
		t.Fatal("Redes[0] é a própria base, não cópia")
	}
	pop.Redes[0].Pesos[0].D[0] += 1
	if iguais(base, pop.Redes[0]) {
		t.Fatal("Redes[0] compartilha memória com a base")
	}
	pop.Redes[0].Pesos[0].D[0] -= 1
	for i, r := range pop.Redes[1:] {
		if len(r.Camadas) != len(base.Camadas) {
			t.Fatalf("rede %d com forma %v", i+1, r.Camadas)
		}
		for k := range base.Camadas {
			if r.Camadas[k] != base.Camadas[k] {
				t.Fatalf("rede %d com forma %v", i+1, r.Camadas)
			}
		}
		if iguais(r, base) {
			t.Fatalf("rede %d igual à base", i+1)
		}
	}
	if pop.Geracao != base.Geracao {
		t.Fatalf("Geracao %d", pop.Geracao)
	}
	if len(pop.Params.Camadas) != 4 {
		t.Fatalf("Params.Camadas %v", pop.Params.Camadas)
	}
}

func TestNovaPopulacaoDeSeed(t *testing.T) {
	base := baseTreinada(t, CamadasPadrao)
	p := ParamsPadrao()
	p.Seed = 42
	a, _ := NovaPopulacaoDe(p, base)
	b, _ := NovaPopulacaoDe(p, base)
	for i := range a.Redes {
		if !iguais(a.Redes[i], b.Redes[i]) {
			t.Fatalf("rede %d difere com a mesma seed", i)
		}
	}
	p.Seed = 43
	c, _ := NovaPopulacaoDe(p, base)
	if iguais(a.Redes[1], c.Redes[1]) {
		t.Fatal("seeds diferentes deram a mesma rede")
	}
}

func TestNovaPopulacaoDeProxima(t *testing.T) {
	base := baseTreinada(t, CamadasPadrao)
	pop, err := NovaPopulacaoDe(ParamsPadrao(), base)
	if err != nil {
		t.Fatal(err)
	}
	pop.Avaliar(brinquedo(base))
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	if pop.Geracao != 141 || len(pop.Redes) != 50 {
		t.Fatalf("geracao %d, %d redes", pop.Geracao, len(pop.Redes))
	}
	if !iguais(pop.Redes[0], base) { // a base é a melhor no brinquedo
		t.Fatal("elite perdeu a base")
	}
}

func TestNovaPopulacaoDeErros(t *testing.T) {
	p := ParamsPadrao()
	if _, err := NovaPopulacaoDe(p, nil); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("nil: %v", err)
	}
	m := baseTreinada(t, CamadasPadrao)
	m.Vieses[0] = m.Vieses[0][:3]
	if _, err := NovaPopulacaoDe(p, m); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("malformada: %v", err)
	}
	if _, err := NovaPopulacaoDe(p, &Rede{}); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("vazia: %v", err)
	}
	p.Tamanho = 1
	if _, err := NovaPopulacaoDe(p, baseTreinada(t, CamadasPadrao)); !errors.Is(err, ErrParams) {
		t.Fatalf("params: %v", err)
	}
}
