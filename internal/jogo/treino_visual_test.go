package jogo

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"vetorial-race/internal/ia"
)

const pistaOval = "../../pistas/oval.json"

func novoVisual(t *testing.T, seed uint64) *TreinoVisual {
	t.Helper()
	v, err := NovoTreinoVisual(pistaOval, t.TempDir(), seed)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// rodarGeracao dá Passo até a geração em andamento fechar (Geracao() sobe) e
// devolve quantos ticks gastou.
func rodarGeracao(t *testing.T, v *TreinoVisual) int {
	t.Helper()
	g, n := v.Geracao(), 0
	for v.Geracao() == g {
		if v.Erro != nil {
			t.Fatal(v.Erro)
		}
		v.Passo()
		if n++; n > 6*ia.NLargadas*3600 {
			t.Fatal("geração não terminou")
		}
	}
	return n
}

func TestTreinoVisualComecaComPopulacaoNaLargada(t *testing.T) {
	v := novoVisual(t, 1)
	if len(v.Individuos) != 50 {
		t.Fatalf("%d indivíduos, quero 50 (D5)", len(v.Individuos))
	}
	if v.Geracao() != 0 || v.Largada != 0 || v.Rodando() != 50 {
		t.Fatalf("geração %d, largada %d, rodando %d", v.Geracao(), v.Largada, v.Rodando())
	}
	l0 := v.Treino.Largada(0)
	for i, ind := range v.Individuos {
		if ind.Carro.Pos != l0.Pos || ind.Carro.Angulo() != l0.Angulo() {
			t.Fatalf("indivíduo %d não largou de Largada(0)", i)
		}
	}
	if v.Nome != "oval" || v.Velocidade() != Velocidades[VelocidadePadrao] {
		t.Fatalf("nome %q, velocidade %d", v.Nome, v.Velocidade())
	}
}

// Prova de que a tela treina de verdade: as notas da geração fechada na tela
// são, bit a bit, as de ia.Treino.Rodar (6 largadas, média) para as mesmas
// redes. Passo a passo e RodarDe são o mesmo tick.
func TestTreinoVisualNotasIguaisAoTreino(t *testing.T) {
	v := novoVisual(t, 7)
	redes := make([]*ia.Rede, len(v.Pop.Redes))
	for i, r := range v.Pop.Redes {
		redes[i] = r.Copia()
	}
	rodarGeracao(t, v)
	if len(v.Historico) != 1 {
		t.Fatalf("historico com %d pontos, quero 1", len(v.Historico))
	}
	ref, err := ia.NovoTreino(v.Pista)
	if err != nil {
		t.Fatal(err)
	}
	melhor, soma := math.Inf(-1), 0.0
	for _, r := range redes { // v.Pop.Redes já é a geração 1: as notas da 0 vêm do Historico
		a := ref.Rodar(r)
		soma += a.Pontuacao
		melhor = max(melhor, a.Pontuacao)
	}
	h := v.Historico[0]
	if h.Melhor != melhor {
		t.Errorf("melhor da geração %v, ia.Treino.Rodar dá %v", h.Melhor, melhor)
	}
	if h.Media != soma/float64(len(redes)) {
		t.Errorf("média da geração %v, ia.Treino.Rodar dá %v", h.Media, soma/float64(len(redes)))
	}
	if v.Melhor == nil || v.MelhorNota != melhor {
		t.Fatalf("melhor cérebro %v nota %v, quero nota %v", v.Melhor != nil, v.MelhorNota, melhor)
	}
	if got := ref.Rodar(v.Melhor).Pontuacao; got != melhor {
		t.Errorf("o cérebro guardado reproduz %v, a nota era %v", got, melhor)
	}
	if v.Geracao() != 1 || v.Largada != 0 || v.Rodando() != 50 {
		t.Errorf("depois da geração 0: geração %d, largada %d, rodando %d", v.Geracao(), v.Largada, v.Rodando())
	}
}

func TestTreinoVisualRodadasPercorremAsSeisLargadas(t *testing.T) {
	v := novoVisual(t, 1)
	visto := []int{0}
	g := v.Geracao()
	for v.Geracao() == g {
		v.Passo()
		if ult := visto[len(visto)-1]; v.Largada != ult && v.Geracao() == g {
			visto = append(visto, v.Largada)
		}
	}
	if len(visto) != ia.NLargadas {
		t.Fatalf("rodadas vistas %v, quero 0..%d", visto, ia.NLargadas-1)
	}
	for k, l := range visto {
		if l != k {
			t.Fatalf("rodadas vistas %v fora de ordem", visto)
		}
	}
}

func TestTreinoVisualPontuacaoMelhoraEmPoucasGeracoes(t *testing.T) {
	if testing.Short() {
		t.Skip("treina 6 gerações")
	}
	v := novoVisual(t, 1)
	for len(v.Historico) < 6 {
		rodarGeracao(t, v)
	}
	h := v.Historico
	if h[len(h)-1].Melhor <= h[0].Melhor {
		t.Errorf("melhor não subiu em 6 gerações: %.1f -> %.1f", h[0].Melhor, h[len(h)-1].Melhor)
	}
	for i := 1; i < len(h); i++ { // elitismo: o melhor nunca piora
		if h[i].Melhor < h[i-1].Melhor {
			t.Errorf("melhor caiu da geração %d para a %d: %.1f -> %.1f", h[i-1].Geracao, h[i].Geracao, h[i-1].Melhor, h[i].Melhor)
		}
	}
	t.Logf("melhor: %.1f -> %.1f; média: %.1f -> %.1f", h[0].Melhor, h[len(h)-1].Melhor, h[0].Media, h[len(h)-1].Media)
}

func TestTreinoVisualSeedReproduzivel(t *testing.T) {
	a, b := novoVisual(t, 3), novoVisual(t, 3)
	rodarGeracao(t, a)
	rodarGeracao(t, b)
	if a.Historico[0] != b.Historico[0] {
		t.Errorf("mesma seed, resultados diferentes: %+v e %+v", a.Historico[0], b.Historico[0])
	}
}

func TestTreinoVisualSalvarSemMelhorAvisa(t *testing.T) {
	v := novoVisual(t, 1)
	if err := v.Salvar(); !errors.Is(err, ErroSemMelhor) {
		t.Fatalf("Salvar antes da 1ª geração: %v", err)
	}
	if v.Aviso == "" {
		t.Error("sem aviso")
	}
	if _, err := os.Stat(v.CaminhoCerebro()); err == nil {
		t.Error("gravou arquivo sem ter melhor")
	}
}

func TestTreinoVisualSalvarGravaArquivo14(t *testing.T) {
	v := novoVisual(t, 1)
	rodarGeracao(t, v)
	if err := v.Salvar(); err != nil {
		t.Fatal(err)
	}
	if got, quero := v.CaminhoCerebro(), filepath.Join(v.DirCerebros, "oval.json"); got != quero {
		t.Fatalf("caminho %q, quero %q (contrato 1.4)", got, quero)
	}
	r, err := ia.CarregarRede(v.CaminhoCerebro())
	if err != nil {
		t.Fatalf("o arquivo salvo não carrega: %v", err)
	}
	if r.Pista != "oval" || r.Pontuacao != v.MelhorNota {
		t.Errorf("metadados: pista %q nota %v, quero oval e %v", r.Pista, r.Pontuacao, v.MelhorNota)
	}
	if len(r.Camadas) != 3 || r.Camadas[0] != ia.NEntrada || r.Camadas[2] != ia.NSaida {
		t.Errorf("camadas %v", r.Camadas)
	}
	ref, _ := ia.NovoTreino(v.Pista)
	if got := ref.Rodar(r).Pontuacao; got != v.MelhorNota {
		t.Errorf("o cérebro do arquivo reproduz %v, a nota era %v", got, v.MelhorNota)
	}
}

func TestTreinoVisualSalvarPastaImpossivelAvisa(t *testing.T) {
	v := novoVisual(t, 1)
	rodarGeracao(t, v)
	arq := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(arq, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	v.DirCerebros = arq // um arquivo no lugar da pasta
	if err := v.Salvar(); err == nil {
		t.Fatal("salvou dentro de um arquivo")
	}
	if v.Aviso == "" {
		t.Error("sem aviso do erro")
	}
}

func TestTreinoVisualVelocidadeLimites(t *testing.T) {
	v := novoVisual(t, 1)
	for range 20 {
		v.MaisRapido()
	}
	if v.Velocidade() != Velocidades[len(Velocidades)-1] {
		t.Errorf("máxima %d", v.Velocidade())
	}
	for range 20 {
		v.MaisLento()
	}
	if v.Velocidade() != 1 {
		t.Errorf("mínima %d", v.Velocidade())
	}
}

func TestTreinoVisualRecusaPistaInvalidaOuQueNaoCabe(t *testing.T) {
	if _, err := NovoTreinoVisual(filepath.Join(t.TempDir(), "nao-existe.json"), "", 1); err == nil {
		t.Error("aceitou pista que não existe")
	}
	dir := t.TempDir()
	grande := escrever(t, dir, "grande.json", `{"versao":1,"nome":"grande","largura":90,"centro":[[200,300],[1500,300],[1600,400],[1500,500],[200,500],[100,400]]}`)
	if _, err := NovoTreinoVisual(grande, dir, 1); !errors.Is(err, ErrNaoCabe) {
		t.Errorf("pista que não cabe: %v", err)
	}
}

func TestTextoPainelTreino(t *testing.T) {
	v := novoVisual(t, 1)
	l := TextoPainelTreino(v)
	if len(l) < 4 {
		t.Fatalf("painel com %d linhas: %q", len(l), l)
	}
	rodarGeracao(t, v)
	l = TextoPainelTreino(v)
	if want := "geracao 0: melhor"; !contem(l, want) {
		t.Errorf("painel sem %q: %q", want, l)
	}
	if !contem(l, "geracao 1") {
		t.Errorf("painel não mostra a geração 1 em andamento: %q", l)
	}
	v.Pausado = true
	if !contem(TextoPainelTreino(v), "PAUSADO") {
		t.Error("painel não mostra pausa")
	}
}

func contem(linhas []string, sub string) bool {
	for _, l := range linhas {
		for i := 0; i+len(sub) <= len(l); i++ {
			if l[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

func TestBotaoSalvarDentroDaTela(t *testing.T) {
	b := BotaoSalvar
	if b.Min.X < 0 || b.Min.Y < 0 || b.Max.X > LarguraTela || b.Max.Y > AlturaTela || b.Empty() {
		t.Errorf("botão fora da tela: %v", b)
	}
}

func TestAppIniciarTreinoEVoltar(t *testing.T) {
	dir := t.TempDir()
	m := NovoMenu([]string{pistaOval}, "")
	m.DirCerebros = dir
	a := NovoAppSemJanela(m)
	if err := a.IniciarTreino(); err != nil {
		t.Fatal(err)
	}
	if a.Tela != TelaTreino || a.Treino == nil || a.Treino.V.DirCerebros != dir {
		t.Fatalf("tela %v, treino %v", a.Tela, a.Treino)
	}
	a.Voltar()
	if a.Tela != TelaMenu || a.Treino != nil {
		t.Errorf("depois de voltar: tela %v, treino %v", a.Tela, a.Treino)
	}
}

func TestAppIniciarTreinoRecusadoFicaNoMenuComAviso(t *testing.T) {
	a := NovoAppSemJanela(NovoMenu(nil, ""))
	if err := a.IniciarTreino(); err == nil {
		t.Fatal("abriu treino sem pista")
	}
	if a.Tela != TelaMenu || a.Menu.Aviso == "" {
		t.Errorf("tela %v, aviso %q", a.Tela, a.Menu.Aviso)
	}
}

// Custo de um tick da população inteira (50 × Ler + rede + Avancar), a
// comparar com os 16,6 ms de um quadro a 60 fps (R14).
func BenchmarkTreinoVisualPasso(b *testing.B) {
	v, err := NovoTreinoVisual(pistaOval, b.TempDir(), 1)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		v.Passo()
	}
}
