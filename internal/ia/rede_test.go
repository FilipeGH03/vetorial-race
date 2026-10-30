package ia

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tanh1 = 0.7615941559557649 // tanh(1), valor de tabela

func perto(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func TestMulVecContaAMao(t *testing.T) {
	// [1 2 3; 4 5 6] · [1 0 -1] = [-2, -2]
	m := Matriz{L: 2, C: 3, D: []float64{1, 2, 3, 4, 5, 6}}
	y, err := m.MulVec([]float64{1, 0, -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(y) != 2 || y[0] != -2 || y[1] != -2 {
		t.Fatalf("M·x = %v, esperava [-2 -2]", y)
	}
	if m.At(1, 2) != 6 {
		t.Fatalf("At(1,2) = %v", m.At(1, 2))
	}
}

func TestMulVecTamanhoErrado(t *testing.T) {
	m := NovaMatriz(2, 3)
	if _, err := m.MulVec([]float64{1, 2}); err == nil {
		t.Fatal("vetor curto deveria dar erro")
	}
	if err := m.MulVecEm(make([]float64, 3), []float64{1, 2, 3}); err == nil {
		t.Fatal("destino de tamanho errado deveria dar erro")
	}
}

// redeConhecida é uma 10-2-2 de pesos simples:
//
//	h0 = tanh(1·x0 + 0·...        + 0)      h1 = tanh(0.5·x1 - 0.25·x2 + 0.1)
//	y0 = tanh(2·h0 + 0.5·h1 - 0.5)          y1 = tanh(-1·h0 + 1·h1)
func redeConhecida() *Rede {
	r := vazia([]int{10, 2, 2})
	r.Pesos[0].Set(0, 0, 1)
	r.Pesos[0].Set(1, 1, 0.5)
	r.Pesos[0].Set(1, 2, -0.25)
	r.Vieses[0][1] = 0.1
	r.Pesos[1].Set(0, 0, 2)
	r.Pesos[1].Set(0, 1, 0.5)
	r.Vieses[1][0] = -0.5
	r.Pesos[1].Set(1, 0, -1)
	r.Pesos[1].Set(1, 1, 1)
	return r
}

func TestForwardPesosConhecidos(t *testing.T) {
	r := redeConhecida()

	// Entrada toda zero: h0 = tanh(0) = 0, h1 = tanh(0.1).
	x := make([]float64, 10)
	got, err := r.Forward(x)
	if err != nil {
		t.Fatal(err)
	}
	h1 := math.Tanh(0.1)
	want := []float64{math.Tanh(0.5*h1 - 0.5), math.Tanh(h1)}
	for i := range want {
		if !perto(got[i], want[i]) {
			t.Fatalf("zero: y[%d] = %v, esperava %v", i, got[i], want[i])
		}
	}

	// x0 = 1, x1 = 2, x2 = 4: h0 = tanh(1) (valor de tabela), h1 = tanh(1 - 1 + 0.1) = tanh(0.1).
	x[0], x[1], x[2] = 1, 2, 4
	got, err = r.Forward(x)
	if err != nil {
		t.Fatal(err)
	}
	want = []float64{
		math.Tanh(2*tanh1 + 0.5*h1 - 0.5),
		math.Tanh(-tanh1 + h1),
	}
	for i := range want {
		if !perto(got[i], want[i]) {
			t.Fatalf("y[%d] = %v, esperava %v", i, got[i], want[i])
		}
	}
	// Os mesmos valores calculados à parte (Python, fora do Go).
	if !perto(got[0], 0.7905971651020123) || !perto(got[1], -0.579643837244701) {
		t.Fatalf("y = %v, a conta à parte deu [0.7905971651020123 -0.579643837244701]", got)
	}
}

func TestForwardSaturaEmMenos1Mais1(t *testing.T) {
	r := redeConhecida()
	x := make([]float64, 10)
	x[0] = 1000
	got, err := r.Forward(x)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range got {
		if v < -1 || v > 1 {
			t.Fatalf("y[%d] = %v fora de [-1,1]", i, v)
		}
	}
	if !perto(got[0], math.Tanh(2+0.5*math.Tanh(0.1)-0.5)) {
		t.Fatalf("y0 = %v", got[0])
	}
}

func TestForwardEntradaTamanhoErrado(t *testing.T) {
	r := redeConhecida()
	for _, n := range []int{0, 9, 11} {
		if _, err := r.Forward(make([]float64, n)); err == nil {
			t.Fatalf("entrada com %d valores deveria dar erro", n)
		}
	}
}

func TestNovaRede10_12_2(t *testing.T) {
	r, err := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(7, 7)))
	if err != nil {
		t.Fatal(err)
	}
	if r.Pesos[0].L != 12 || r.Pesos[0].C != 10 || r.Pesos[1].L != 2 || r.Pesos[1].C != 12 {
		t.Fatalf("formas erradas: %dx%d e %dx%d", r.Pesos[0].L, r.Pesos[0].C, r.Pesos[1].L, r.Pesos[1].C)
	}
	if len(r.Vieses[0]) != 12 || len(r.Vieses[1]) != 2 {
		t.Fatal("vieses com tamanho errado")
	}
	lim := math.Sqrt(6.0 / 22)
	for _, v := range r.Pesos[0].D {
		if math.Abs(v) > lim {
			t.Fatalf("peso %v acima do limite %v", v, lim)
		}
	}
	out, err := r.Forward([]float64{1, 0.5, 0, 0.2, 1, 0.3, 0.7, 0.1, -0.1, 0.4})
	if err != nil || len(out) != 2 {
		t.Fatalf("forward: %v %v", out, err)
	}
}

func TestNovaRedeMesmaSeedMesmaRede(t *testing.T) {
	a, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 4)))
	b, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(3, 4)))
	for k := range a.Pesos {
		for i := range a.Pesos[k].D {
			if a.Pesos[k].D[i] != b.Pesos[k].D[i] {
				t.Fatal("mesma seed deveria dar os mesmos pesos")
			}
		}
	}
}

func TestNovaRedeCamadasInvalidas(t *testing.T) {
	for _, c := range [][]int{nil, {10}, {9, 12, 2}, {10, 12, 3}, {10, 0, 2}, {10, -1, 2}} {
		if _, err := NovaRede(c, nil); !errors.Is(err, ErrCerebroInvalido) {
			t.Errorf("camadas %v: erro = %v, esperava ErrCerebroInvalido", c, err)
		}
	}
	if _, err := NovaRede([]int{10, 8, 6, 2}, nil); err != nil {
		t.Errorf("10-8-6-2 deveria valer: %v", err)
	}
}

func TestSalvarCarregarMesmaSaida(t *testing.T) {
	r, _ := NovaRede(CamadasPadrao, rand.New(rand.NewPCG(11, 12)))
	r.Pista, r.Geracao, r.Pontuacao = "oval", 140, 5321.7
	for i := range r.Vieses[0] {
		r.Vieses[0][i] = float64(i)/10 - 0.5
	}
	caminho := filepath.Join(t.TempDir(), "sub", "teste.json") // a pasta não existe ainda
	if err := r.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	c, err := CarregarRede(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if c.Pista != "oval" || c.Geracao != 140 || c.Pontuacao != 5321.7 {
		t.Fatalf("metadados: %q %d %v", c.Pista, c.Geracao, c.Pontuacao)
	}
	rng := rand.New(rand.NewPCG(5, 6))
	for n := 0; n < 50; n++ {
		x := make([]float64, 10)
		for i := range x {
			x[i] = rng.Float64()*2 - 1
		}
		a, _ := r.Forward(x)
		b, err := c.Forward(x)
		if err != nil {
			t.Fatal(err)
		}
		for i := range a {
			if a[i] != b[i] { // idêntico, bit a bit
				t.Fatalf("saída diferente depois de salvar/carregar: %v vs %v", a, b)
			}
		}
	}
}

func TestSalvarFormatoDoContrato(t *testing.T) {
	r := redeConhecida()
	r.Pista = "oval"
	caminho := filepath.Join(t.TempDir(), "c.json")
	if err := r.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	dados, _ := os.ReadFile(caminho)
	var m map[string]json.RawMessage
	if err := json.Unmarshal(dados, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"versao", "camadas", "ativacao", "pesos", "vieses", "pista", "geracao", "pontuacao"} {
		if _, ok := m[k]; !ok {
			t.Errorf("falta o campo %q", k)
		}
	}
	if string(m["versao"]) != "1" || string(m["ativacao"]) != `"tanh"` {
		t.Errorf("versao/ativacao: %s %s", m["versao"], m["ativacao"])
	}
	var pesos [][][]float64
	if err := json.Unmarshal(m["pesos"], &pesos); err != nil {
		t.Fatal(err)
	}
	// linhas = neurônios de saída: pesos[0] é 2 linhas de 10; pesos[1] é 2 linhas de 2.
	if len(pesos[0]) != 2 || len(pesos[0][0]) != 10 || pesos[0][1][1] != 0.5 || pesos[0][1][2] != -0.25 {
		t.Errorf("pesos[0] fora da convenção linhas = saída: %v", pesos[0])
	}
	if pesos[1][1][0] != -1 {
		t.Errorf("pesos[1] = %v", pesos[1])
	}
	// Não deixa temporário para trás.
	ents, _ := os.ReadDir(filepath.Dir(caminho))
	if len(ents) != 1 {
		t.Errorf("sobrou lixo na pasta: %v", ents)
	}
}

func TestSalvarSobrescreve(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "c.json")
	a := redeConhecida()
	a.Geracao = 1
	b := redeConhecida()
	b.Geracao = 2
	if err := a.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	if err := b.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	c, err := CarregarRede(caminho)
	if err != nil || c.Geracao != 2 {
		t.Fatalf("geração = %v, err = %v", c, err)
	}
}

func TestSalvarRecusaNaN(t *testing.T) {
	r := redeConhecida()
	r.Pesos[0].D[0] = math.NaN()
	caminho := filepath.Join(t.TempDir(), "c.json")
	if err := r.Salvar(caminho); err == nil {
		t.Fatal("NaN deveria dar erro")
	}
	if _, err := os.Stat(caminho); err == nil {
		t.Fatal("não deveria criar o arquivo")
	}
}

func TestSalvarRedeMalformada(t *testing.T) {
	r := redeConhecida()
	r.Vieses[1] = r.Vieses[1][:1]
	if err := r.Salvar(filepath.Join(t.TempDir(), "c.json")); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("erro = %v", err)
	}
	if _, err := r.Forward(make([]float64, 10)); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("forward em rede malformada: %v", err)
	}
}

// brutoValido devolve o JSON de uma rede 10-2-2 (valores 0) para os testes de
// arquivo corrompido mexerem.
func brutoValido() map[string]any {
	zeros := func(l, c int) [][]float64 {
		m := make([][]float64, l)
		for i := range m {
			m[i] = make([]float64, c)
		}
		return m
	}
	return map[string]any{
		"versao":   1,
		"camadas":  []int{10, 2, 2},
		"ativacao": "tanh",
		"pesos":    [][][]float64{zeros(2, 10), zeros(2, 2)},
		"vieses":   [][]float64{{0, 0}, {0, 0}},
		"pista":    "x", "geracao": 0, "pontuacao": 0,
	}
}

func carregarBruto(t *testing.T, m map[string]any) error {
	t.Helper()
	dados, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	caminho := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(caminho, dados, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = CarregarRede(caminho)
	return err
}

func TestCarregarBrutoValido(t *testing.T) {
	if err := carregarBruto(t, brutoValido()); err != nil {
		t.Fatalf("o arquivo base deveria carregar: %v", err)
	}
}

func TestCarregarCamadasErradas(t *testing.T) {
	casos := map[string][]int{
		"entrada 9":  {9, 2, 2},
		"entrada 11": {11, 2, 2},
		"saida 3":    {10, 2, 3},
		"saida 1":    {10, 2, 1},
		"uma camada": {10},
		"sem camada": {},
		"zero":       {10, 0, 2},
	}
	for nome, camadas := range casos {
		m := brutoValido()
		m["camadas"] = camadas
		if err := carregarBruto(t, m); !errors.Is(err, ErrCerebroInvalido) {
			t.Errorf("%s: erro = %v, esperava ErrCerebroInvalido", nome, err)
		}
	}
	m := brutoValido()
	delete(m, "camadas")
	if err := carregarBruto(t, m); !errors.Is(err, ErrCerebroInvalido) {
		t.Errorf("sem camadas: %v", err)
	}
}

func TestCarregarCamadasGigantesNaoAlocam(t *testing.T) {
	m := brutoValido()
	m["camadas"] = []int{10, 1 << 40, 2}
	if err := carregarBruto(t, m); !errors.Is(err, ErrCerebroInvalido) {
		t.Fatalf("erro = %v", err)
	}
}

func TestCarregarPesosOuViesesNaoCombinam(t *testing.T) {
	zeros := func(l, c int) [][]float64 {
		m := make([][]float64, l)
		for i := range m {
			m[i] = make([]float64, c)
		}
		return m
	}
	casos := map[string]func(m map[string]any){
		"linhas a menos":    func(m map[string]any) { m["pesos"] = [][][]float64{zeros(1, 10), zeros(2, 2)} },
		"colunas a menos":   func(m map[string]any) { m["pesos"] = [][][]float64{zeros(2, 9), zeros(2, 2)} },
		"matriz a menos":    func(m map[string]any) { m["pesos"] = [][][]float64{zeros(2, 10)} },
		"matriz a mais":     func(m map[string]any) { m["pesos"] = [][][]float64{zeros(2, 10), zeros(2, 2), zeros(2, 2)} },
		"transposta":        func(m map[string]any) { m["pesos"] = [][][]float64{zeros(10, 2), zeros(2, 2)} },
		"vies a menos":      func(m map[string]any) { m["vieses"] = [][]float64{{0}, {0, 0}} },
		"vieses a menos":    func(m map[string]any) { m["vieses"] = [][]float64{{0, 0}} },
		"pesos nulo":        func(m map[string]any) { m["pesos"] = nil },
		"vieses nulo":       func(m map[string]any) { m["vieses"] = nil },
		"versao 2":          func(m map[string]any) { m["versao"] = 2 },
		"sem versao":        func(m map[string]any) { delete(m, "versao") },
		"ativacao relu":     func(m map[string]any) { m["ativacao"] = "relu" },
		"sem ativacao":      func(m map[string]any) { delete(m, "ativacao") },
		"peso nao numerico": func(m map[string]any) { m["pesos"] = "oi" },
	}
	for nome, mexe := range casos {
		m := brutoValido()
		mexe(m)
		if err := carregarBruto(t, m); !errors.Is(err, ErrCerebroInvalido) {
			t.Errorf("%s: erro = %v, esperava ErrCerebroInvalido", nome, err)
		}
	}
}

func TestCarregarArquivoRuim(t *testing.T) {
	dir := t.TempDir()
	if _, err := CarregarRede(filepath.Join(dir, "nao-existe.json")); !os.IsNotExist(err) {
		t.Errorf("arquivo ausente: %v", err)
	}
	for nome, conteudo := range map[string]string{
		"vazio":    "",
		"lixo":     "isto não é json",
		"cortado":  `{"versao": 1, "camadas": [10, 2`,
		"array":    "[]",
		"nulo":     "null",
		"so chave": "{}",
	} {
		caminho := filepath.Join(dir, strings.ReplaceAll(nome, " ", "_")+".json")
		if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := CarregarRede(caminho); !errors.Is(err, ErrCerebroInvalido) {
			t.Errorf("%s: erro = %v, esperava ErrCerebroInvalido", nome, err)
		}
	}
}

func TestCopiaEhIndependente(t *testing.T) {
	a := redeConhecida()
	b := a.Copia()
	b.Pesos[0].Set(0, 0, 99)
	b.Vieses[1][0] = 99
	b.Camadas[1] = 99
	if a.Pesos[0].At(0, 0) != 1 || a.Vieses[1][0] != -0.5 || a.Camadas[1] != 2 {
		t.Fatal("a cópia compartilha memória com a original")
	}
}
