package jogo

import (
	"errors"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"vetorial-race/internal/sim"
)

// ovalTela devolve um traço de mouse (coordenadas de tela) em volta de um oval
// de centro (cx, cy) e semieixos a, b, andando de 1 px em 1 px, com tremida de
// até tremor px em cada eixo. Anda no sentido anti-horário em mundo.
func ovalTela(cx, cy, a, b, tremor float64, rng *rand.Rand) []sim.Vec2 {
	var tr []sim.Vec2
	perim := 2 * math.Pi * (a + b) / 2
	passos := int(perim)
	for i := 0; i < passos; i++ {
		ang := 2 * math.Pi * float64(i) / float64(passos)
		x := cx + a*math.Cos(ang) + (rng.Float64()*2-1)*tremor
		yMundo := cy + b*math.Sin(ang) + (rng.Float64()*2-1)*tremor
		_, yTela := MundoParaTela(0, yMundo)
		tr = append(tr, sim.Vec2{X: x, Y: yTela})
	}
	return tr
}

func desenhar(e *Editor, tr []sim.Vec2) {
	e.Comecar(tr[0].X, tr[0].Y)
	for _, q := range tr[1:] {
		e.Arrastar(q.X, q.Y)
	}
	e.Soltar()
}

func TestTracoParaPontosTelaParaMundo(t *testing.T) {
	// Pontos bem espaçados (30 px): nenhum é descartado, só muda o y.
	tela := []sim.Vec2{{X: 100, Y: 100}, {X: 130, Y: 100}, {X: 160, Y: 100}, {X: 160, Y: 130}}
	// Sem suavizar não dá para comparar; confere a conversão pelo fim do pipeline
	// com PassosSuavizar aplicado: o centro de massa se conserva na média 1-2-1.
	p := TracoParaPontos(tela, 90)
	if len(p) != 4 {
		t.Fatalf("len = %d, quer 4", len(p))
	}
	var cx, cy float64
	for _, q := range p {
		cx += q.X / 4
		cy += q.Y / 4
	}
	wx, wy := 0.0, 0.0
	for _, q := range tela {
		x, y := TelaParaMundo(q.X, q.Y)
		wx += x / 4
		wy += y / 4
	}
	if math.Abs(cx-wx) > 1e-9 || math.Abs(cy-wy) > 1e-9 {
		t.Fatalf("centro de massa (%v,%v), quer (%v,%v): y deve virar %d - y", cx, cy, wx, wy, AlturaTela)
	}
	if wy < 500 {
		t.Fatalf("o y de tela 100..130 deveria virar ~600 em mundo, deu %v", wy)
	}
}

func TestTracoParaPontosDescartaProximos(t *testing.T) {
	// 300 pontos a 0,5 px um do outro numa reta, depois uma curva fechando: só
	// sobram pontos a >= EspacoMinTraco (antes de suavizar).
	var tela []sim.Vec2
	for i := 0; i < 600; i++ {
		tela = append(tela, sim.Vec2{X: 100 + float64(i)*0.5, Y: 100})
	}
	tela = append(tela, sim.Vec2{X: 400, Y: 300}, sim.Vec2{X: 100, Y: 300})
	p := TracoParaPontos(tela, 90)
	// ~300 px de reta / 12 + 2 cantos.
	if len(p) < 20 || len(p) > 30 {
		t.Fatalf("len = %d, esperado ~27 pontos a cada %v px", len(p), EspacoMinTraco)
	}
	if TracoParaPontos(tela[:5], 90) != nil {
		t.Fatalf("traço de 2 px deveria dar nil (menos de 3 pontos)")
	}
	if TracoParaPontos(nil, 90) != nil {
		t.Fatalf("traço vazio deveria dar nil")
	}
}

func TestTracoParaPontosCortaSobraDaLargada(t *testing.T) {
	// Oval completo e mais um pedaço repetindo o começo: o excesso é cortado,
	// então a pista continua valendo.
	rng := rand.New(rand.NewSource(1))
	tr := ovalTela(640, 360, 400, 200, 0, rng)
	tr = append(tr, tr[:len(tr)/6]...)
	e := NovoEditor("sobra")
	desenhar(e, tr)
	if e.Erro != nil {
		t.Fatalf("passar da largada deveria ser cortado, mas deu erro: %v", e.Erro)
	}
}

func TestEditorOvalComTremida(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		tr := ovalTela(640, 360, 450, 220, 5, rng)
		e := NovoEditor("oval")
		desenhar(e, tr)
		if e.Erro != nil || e.Pista == nil {
			t.Fatalf("seed %d: traço com tremida de 5 px (sem suavizar seria recusada) deveria valer: %v", seed, e.Erro)
		}
		if len(e.Pista.Centro) < sim.PontosPistaMin {
			t.Fatalf("seed %d: %d pontos reamostrados", seed, len(e.Pista.Centro))
		}
	}
}

func TestEditorOitoMostraErro(t *testing.T) {
	// Oito: dois ovais ligados no meio. Em mundo, lemniscata.
	var tr []sim.Vec2
	for i := 0; i < 700; i++ {
		ang := 2 * math.Pi * float64(i) / 700
		den := 1 + math.Sin(ang)*math.Sin(ang)
		x := 640 + 400*math.Cos(ang)/den
		yMundo := 360 + 400*math.Sin(ang)*math.Cos(ang)/den
		_, yTela := MundoParaTela(0, yMundo)
		tr = append(tr, sim.Vec2{X: x, Y: yTela})
	}
	e := NovoEditor("oito")
	desenhar(e, tr)
	if !errors.Is(e.Erro, sim.ErrPistaCruza) {
		t.Fatalf("oito deveria dar ErrPistaCruza, deu %v", e.Erro)
	}
	if e.Pista != nil {
		t.Fatalf("pista inválida não deveria ficar montada")
	}
	if !e.TemCruzamento {
		t.Fatalf("o erro deveria vir com o ponto do cruzamento")
	}
	if d := e.Cruzamento.Sub(sim.Vec2{X: 640, Y: 360}).Len(); d > 15 {
		t.Fatalf("cruzamento em %v, esperado perto do centro do oito (640,360); dist %v", e.Cruzamento, d)
	}
	if err := e.Salvar(filepath.Join(t.TempDir(), "oito.json")); err == nil {
		t.Fatalf("não deveria salvar pista inválida")
	}
}

func TestEditorTracoCurto(t *testing.T) {
	e := NovoEditor("curta")
	e.Comecar(300, 300)
	e.Soltar() // um clique só
	if !errors.Is(e.Erro, sim.ErrPistaCurta) {
		t.Fatalf("um clique deveria dar ErrPistaCurta, deu %v", e.Erro)
	}
	// Círculo pequeno demais (perímetro ~150 px < 8 pontos).
	tr := ovalTela(300, 300, 24, 24, 0, rand.New(rand.NewSource(1)))
	desenhar(e, tr)
	if e.Erro == nil || e.Pista != nil {
		t.Fatalf("círculo de 24 px de raio não deveria valer")
	}
}

func TestEditorDentroDaJanela(t *testing.T) {
	e := NovoEditor("borda")
	// Mouse arrastado bem para fora da janela: os pontos ficam presos à margem.
	tr := ovalTela(640, 360, 900, 500, 0, rand.New(rand.NewSource(1)))
	desenhar(e, tr)
	m := e.Largura / 2
	for _, q := range e.Pontos {
		if q.X < m-1e-6 || q.X > LarguraTela-m+1e-6 || q.Y < m-1e-6 || q.Y > AlturaTela-m+1e-6 {
			t.Fatalf("ponto %v fora da margem de %v px", q, m)
		}
	}
	if e.Pista != nil && !CabeNaTela(e.Pista) {
		t.Fatalf("pista aceita com paredes fora da janela")
	}
}

func TestEditorLarguraQueNaoCabe(t *testing.T) {
	e := NovoEditor("larga")
	e.Largura = 50
	tr := ovalTela(640, 360, 560, 300, 0, rand.New(rand.NewSource(1)))
	desenhar(e, tr)
	if e.Erro != nil {
		t.Fatalf("largura 50 deveria caber: %v", e.Erro)
	}
	for i := 0; i < 10; i++ {
		e.AjustarLargura(PassoLargura)
	}
	if e.Largura != 150 {
		t.Fatalf("largura %v", e.Largura)
	}
	if !errors.Is(e.Erro, ErrNaoCabe) {
		t.Fatalf("largura %v com traçado perto da borda deveria dar ErrNaoCabe, deu %v", e.Largura, e.Erro)
	}
	e.AjustarLargura(-1000)
	if e.Largura != sim.LarguraPistaMin {
		t.Fatalf("largura mínima = %v", e.Largura)
	}
	if e.Erro != nil {
		t.Fatalf("largura mínima deveria caber de novo: %v", e.Erro)
	}
	e.AjustarLargura(1e6)
	if e.Largura != sim.LarguraPistaMax {
		t.Fatalf("largura máxima = %v", e.Largura)
	}
}

func TestEditorSalvarECarregar(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	tr := ovalTela(640, 360, 450, 220, 2, rng)
	e := NovoEditor("x")
	e.Largura = 100
	desenhar(e, tr)
	if e.Erro != nil {
		t.Fatal(e.Erro)
	}
	caminho := filepath.Join(t.TempDir(), "sub", "meu_oval.json")
	if err := e.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	if e.Nome != "meu_oval" {
		t.Fatalf("nome = %q, quer o nome do arquivo", e.Nome)
	}

	// O jogo lê o mesmo arquivo: a pista é a mesma e dá para largar nela.
	p, err := sim.CarregarPista(caminho)
	if err != nil {
		t.Fatalf("o jogo não carregou o arquivo do editor: %v", err)
	}
	if p.Nome != "meu_oval" || p.Largura != 100 || len(p.Pontos) != len(e.Pontos) {
		t.Fatalf("pista lida: %q %v %d pontos", p.Nome, p.Largura, len(p.Pontos))
	}
	for i := range p.Pontos {
		if p.Pontos[i] != e.Pontos[i] {
			t.Fatalf("ponto %d mudou na ida e volta: %v != %v", i, p.Pontos[i], e.Pontos[i])
		}
	}
	if !CabeNaTela(p) {
		t.Fatalf("pista salva não cabe na janela")
	}
	if carros := sim.Largada(p, 3); len(carros) != 3 {
		t.Fatalf("largada devolveu %d carros", len(carros))
	}
	j := NovoComPista(p)
	for i := 0; i < 60; i++ {
		j.Passo(Teclas{Direita: true, Cima: true})
	}

	// Reabrir no editor devolve a mesma pista.
	e2 := NovoEditor("outro")
	if err := e2.Carregar(caminho); err != nil {
		t.Fatal(err)
	}
	if e2.Pista == nil || e2.Largura != 100 || len(e2.Pontos) != len(e.Pontos) {
		t.Fatalf("carregar: pista %v largura %v", e2.Pista != nil, e2.Largura)
	}
	// Pista em branco no arquivo ruim: erro e o editor não muda.
	ruim := filepath.Join(t.TempDir(), "ruim.json")
	if err := os.WriteFile(ruim, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e2.Carregar(ruim); !errors.Is(err, sim.ErrPistaInvalida) {
		t.Fatalf("arquivo ruim: %v", err)
	}
	if e2.Pista == nil {
		t.Fatalf("falha ao carregar não deveria apagar a pista aberta")
	}
}

func TestEditorCarregaOvalDoRepositorio(t *testing.T) {
	e := NovoEditor("oval")
	if err := e.Carregar(filepath.Join("..", "..", "pistas", "oval.json")); err != nil {
		t.Fatal(err)
	}
	if e.Pista == nil || e.Largura != 90 {
		t.Fatalf("oval.json: pista %v largura %v erro %v", e.Pista != nil, e.Largura, e.Erro)
	}
}

func TestEditorLimparEArrastarSemComecar(t *testing.T) {
	e := NovoEditor("a")
	e.Arrastar(100, 100) // sem Comecar: ignora
	e.Soltar()
	if len(e.Traco) != 0 || e.Erro != nil {
		t.Fatalf("arrastar sem começar não deveria fazer nada")
	}
	desenhar(e, ovalTela(640, 360, 300, 150, 0, rand.New(rand.NewSource(3))))
	if e.Pista == nil {
		t.Fatal(e.Erro)
	}
	e.Limpar()
	if e.Pista != nil || e.Erro != nil || len(e.Pontos) != 0 {
		t.Fatalf("Limpar deveria zerar tudo")
	}
	// Um traço novo apaga a pista anterior.
	desenhar(e, ovalTela(640, 360, 300, 150, 0, rand.New(rand.NewSource(3))))
	e.Comecar(10, 10)
	if e.Pista != nil || !e.Arrastando {
		t.Fatalf("começar um traço deveria apagar a pista")
	}
}
