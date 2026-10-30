package sim

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const caminhoOval = "../../pistas/oval.json"

func carregarOval(t *testing.T) *Pista {
	t.Helper()
	p, err := CarregarPista(caminhoOval)
	if err != nil {
		t.Fatalf("CarregarPista(oval): %v", err)
	}
	return p
}

// area é a área com sinal do polígono fechado p (>0 se anti-horário).
func area(p []Vec2) float64 {
	s := 0.0
	for i := range p {
		s += p[i].Cross(p[(i+1)%len(p)])
	}
	return s / 2
}

func TestOvalCarrega(t *testing.T) {
	p := carregarOval(t)
	if p.Nome != "oval" || p.Largura != 90 || len(p.Pontos) != 6 {
		t.Fatalf("cabeçalho: nome=%q largura=%v pontos=%d", p.Nome, p.Largura, len(p.Pontos))
	}
	n := len(p.Centro)
	if n < PontosPistaMin {
		t.Fatalf("Centro tem %d pontos, mínimo %d", n, PontosPistaMin)
	}
	if len(p.ParedeEsq) != n || len(p.ParedeDir) != n || len(p.Checkpoints) != n {
		t.Fatalf("tamanhos: centro=%d esq=%d dir=%d checkpoints=%d", n, len(p.ParedeEsq), len(p.ParedeDir), len(p.Checkpoints))
	}
	// Perímetro do oval: 2*400 + 4*100*√2.
	if want := 800 + 400*math.Sqrt2; !perto(p.Comprimento, want) {
		t.Errorf("Comprimento = %v, want %v", p.Comprimento, want)
	}
	if !perto(p.Passo*float64(n), p.Comprimento) || math.Abs(p.Passo-EspacamentoPista) > 1 {
		t.Errorf("Passo = %v (n=%d), want ~%v com Passo*n = Comprimento", p.Passo, n, EspacamentoPista)
	}
	if p.Centro[0] != p.Pontos[0] {
		t.Errorf("Centro[0] = %v, want Pontos[0] = %v (largada)", p.Centro[0], p.Pontos[0])
	}
}

func TestOvalReamostraIgual(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	for i := range p.Centro {
		d := p.Centro[(i+1)%n].Sub(p.Centro[i]).Len()
		// Numa reta a corda é o passo; numa quina de 45° fica um pouco menor.
		if d > p.Passo+1e-9 || d < p.Passo*0.9 {
			t.Errorf("distância Centro[%d]→[%d] = %v, passo %v", i, (i+1)%n, d, p.Passo)
		}
	}
}

func TestOvalParedesFechadas(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	retas := 0
	for i := range p.Centro {
		// Numa reta a parede fica a meia largura do ponto; numa curva, mais longe
		// (miter, ou o ponto onde o laço de dentro da quina foi cortado).
		reta := math.Abs(p.Centro[i].Sub(p.Centro[(i-1+n)%n]).Cross(p.Centro[(i+1)%n].Sub(p.Centro[i]))) < 1e-6
		for _, w := range []struct {
			nome string
			p    []Vec2
		}{{"ParedeEsq", p.ParedeEsq}, {"ParedeDir", p.ParedeDir}} {
			d := w.p[i].Sub(p.Centro[i]).Len()
			cortado := w.p[i] == w.p[(i-1+n)%n] || w.p[i] == w.p[(i+1)%n]
			if reta && !cortado && !perto(d, p.Largura/2) || d < p.Largura/2-1e-9 {
				t.Errorf("%s[%d] a %v do ponto %d (reta=%v), want %v (ou mais, se curva)", w.nome, i, d, i, reta, p.Largura/2)
			}
		}
		if reta {
			retas++
		}
		// Esquerda de quem anda no sentido da pista.
		tan := p.Centro[(i+1)%n].Sub(p.Centro[(i-1+n)%n])
		if tan.Cross(p.ParedeEsq[i].Sub(p.Centro[i])) <= 0 {
			t.Errorf("ParedeEsq[%d] não está à esquerda", i)
		}
	}
	if retas < n/2 {
		t.Errorf("só %d de %d pontos em reta; o oval tem mais da metade em reta", retas, n)
	}
	// Fechadas e contínuas: todo trecho, inclusive o que liga o último ponto ao
	// primeiro, é curto. Por fora de uma quina a parede abre um pouco: passo + largura.
	for _, w := range []struct {
		nome string
		p    []Vec2
	}{{"esq", p.ParedeEsq}, {"dir", p.ParedeDir}} {
		for i := range w.p {
			if d := w.p[(i+1)%n].Sub(w.p[i]).Len(); d > p.Passo+p.Largura {
				t.Errorf("parede %s: trecho %d mede %v, longo demais para uma polilinha contínua", w.nome, i, d)
			}
		}
	}
	// O oval anda no sentido anti-horário: a parede esquerda é a de dentro.
	ac, ae, ad := area(p.Centro), area(p.ParedeEsq), area(p.ParedeDir)
	if !(ac > 0 && ae > 0 && ad > 0 && ae < ac && ac < ad) {
		t.Errorf("áreas: esq=%v centro=%v dir=%v; want 0 < esq < centro < dir", ae, ac, ad)
	}
	if _, _, ok := cruzaEntre(p.ParedeEsq, p.ParedeDir); ok {
		t.Error("as paredes se cruzam")
	}
}

func TestOvalCheckpoints(t *testing.T) {
	p := carregarOval(t)
	for i, cp := range p.Checkpoints {
		if cp.A != p.ParedeEsq[i] || cp.B != p.ParedeDir[i] {
			t.Errorf("checkpoint %d = %v, want de ParedeEsq[%d] a ParedeDir[%d]", i, cp, i, i)
		}
		// De parede a parede: nunca menor que a largura (maior nas curvas).
		if l := cp.B.Sub(cp.A).Len(); l < p.Largura-1e-9 {
			t.Errorf("checkpoint %d mede %v, want ≥ %v", i, l, p.Largura)
		}
	}
	// Largada: atravessa a pista no primeiro ponto do arquivo, (200,300), com a
	// parede esquerda por cima (a pista anda para +x ali).
	cp0 := p.Checkpoints[0]
	if m := cp0.A.Add(cp0.B).Scale(0.5); !perto(m.X, 200) || !perto(m.Y, 300) {
		t.Errorf("meio do checkpoint 0 = %v, want (200,300)", m)
	}
	if cp0.A.Y <= 300 || cp0.B.Y >= 300 {
		t.Errorf("checkpoint 0 = %v, want ParedeEsq acima e ParedeDir abaixo", cp0)
	}
}

func TestPistaQueSeCruzaDaErro(t *testing.T) {
	casos := []struct {
		nome    string
		largura float64
		pontos  []Vec2
	}{
		// Oito (gravata): os trechos 0→1 e 2→3 se cruzam em (200,200).
		{"oito", 60, []Vec2{{0, 0}, {400, 400}, {400, 0}, {0, 400}}},
		// Oito com laços maiores.
		{"oito largo", 60, []Vec2{{0, 0}, {300, 0}, {600, 600}, {900, 600}, {900, 0}, {600, 0}, {300, 600}, {0, 600}}},
		// Linha central não se cruza, mas as idas e voltas estão a 60 px com largura 90.
		{"trechos colados", 90, []Vec2{{0, 0}, {600, 0}, {600, 60}, {0, 60}}},
		// Ida e volta pelo mesmo caminho.
		{"vai e volta", 60, []Vec2{{0, 0}, {300, 0}, {600, 0}, {300, 0}}},
	}
	for _, tc := range casos {
		p, err := NovaPista(tc.nome, tc.largura, tc.pontos)
		if !errors.Is(err, ErrPistaCruza) {
			t.Errorf("%s: err = %v (pista %v), want ErrPistaCruza", tc.nome, err, p != nil)
		}
	}
}

func TestPistaQuaseColadaPassa(t *testing.T) {
	// Ida e volta a 100 px com largura 90: perto, mas sem sobrepor.
	p, err := NovaPista("grampo", 90, []Vec2{{0, 0}, {600, 0}, {600, 100}, {0, 100}})
	if err != nil {
		t.Fatalf("NovaPista: %v", err)
	}
	if _, _, ok := cruzaEntre(p.ParedeEsq, p.ParedeDir); ok {
		t.Error("as paredes se cruzam")
	}
	// Quina viva de 90° (quadrado) também passa.
	if _, err := NovaPista("quadrado", 200, []Vec2{{0, 0}, {800, 0}, {800, 800}, {0, 800}}); err != nil {
		t.Errorf("quadrado: %v", err)
	}
}

func TestPistaCurtaDaErro(t *testing.T) {
	casos := []struct {
		nome   string
		pontos []Vec2
	}{
		{"triângulo pequeno", []Vec2{{0, 0}, {40, 0}, {20, 30}}},       // ~112 px → 6 pontos
		{"quadrado de 35", []Vec2{{0, 0}, {35, 0}, {35, 35}, {0, 35}}}, // 140 px → 7 pontos
		{"dois pontos", []Vec2{{0, 0}, {500, 0}}},
		{"um ponto", []Vec2{{0, 0}}},
		{"vazia", nil},
		{"pontos repetidos", []Vec2{{5, 5}, {5, 5}, {5, 5}}},
	}
	for _, tc := range casos {
		if _, err := NovaPista(tc.nome, 40, tc.pontos); !errors.Is(err, ErrPistaCurta) {
			t.Errorf("%s: err = %v, want ErrPistaCurta", tc.nome, err)
		}
	}
}

func TestArquivoInvalidoDaErro(t *testing.T) {
	casos := []struct {
		nome string
		json string
	}{
		{"json quebrado", `{"versao": 1,`},
		{"versao 2", `{"versao":2,"nome":"x","largura":90,"centro":[[0,0],[400,0],[400,400],[0,400]]}`},
		{"sem versao", `{"nome":"x","largura":90,"centro":[[0,0],[400,0],[400,400],[0,400]]}`},
		{"nome vazio", `{"versao":1,"nome":"  ","largura":90,"centro":[[0,0],[400,0],[400,400],[0,400]]}`},
		{"largura 39", `{"versao":1,"nome":"x","largura":39,"centro":[[0,0],[400,0],[400,400],[0,400]]}`},
		{"largura 201", `{"versao":1,"nome":"x","largura":201,"centro":[[0,0],[400,0],[400,400],[0,400]]}`},
		{"ponto com 3 coordenadas", `{"versao":1,"nome":"x","largura":90,"centro":[[0,0,1],[400,0],[400,400],[0,400]]}`},
		{"ponto com 1 coordenada", `{"versao":1,"nome":"x","largura":90,"centro":[[0],[400,0],[400,400],[0,400]]}`},
		{"centro não é lista", `{"versao":1,"nome":"x","largura":90,"centro":"oval"}`},
	}
	for _, tc := range casos {
		if _, err := LerPista(strings.NewReader(tc.json)); !errors.Is(err, ErrPistaInvalida) {
			t.Errorf("%s: err = %v, want ErrPistaInvalida", tc.nome, err)
		}
	}
	// Limites da largura são válidos.
	for _, l := range []float64{40, 200} {
		if _, err := NovaPista("x", l, []Vec2{{0, 0}, {800, 0}, {800, 800}, {0, 800}}); err != nil {
			t.Errorf("largura %v: %v", l, err)
		}
	}
	if _, err := CarregarPista(filepath.Join(t.TempDir(), "nao-existe.json")); err == nil {
		t.Error("arquivo inexistente: want erro")
	}
}

func TestSalvarECarregarDevolveOMesmo(t *testing.T) {
	dir := t.TempDir()
	oval := carregarOval(t)
	quebrada, err := NovaPista("quebrada \"1\"", 73.25, []Vec2{
		{100.125, 100}, {500.3, 120.7}, {610, 333.333333333}, {420.5, 560}, {130, 480.1},
	})
	if err != nil {
		t.Fatalf("NovaPista: %v", err)
	}
	for _, orig := range []*Pista{oval, quebrada} {
		caminho := filepath.Join(dir, "p.json")
		if err := SalvarPista(caminho, orig); err != nil {
			t.Fatalf("%s: SalvarPista: %v", orig.Nome, err)
		}
		volta, err := CarregarPista(caminho)
		if err != nil {
			t.Fatalf("%s: CarregarPista: %v", orig.Nome, err)
		}
		if !reflect.DeepEqual(orig, volta) {
			t.Errorf("%s: salvar e carregar mudou a pista", orig.Nome)
		}
	}
}

func TestSalvarOvalDaOMesmoArquivo(t *testing.T) {
	want, err := os.ReadFile(caminhoOval)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := EscreverPista(&got, carregarOval(t)); err != nil {
		t.Fatal(err)
	}
	if norm := bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")); !bytes.Equal(got.Bytes(), norm) {
		t.Errorf("EscreverPista(oval) =\n%s\nwant\n%s", got.Bytes(), norm)
	}
}

func TestSalvarPistaInvalidaDaErro(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "x.json")
	ruim := &Pista{Nome: "", Largura: 90, Pontos: []Vec2{{0, 0}, {400, 0}, {400, 400}}}
	if err := SalvarPista(caminho, ruim); !errors.Is(err, ErrPistaInvalida) {
		t.Errorf("nome vazio: err = %v, want ErrPistaInvalida", err)
	}
	if err := SalvarPista(caminho, nil); err == nil {
		t.Error("pista nula: want erro")
	}
	if _, err := os.Stat(caminho); !os.IsNotExist(err) {
		t.Error("pista inválida não deveria gravar arquivo")
	}
}

func TestSegmentosSeTocam(t *testing.T) {
	casos := []struct {
		nome           string
		p1, p2, q1, q2 Vec2
		want           bool
	}{
		{"cruz", Vec2{0, 0}, Vec2{2, 2}, Vec2{0, 2}, Vec2{2, 0}, true},
		{"paralelos", Vec2{0, 0}, Vec2{2, 0}, Vec2{0, 1}, Vec2{2, 1}, false},
		{"T", Vec2{0, 0}, Vec2{2, 0}, Vec2{1, 0}, Vec2{1, 5}, true},
		{"colineares separados", Vec2{0, 0}, Vec2{1, 0}, Vec2{2, 0}, Vec2{3, 0}, false},
		{"colineares sobrepostos", Vec2{0, 0}, Vec2{2, 0}, Vec2{1, 0}, Vec2{3, 0}, true},
		{"quase", Vec2{0, 0}, Vec2{2, 0}, Vec2{1, 0.1}, Vec2{1, 5}, false},
	}
	for _, tc := range casos {
		if got := segmentosSeTocam(tc.p1, tc.p2, tc.q1, tc.q2); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.nome, got, tc.want)
		}
	}
}
