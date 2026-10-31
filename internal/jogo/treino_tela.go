package jogo

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

var (
	corRaio       = color.RGBA{R: 0x7f, G: 0xc8, B: 0xff, A: 0x38}
	corRaioLider  = color.RGBA{R: 0xff, G: 0xd5, B: 0x4a, A: 0xe0}
	corBotao      = color.RGBA{R: 0x2e, G: 0x7d, B: 0x32, A: 0xff}
	corBotaoSobre = color.RGBA{R: 0x43, G: 0xa0, B: 0x47, A: 0xff}
	corBotaoOff   = color.RGBA{R: 0x55, G: 0x5a, B: 0x62, A: 0xff}
	corMelhor     = color.RGBA{R: 0xff, G: 0xd5, B: 0x4a, A: 0xff} // linha "melhor" do gráfico
	corMedia      = color.RGBA{R: 0x7f, G: 0xc8, B: 0xff, A: 0xff} // linha "média" do gráfico
)

// Retângulos fixos da tela de treino (px de tela).
var (
	// BotaoSalvar é o botão "salvar melhor cérebro", no canto de baixo à direita.
	BotaoSalvar = image.Rect(LarguraTela-290, AlturaTela-52, LarguraTela-10, AlturaTela-12)
	// areaGrafico é o painel do gráfico melhor/média por geração, no canto de cima à direita.
	areaGrafico = image.Rect(LarguraTela-350, 8, LarguraTela-10, 188)
)

// OrcamentoQuadro é o tempo máximo que um Update gasta simulando, para a tela
// seguir a 60 fps mesmo em velocidade alta: se estoura, simula menos ticks
// naquele quadro (o resultado do treino não muda, só a velocidade).
const OrcamentoQuadro = 9 * time.Millisecond

// TreinoTela é a tela do treino (ebiten.Game): a população andando na pista com
// os raios dos sensores, geração, melhor e média da pontuação e o botão de
// salvar o melhor cérebro. A lógica está em TreinoVisual; aqui só há entrada e
// desenho.
//
// Teclas: Espaço pausa, +/- (ou cima/baixo) mudam a velocidade, S ou clique no
// botão salvam, Esc sai.
type TreinoTela struct {
	V *TreinoVisual

	imgCarro *ebiten.Image // sprite do carro (nil nos testes, que não desenham)
	imgPista *ebiten.Image // pista desenhada uma vez
}

// NovoTreinoTela abre a tela de treino na pista de caminhoPista; os cérebros
// vão para dirCerebros ("" = DirCerebros).
func NovoTreinoTela(caminhoPista, dirCerebros string, seed uint64) (*TreinoTela, error) {
	v, err := NovoTreinoVisual(caminhoPista, dirCerebros, seed)
	if err != nil {
		return nil, err
	}
	img, err := decodificarCarro()
	if err != nil {
		return nil, err
	}
	return &TreinoTela{V: v, imgCarro: ebiten.NewImageFromImage(img)}, nil
}

// Atualizar lê a entrada e simula até Velocidade() ticks (dentro do
// orçamento); devolve true quando o jogador pede para sair.
func (t *TreinoTela) Atualizar() (sair bool) {
	v := t.V
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return true
	}
	x, y := ebiten.CursorPosition()
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		v.Pausado = !v.Pausado
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd):
		v.MaisRapido()
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract):
		v.MaisLento()
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		_ = v.Salvar() // o resultado vai para Aviso
	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && image.Pt(x, y).In(BotaoSalvar):
		_ = v.Salvar()
	}
	if v.Pausado {
		return false
	}
	ini := time.Now()
	for range v.Velocidade() {
		v.Passo()
		if time.Since(ini) > OrcamentoQuadro {
			break
		}
	}
	return false
}

// Update roda a 60 TPS; Esc encerra a janela.
func (t *TreinoTela) Update() error {
	if t.Atualizar() {
		return ebiten.Termination
	}
	return nil
}

// Layout fixa a resolução lógica em 1280x720.
func (t *TreinoTela) Layout(_, _ int) (int, int) { return LarguraTela, AlturaTela }

// Draw desenha o quadro.
func (t *TreinoTela) Draw(tela *ebiten.Image) {
	v := t.V
	if t.imgPista == nil {
		t.imgPista = ebiten.NewImage(LarguraTela, AlturaTela)
		desenharPista(t.imgPista, v.Pista)
	}
	tela.DrawImage(t.imgPista, nil)

	lider := v.Lider()
	// Primeiro os que já pararam (esmaecidos), depois os que dirigem, o líder por cima.
	for _, parados := range [2]bool{true, false} {
		for i := range v.Individuos {
			if v.Individuos[i].Fim == parados && i != lider {
				t.desenharIndividuo(tela, i, false)
			}
		}
	}
	if lider >= 0 {
		t.desenharIndividuo(tela, lider, true)
	}
	t.desenharPainel(tela)
	t.desenharGrafico(tela)
	t.desenharBotao(tela)
}

// desenharIndividuo desenha os 7 raios dos sensores (direções fixas do mundo,
// D20, até onde batem na parede) e o carro girado pelo ângulo da resultante.
// Quem já parou fica sem raios e esmaecido.
func (t *TreinoTela) desenharIndividuo(tela *ebiten.Image, i int, lider bool) {
	v := t.V
	ind := &v.Individuos[i]
	c := &ind.Carro
	if !ind.Fim {
		cor, esp := corRaio, float32(1)
		if lider {
			cor, esp = corRaioLider, 2
		}
		for _, r := range sim.RaiosSensores(c, v.Treino.Pista, v.Treino.Cfg) {
			ax, ay := MundoParaTela(r.A.X, r.A.Y)
			bx, by := MundoParaTela(r.B.X, r.B.Y)
			vector.StrokeLine(tela, float32(ax), float32(ay), float32(bx), float32(by), esp, cor, true)
			if lider {
				vector.FillCircle(tela, float32(bx), float32(by), 2.5, cor, true)
			}
		}
	}
	if t.imgCarro == nil {
		return
	}
	b := t.imgCarro.Bounds()
	escala := ComprimentoCarro(v.Pista) / float64(b.Dx())
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(escala, escala)
	op.GeoM.Rotate(AnguloTela(c.Angulo()))
	tx, ty := MundoParaTela(c.Pos.X, c.Pos.Y)
	op.GeoM.Translate(tx, ty)
	alfa := float32(0.7)
	switch {
	case lider:
		alfa = 1
	case ind.Fim:
		alfa = 0.2
	}
	op.ColorScale.ScaleAlpha(alfa)
	tela.DrawImage(t.imgCarro, &op)
}

// TextoPainelTreino devolve as linhas do painel do treino: pista, geração,
// rodada (largada), carros ainda andando, pontuação da última geração (melhor
// e média) e do melhor cérebro de todas, e as teclas.
func TextoPainelTreino(v *TreinoVisual) []string {
	estado := fmt.Sprintf("%dx", v.Velocidade())
	if v.Pausado {
		estado = "PAUSADO"
	}
	linhas := []string{
		fmt.Sprintf("TREINO  pista %s  geracao %d  largada %d/%d  andando %d/%d  vel %s",
			v.Nome, v.Geracao(), v.Largada+1, ia.NLargadas, v.Rodando(), len(v.Individuos), estado),
	}
	if n := len(v.Historico); n > 0 {
		u := v.Historico[n-1]
		linhas = append(linhas, fmt.Sprintf("geracao %d: melhor %.1f  media %.1f", u.Geracao, u.Melhor, u.Media))
		linhas = append(linhas, fmt.Sprintf("melhor de todas: %.1f (nasceu na geracao %d)", v.MelhorNota, v.MelhorGeracao))
	} else {
		linhas = append(linhas, "geracao 0 ainda nao terminou (6 largadas)")
	}
	melhor, media := v.DistanciaAoVivo()
	linhas = append(linhas, fmt.Sprintf("ao vivo: distancia melhor %.0f px  media %.0f px", melhor, media))
	linhas = append(linhas, "Espaco: pausa   +/-: velocidade   S: salvar melhor   Esc: sair")
	if v.Erro != nil {
		linhas = append(linhas, "ERRO: "+v.Erro.Error())
	}
	return linhas
}

func (t *TreinoTela) desenharPainel(tela *ebiten.Image) {
	linhas := TextoPainelTreino(t.V)
	vector.FillRect(tela, 4, 4, 560, float32(16*len(linhas)+8), corPainel, false)
	for i, l := range linhas {
		texto(tela, l, 8, 6+16*i)
	}
	if t.V.Aviso != "" {
		texto(tela, t.V.Aviso, 8, AlturaTela-20)
	}
}

// desenharGrafico traça melhor e média da pontuação por geração.
func (t *TreinoTela) desenharGrafico(tela *ebiten.Image) {
	a := areaGrafico
	vector.FillRect(tela, float32(a.Min.X), float32(a.Min.Y), float32(a.Dx()), float32(a.Dy()), corPainel, false)
	h := t.V.Historico
	texto(tela, "pontuacao por geracao", a.Min.X+8, a.Min.Y+4)
	texto(tela, "melhor", a.Min.X+8, a.Min.Y+20)
	vector.FillRect(tela, float32(a.Min.X+56), float32(a.Min.Y+27), 16, 3, corMelhor, false)
	texto(tela, "media", a.Min.X+88, a.Min.Y+20)
	vector.FillRect(tela, float32(a.Min.X+130), float32(a.Min.Y+27), 16, 3, corMedia, false)
	if len(h) == 0 {
		return
	}
	lo, hi := h[0].Media, h[0].Melhor
	for _, p := range h {
		lo, hi = min(lo, p.Media, p.Melhor), max(hi, p.Melhor, p.Media)
	}
	if hi-lo < 1 {
		hi = lo + 1
	}
	x0, x1 := float64(a.Min.X+56), float64(a.Max.X-12)
	y0, y1 := float64(a.Max.Y-24), float64(a.Min.Y+44) // y0 embaixo (lo), y1 em cima (hi)
	px := func(i int) float64 {
		if len(h) == 1 {
			return x1
		}
		return x0 + (x1-x0)*float64(i)/float64(len(h)-1)
	}
	py := func(v float64) float64 { return y0 + (y1-y0)*(v-lo)/(hi-lo) }
	texto(tela, fmt.Sprintf("%.0f", hi), a.Min.X+4, int(y1)-6)
	texto(tela, fmt.Sprintf("%.0f", lo), a.Min.X+4, int(y0)-6)
	texto(tela, fmt.Sprintf("geracao %d a %d", h[0].Geracao, h[len(h)-1].Geracao), a.Min.X+56, a.Max.Y-18)
	for k, cor := range [2]color.Color{corMedia, corMelhor} {
		val := func(p PontoGeracao) float64 {
			if k == 0 {
				return p.Media
			}
			return p.Melhor
		}
		for i := range h {
			if i > 0 {
				vector.StrokeLine(tela, float32(px(i-1)), float32(py(val(h[i-1]))), float32(px(i)), float32(py(val(h[i]))), 2, cor, true)
			}
		}
		u := len(h) - 1
		vector.FillCircle(tela, float32(px(u)), float32(py(val(h[u]))), 3, cor, true)
	}
}

// desenharBotao desenha o botão de salvar o melhor cérebro.
func (t *TreinoTela) desenharBotao(tela *ebiten.Image) {
	b := BotaoSalvar
	x, y := ebiten.CursorPosition()
	cor := corBotao
	switch {
	case t.V.Melhor == nil:
		cor = corBotaoOff
	case image.Pt(x, y).In(b):
		cor = corBotaoSobre
	}
	vector.FillRect(tela, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), cor, false)
	vector.StrokeRect(tela, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), 2, corParede, false)
	texto(tela, "Salvar melhor cerebro (S)", b.Min.X+14, b.Min.Y+4)
	destino := "-> " + filepath.ToSlash(t.V.CaminhoCerebro())
	if cabem := (b.Dx() - 20) / 6; len(destino) > cabem { // a fonte tem 6 px por letra
		destino = "-> ..." + destino[len(destino)-(cabem-6):]
	}
	texto(tela, destino, b.Min.X+14, b.Min.Y+20)
}
