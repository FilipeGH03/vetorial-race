package ia

import (
	"errors"
	"math"
	"math/rand/v2"
	"testing"
)

// parametros devolve todos os pesos e vieses numa fila só.
func parametros(r *Rede) []float64 {
	var v []float64
	for k := range r.Pesos {
		v = append(v, r.Pesos[k].D...)
		v = append(v, r.Vieses[k]...)
	}
	return v
}

func iguais(a, b *Rede) bool {
	pa, pb := parametros(a), parametros(b)
	if len(pa) != len(pb) {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return false
		}
	}
	return true
}

// brinquedo: pontuação = -distância² aos pesos de uma rede alvo.
func brinquedo(alvo *Rede) func(*Rede) float64 {
	pa := parametros(alvo)
	return func(r *Rede) float64 {
		var s float64
		for i, v := range parametros(r) {
			d := v - pa[i]
			s += d * d
		}
		return -s
	}
}

func rodar(t *testing.T, seed uint64, geracoes int) *Populacao {
	t.Helper()
	p := ParamsPadrao()
	p.Seed = seed
	pop, err := NovaPopulacao(p)
	if err != nil {
		t.Fatal(err)
	}
	alvo, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(99, 99)))
	pont := brinquedo(alvo)
	for g := 0; g < geracoes; g++ {
		pop.Avaliar(pont)
		if err := pop.Proxima(); err != nil {
			t.Fatal(err)
		}
	}
	pop.Avaliar(pont)
	return pop
}

func TestMesmaSeedMesmasRedes(t *testing.T) {
	a, b := rodar(t, 7, 10), rodar(t, 7, 10)
	for i := range a.Redes {
		if !iguais(a.Redes[i], b.Redes[i]) {
			t.Fatalf("rede %d difere entre duas execuções com a mesma seed", i)
		}
		if a.Pontuacoes[i] != b.Pontuacoes[i] {
			t.Fatalf("nota %d difere", i)
		}
	}
	c := rodar(t, 8, 10)
	if iguais(a.Redes[0], c.Redes[0]) {
		t.Fatal("seeds diferentes deram a mesma rede")
	}
}

func TestPopulacaoInicialPadrao(t *testing.T) {
	pop, err := NovaPopulacao(ParamsPadrao())
	if err != nil {
		t.Fatal(err)
	}
	if len(pop.Redes) != 50 {
		t.Fatalf("%d redes, esperava 50", len(pop.Redes))
	}
	for _, r := range pop.Redes {
		if len(r.Camadas) != 3 || r.Camadas[1] != 12 {
			t.Fatalf("forma %v", r.Camadas)
		}
	}
	if iguais(pop.Redes[0], pop.Redes[1]) {
		t.Fatal("indivíduos iniciais iguais")
	}
}

func TestParamsNaoCompartilhaCamadas(t *testing.T) {
	p := ParamsPadrao()
	pop, _ := NovaPopulacao(p)
	p.Camadas[1] = 99
	if pop.Params.Camadas[1] != 12 {
		t.Fatal("a população guardou o slice do chamador")
	}
}

func TestElitismoOsCincoMelhoresPassamIguais(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	alvo, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(5, 5)))
	pop.Avaliar(brinquedo(alvo))
	rank := pop.Ranking()
	melhores := make([]*Rede, 5)
	notas := make([]float64, 5)
	for i := 0; i < 5; i++ {
		melhores[i] = pop.Redes[rank[i]].Copia()
		notas[i] = pop.Pontuacoes[rank[i]]
	}
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	if len(pop.Redes) != 50 || len(pop.Pontuacoes) != 50 {
		t.Fatalf("população virou %d/%d", len(pop.Redes), len(pop.Pontuacoes))
	}
	for i := 0; i < 5; i++ {
		if !iguais(pop.Redes[i], melhores[i]) {
			t.Fatalf("elite %d mudou", i)
		}
		if pop.Pontuacoes[i] != notas[i] {
			t.Fatalf("nota da elite %d mudou", i)
		}
	}
	// os demais não são cópias da elite
	for i := 5; i < 50; i++ {
		for j := 0; j < 5; j++ {
			if iguais(pop.Redes[i], melhores[j]) {
				t.Fatalf("filho %d igual à elite %d", i, j)
			}
		}
	}
	if pop.Geracao != 1 || pop.Redes[7].Geracao != 1 {
		t.Fatalf("geração = %d / %d", pop.Geracao, pop.Redes[7].Geracao)
	}
}

func TestElitismoNaoCompartilhaMemoria(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	pop.Avaliar(func(r *Rede) float64 { return r.Pesos[0].D[0] })
	antigo, _ := pop.Melhor()
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	if pop.Redes[0] == antigo {
		t.Fatal("elite é o mesmo ponteiro, não uma cópia")
	}
}

func TestMutacaoMudaCercaDe10PorCento(t *testing.T) {
	base, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 3)))
	rng := rand.New(rand.NewPCG(4, 4))
	total, mudou, retornados := 0, 0, 0
	for rep := 0; rep < 300; rep++ {
		r := base.Copia()
		retornados += Mutar(r, rng, 0.1, 0.3)
		a, b := parametros(base), parametros(r)
		for i := range a {
			total++
			if a[i] != b[i] {
				mudou++
			}
		}
	}
	if mudou != retornados {
		t.Fatalf("Mutar disse %d, contei %d", retornados, mudou)
	}
	frac := float64(mudou) / float64(total)
	if frac < 0.09 || frac > 0.11 {
		t.Fatalf("mutação mudou %.4f dos parâmetros, esperava ~0.10", frac)
	}
}

func TestMutacaoTaxaZeroERuidoGaussiano(t *testing.T) {
	base, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 3)))
	rng := rand.New(rand.NewPCG(4, 4))
	r := base.Copia()
	if n := Mutar(r, rng, 0, 0.3); n != 0 || !iguais(r, base) {
		t.Fatal("taxa 0 mudou a rede")
	}
	// com taxa 1, o ruído tem média 0 e desvio sigma
	var s, s2 float64
	N := 0
	for rep := 0; rep < 200; rep++ {
		r := base.Copia()
		Mutar(r, rng, 1, 0.3)
		a, b := parametros(base), parametros(r)
		for i := range a {
			d := b[i] - a[i]
			s += d
			s2 += d * d
			N++
		}
	}
	media := s / float64(N)
	dp := math.Sqrt(s2/float64(N) - media*media)
	if math.Abs(media) > 0.01 || math.Abs(dp-0.3) > 0.01 {
		t.Fatalf("ruído com média %.4f e desvio %.4f, esperava 0 e 0.3", media, dp)
	}
}

func TestCruzamentoUniforme(t *testing.T) {
	a, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(1, 2)))
	b, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 4)))
	a0, b0 := a.Copia(), b.Copia()
	rng := rand.New(rand.NewPCG(5, 6))
	f, err := Cruzar(a, b, rng)
	if err != nil {
		t.Fatal(err)
	}
	pa, pb, pf := parametros(a), parametros(b), parametros(f)
	doA, doB := 0, 0
	for i := range pf {
		switch pf[i] {
		case pa[i]:
			doA++
		case pb[i]:
			doB++
		default:
			t.Fatalf("parâmetro %d não veio de nenhum pai", i)
		}
	}
	if doA < len(pf)/4 || doB < len(pf)/4 {
		t.Fatalf("mistura torta: %d do a, %d do b", doA, doB)
	}
	if !iguais(a, a0) || !iguais(b, b0) {
		t.Fatal("cruzar alterou um pai")
	}
}

func TestCruzarFormasDiferentes(t *testing.T) {
	a, _ := NovaRede([]int{10, 12, 2}, nil)
	b, _ := NovaRede([]int{10, 8, 2}, nil)
	if _, err := Cruzar(a, b, rand.New(rand.NewPCG(1, 1))); !errors.Is(err, ErrParams) {
		t.Fatalf("err = %v, esperava ErrParams", err)
	}
}

func TestTorneioFavoreceMelhor(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	for i := range pop.Pontuacoes {
		pop.Pontuacoes[i] = float64(i)
	}
	soma := 0
	const n = 5000
	for i := 0; i < n; i++ {
		soma += pop.sortearTorneio()
	}
	// com torneio de 3, o índice médio esperado é 0.75*(N-1) ≈ 36.75
	if m := float64(soma) / n; m < 33 || m > 40 {
		t.Fatalf("índice médio do vencedor %.1f, esperava ~36.75", m)
	}
}

func TestBrinquedoMelhoraEm30Geracoes(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	alvo, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(99, 99)))
	pont := brinquedo(alvo)
	pop.Avaliar(pont)
	_, inicial := pop.Melhor()
	anterior := inicial
	for g := 0; g < 30; g++ {
		if err := pop.Proxima(); err != nil {
			t.Fatal(err)
		}
		pop.Avaliar(pont)
		_, m := pop.Melhor()
		if m < anterior {
			t.Fatalf("geração %d: melhor caiu de %v para %v apesar do elitismo", g+1, anterior, m)
		}
		anterior = m
	}
	// notas são negativas (-distância²): subir é chegar mais perto de 0
	if anterior <= inicial*0.7 {
		t.Fatalf("melhor nota %v -> %v, esperava melhora de pelo menos 30%%", inicial, anterior)
	}
	t.Logf("melhor nota: %.3f -> %.3f", inicial, anterior)
}

func TestNotaNaNEoPior(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	pop.Avaliar(func(r *Rede) float64 { return math.NaN() })
	pop.Pontuacoes[10] = 1
	if rank := pop.Ranking(); rank[0] != 10 {
		t.Fatalf("melhor = %d", rank[0])
	}
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
}

func TestParamsInvalidos(t *testing.T) {
	casos := map[string]func(*Params){
		"tamanho 1":    func(p *Params) { p.Tamanho = 1 },
		"elite = pop":  func(p *Params) { p.Elite = p.Tamanho },
		"elite < 0":    func(p *Params) { p.Elite = -1 },
		"torneio 0":    func(p *Params) { p.Torneio = 0 },
		"taxa > 1":     func(p *Params) { p.TaxaMutacao = 1.5 },
		"taxa NaN":     func(p *Params) { p.TaxaMutacao = math.NaN() },
		"sigma < 0":    func(p *Params) { p.Sigma = -1 },
		"camadas ruim": func(p *Params) { p.Camadas = []int{9, 2} },
	}
	for nome, mexe := range casos {
		p := ParamsPadrao()
		mexe(&p)
		if _, err := NovaPopulacao(p); err == nil {
			t.Errorf("%s: deveria dar erro", nome)
		}
	}
}

func TestRedeMaisFundaEvolui(t *testing.T) {
	p := ParamsPadrao()
	p.Camadas = []int{10, 8, 6, 2}
	p.Tamanho, p.Elite = 12, 2
	pop, err := NovaPopulacao(p)
	if err != nil {
		t.Fatal(err)
	}
	pop.Avaliar(func(r *Rede) float64 { return r.Pesos[0].D[0] })
	if err := pop.Proxima(); err != nil {
		t.Fatal(err)
	}
	if got := pop.Redes[11].Camadas; len(got) != 4 {
		t.Fatalf("forma %v", got)
	}
}

func TestDefinirNotas(t *testing.T) {
	pop, _ := NovaPopulacao(ParamsPadrao())
	if err := pop.Definir([]float64{1, 2}); !errors.Is(err, ErrParams) {
		t.Fatalf("err = %v", err)
	}
	notas := make([]float64, 50)
	notas[42] = 5
	if err := pop.Definir(notas); err != nil {
		t.Fatal(err)
	}
	if _, m := pop.Melhor(); m != 5 {
		t.Fatalf("melhor %v", m)
	}
}
