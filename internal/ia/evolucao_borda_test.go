package ia

import (
	"errors"
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"
)

func TestBordaSeedsDiferentesRedesDiferentes(t *testing.T) {
	a, b := rodar(t, 1, 3), rodar(t, 2, 3)
	if iguais(a.Redes[0], b.Redes[0]) {
		t.Fatal("seeds diferentes deram a mesma rede")
	}
}

func TestBordaMesmaSeedVariasGeracoes(t *testing.T) {
	a, b := rodar(t, 7, 15), rodar(t, 7, 15)
	for i := range a.Redes {
		if !iguais(a.Redes[i], b.Redes[i]) {
			t.Fatalf("rede %d difere com a mesma seed", i)
		}
		if a.Pontuacoes[i] != b.Pontuacoes[i] {
			t.Fatalf("nota %d difere", i)
		}
	}
}

func TestBordaElite0Tamanho2Torneio1(t *testing.T) {
	for _, p := range []Params{
		{Camadas: []int{NEntrada, 4, 2}, Tamanho: 2, Elite: 0, Torneio: 1, TaxaMutacao: 1, Sigma: 0, Seed: 3},
		{Camadas: []int{NEntrada, 4, 2}, Tamanho: 2, Elite: 1, Torneio: 50, TaxaMutacao: 0, Sigma: 0.3, Seed: 3},
		{Camadas: []int{NEntrada, 6, 5, 2}, Tamanho: 7, Elite: 6, Torneio: 3, TaxaMutacao: 0.1, Sigma: 0.3, Seed: 0},
	} {
		pop, err := NovaPopulacao(p)
		if err != nil {
			t.Fatal(err)
		}
		for g := 0; g < 5; g++ {
			pop.Avaliar(func(r *Rede) float64 { return rand.Float64() })
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
			if len(pop.Redes) != p.Tamanho {
				t.Fatalf("tamanho %d", len(pop.Redes))
			}
		}
		if pop.Geracao != 5 {
			t.Fatalf("geracao %d", pop.Geracao)
		}
	}
}

func TestBordaProximaNaoAlteraPaisNemElite(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	alvo, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(5, 5)))
	pop.Avaliar(brinquedo(alvo))
	antes := make([]*Rede, len(pop.Redes))
	for i, r := range pop.Redes {
		antes[i] = r.Copia()
	}
	velhas := append([]*Rede(nil), pop.Redes...)
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	for i := range velhas {
		if !iguais(velhas[i], antes[i]) {
			t.Fatalf("rede antiga %d foi alterada por Proxima", i)
		}
	}
	for _, r := range pop.Redes {
		for _, v := range velhas {
			if r == v {
				t.Fatal("nova geração reaproveita ponteiro da anterior")
			}
		}
	}
}

func TestBordaSemElitismoSemMutacaoNaoInventaValores(t *testing.T) {
	// cruzamento uniforme com mutação 0: todo parâmetro do filho existe em algum pai da geração.
	p := ParamsPadrao()
	p.TaxaMutacao = 0
	p.Elite = 0
	pop, _ := NovaPopulacao(p)
	pop.Avaliar(func(r *Rede) float64 { return 0 })
	vistos := map[float64]bool{}
	for _, r := range pop.Redes {
		for _, v := range parametros(r) {
			vistos[v] = true
		}
	}
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	for _, r := range pop.Redes {
		for _, v := range parametros(r) {
			if !vistos[v] {
				t.Fatalf("valor %v não veio de nenhum pai", v)
			}
		}
	}
}

func TestBordaMutarSigmaZeroENaoMudaValores(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(1, 1)))
	c := r.Copia()
	n := Mutar(r, rand.New(rand.NewPCG(2, 2)), 1, 0)
	if !iguais(r, c) {
		t.Fatal("sigma 0 alterou a rede")
	}
	if n != len(parametros(r)) {
		t.Fatalf("taxa 1 mutou %d de %d", n, len(parametros(r)))
	}
}

func TestBordaDefinirAvaliaFora(t *testing.T) {
	a, _ := NovaPopulacao(ParamsPadrao())
	b, _ := NovaPopulacao(ParamsPadrao())
	alvo, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(5, 5)))
	f := brinquedo(alvo)
	notas := make([]float64, len(b.Redes))
	for i, r := range b.Redes {
		notas[i] = f(r)
	}
	a.Avaliar(f)
	if err := b.Definir(notas); err != nil {
		t.Fatal(err)
	}
	_ = a.Proxima()
	_ = b.Proxima()
	for i := range a.Redes {
		if !iguais(a.Redes[i], b.Redes[i]) {
			t.Fatalf("Avaliar e Definir divergem na rede %d", i)
		}
	}
	if b.Definir(notas[:3]) == nil {
		t.Fatal("Definir com tamanho errado devia falhar")
	}
}

func TestBordaInfinitosNaoQuebram(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	pop.Avaliar(func(r *Rede) float64 { return math.Inf(1) })
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	pop.Avaliar(func(r *Rede) float64 { return math.NaN() })
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
}

func TestBordaParamsInvalidosExtras(t *testing.T) {
	base := ParamsPadrao()
	casos := map[string]func(*Params){
		"nan taxa":      func(p *Params) { p.TaxaMutacao = math.NaN() },
		"taxa>1":        func(p *Params) { p.TaxaMutacao = 1.5 },
		"sigma neg":     func(p *Params) { p.Sigma = -1 },
		"sigma inf":     func(p *Params) { p.Sigma = math.Inf(1) },
		"elite=tamanho": func(p *Params) { p.Elite = p.Tamanho },
		"tamanho 1":     func(p *Params) { p.Tamanho = 1; p.Elite = 0 },
		"torneio 0":     func(p *Params) { p.Torneio = 0 },
		"camadas curta": func(p *Params) { p.Camadas = []int{NEntrada} },
		"camadas ruim":  func(p *Params) { p.Camadas = []int{9, 12, 2} },
		"camadas nil":   func(p *Params) { p.Camadas = nil },
	}
	for nome, f := range casos {
		p := base
		p.Camadas = append([]int(nil), base.Camadas...)
		f(&p)
		pop, err := NovaPopulacao(p)
		if err == nil || pop != nil {
			t.Errorf("%s: esperava erro", nome)
		}
	}
	_ = errors.Is
}

func TestBordaMelhorSalvaECarrega(t *testing.T) {
	pop := rodar(t, 4, 10)
	m, nota := pop.Melhor()
	m.Pista = "oval"
	dir := filepath.Join(t.TempDir(), "sub")
	cam := filepath.Join(dir, "c.json")
	if err := m.Salvar(cam); err != nil {
		t.Fatal(err)
	}
	c, err := CarregarRede(cam)
	if err != nil {
		t.Fatal(err)
	}
	if !iguais(m, c) || c.Pontuacao != nota {
		t.Fatalf("ida e volta divergiu (nota %v vs %v)", c.Pontuacao, nota)
	}
	if c.Geracao != pop.Geracao {
		t.Fatalf("geracao %d vs %d", c.Geracao, pop.Geracao)
	}
}

func TestBordaCruzarRedeInvalida(t *testing.T) {
	a, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(1, 1)))
	if _, err := Cruzar(a, &Rede{}, rand.New(rand.NewPCG(1, 1))); err == nil {
		t.Fatal("rede vazia devia dar erro")
	}
}
