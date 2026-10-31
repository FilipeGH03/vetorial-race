package jogo

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/sim"
)

var (
	corTraco    = color.RGBA{R: 0x7f, G: 0xc8, B: 0xff, A: 0xff}
	corLinha    = color.RGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	corErro     = color.RGBA{R: 0xff, G: 0x5a, B: 0x4f, A: 0xff}
	corInicio   = color.RGBA{R: 0x5a, G: 0xe0, B: 0x7a, A: 0xff}
	corGrade    = color.RGBA{R: 0x34, G: 0x39, B: 0x41, A: 0xff}
	corBordaJan = color.RGBA{R: 0x6a, G: 0x70, B: 0x7a, A: 0xff}
)

// EditorTela é o editor de pista com o mouse (ebiten.Game): arraste para
// desenhar a linha central, roda ou setas mudam a largura, S salva, L carrega,
// C limpa. A lógica está em Editor; aqui só há entrada e desenho.
type EditorTela struct {
	E       *Editor
	Caminho string // arquivo em que S salva e L carrega
	Aviso   string // última mensagem (salvo, carregado, erro de arquivo)

	imgPista *ebiten.Image // pista válida desenhada, refeita quando a pista muda
	pistaImg *sim.Pista    // pista a que imgPista corresponde
}

// NovoEditorTela abre o editor para o arquivo caminho: se ele já existe, carrega
// a pista para continuar editando; senão começa vazio. O nome da pista é o do
// arquivo, sem a extensão.
func NovoEditorTela(caminho string) (*EditorTela, error) {
	nome := strings.TrimSuffix(filepath.Base(caminho), filepath.Ext(caminho))
	t := &EditorTela{E: NovoEditor(nome), Caminho: caminho}
	if _, err := os.Stat(caminho); err == nil {
		if err := t.E.Carregar(caminho); err != nil {
			return nil, err
		}
		t.Aviso = "carregado: " + caminho
	}
	return t, nil
}

// Update lê o mouse e as teclas, 60 vezes por segundo.
func (t *EditorTela) Update() error {
	e := t.E
	x, y := ebiten.CursorPosition()
	switch {
	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft):
		e.Comecar(float64(x), float64(y))
		t.Aviso = ""
	case e.Arrastando && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft):
		e.Arrastar(float64(x), float64(y))
	case e.Arrastando:
		e.Soltar()
	}

	if !e.Arrastando {
		if _, dy := ebiten.Wheel(); dy > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			e.AjustarLargura(PassoLargura)
		} else if dy < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			e.AjustarLargura(-PassoLargura)
		}
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyC) || inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			e.Limpar()
			t.Aviso = ""
		case inpututil.IsKeyJustPressed(ebiten.KeyS):
			if err := e.Salvar(t.Caminho); err != nil {
				t.Aviso = "não salvou: " + err.Error()
			} else {
				t.Aviso = "salvo em " + t.Caminho
			}
		case inpututil.IsKeyJustPressed(ebiten.KeyL):
			if err := e.Carregar(t.Caminho); err != nil {
				t.Aviso = "não carregou: " + err.Error()
			} else {
				t.Aviso = "carregado: " + t.Caminho
			}
		}
	}
	return nil
}

// Draw desenha o quadro: a pista pronta, ou o traço e o erro.
func (t *EditorTela) Draw(tela *ebiten.Image) {
	e := t.E
	tela.Fill(corFundo)
	for g := 80; g < LarguraTela; g += 80 {
		vector.StrokeLine(tela, float32(g), 0, float32(g), AlturaTela, 1, corGrade, false)
	}
	for g := 80; g < AlturaTela; g += 80 {
		vector.StrokeLine(tela, 0, float32(g), LarguraTela, float32(g), 1, corGrade, false)
	}
	vector.StrokeRect(tela, 1, 1, LarguraTela-2, AlturaTela-2, 2, corBordaJan, false)

	switch {
	case e.Pista != nil:
		if t.imgPista == nil {
			t.imgPista = ebiten.NewImage(LarguraTela, AlturaTela)
		}
		if t.pistaImg != e.Pista {
			desenharPista(t.imgPista, e.Pista)
			t.pistaImg = e.Pista
		}
		tela.DrawImage(t.imgPista, nil)
		desenharSentido(tela, e.Pista)
	case e.Arrastando:
		for i := 1; i < len(e.Traco); i++ {
			a, b := e.Traco[i-1], e.Traco[i]
			vector.StrokeLine(tela, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 3, corTraco, true)
		}
	default:
		desenharLinhaCentral(tela, e.Pontos)
		if e.TemCruzamento {
			x, y := MundoParaTela(e.Cruzamento.X, e.Cruzamento.Y)
			vector.StrokeCircle(tela, float32(x), float32(y), 14, 3, corErro, true)
			vector.FillCircle(tela, float32(x), float32(y), 4, corErro, true)
		}
	}
	if len(e.Traco) > 0 {
		vector.FillCircle(tela, float32(e.Traco[0].X), float32(e.Traco[0].Y), 6, corInicio, true)
	}

	ebitenutil.DebugPrint(tela, fmt.Sprintf(
		"EDITOR DE PISTA  %s  largura: %.0f px\n"+
			"arraste: desenhar (o circuito fecha sozinho)  roda/cima/baixo: largura\n"+
			"S: salvar em %s   L: carregar   C ou Esc: limpar",
		t.E.Nome, e.Largura, t.Caminho))
	if e.Pista != nil {
		ebitenutil.DebugPrintAt(tela, fmt.Sprintf("pista valida: %.0f px de volta, %d checkpoints", e.Pista.Comprimento, len(e.Pista.Checkpoints)), 0, AlturaTela-32)
	}
	if e.Erro != nil && !e.Arrastando {
		ebitenutil.DebugPrintAt(tela, "ERRO: "+textoErro(e.Erro), 0, AlturaTela-48)
	}
	if t.Aviso != "" {
		ebitenutil.DebugPrintAt(tela, t.Aviso, 0, AlturaTela-16)
	}
}

// Layout fixa a resolução lógica em 1280x720.
func (t *EditorTela) Layout(_, _ int) (int, int) { return LarguraTela, AlturaTela }

// textoErro é a mensagem de erro da validação com uma dica do que fazer.
func textoErro(err error) string {
	dica := ""
	switch {
	case errors.Is(err, sim.ErrPistaCruza):
		dica = " (a linha se cruza ou fecha curva demais: refaça o traçado ou diminua a largura; o círculo vermelho marca o cruzamento)"
	case errors.Is(err, sim.ErrPistaCurta):
		dica = " (desenhe um circuito maior)"
	}
	return err.Error() + dica
}

// desenharLinhaCentral desenha a linha central pontos (mundo) como circuito
// fechado, para uma pista que não vale.
func desenharLinhaCentral(dst *ebiten.Image, pontos []sim.Vec2) {
	n := len(pontos)
	for i := range pontos {
		a, b := pontos[i], pontos[(i+1)%n]
		ax, ay := MundoParaTela(a.X, a.Y)
		bx, by := MundoParaTela(b.X, b.Y)
		cor := corLinha
		if i == n-1 {
			cor = corErro // o trecho que fecha o circuito
		}
		vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 3, cor, true)
	}
	if n > 0 {
		x, y := MundoParaTela(pontos[0].X, pontos[0].Y)
		vector.FillCircle(dst, float32(x), float32(y), 6, corInicio, true)
	}
}

// desenharSentido marca o início da linha central e a direção em que se anda.
func desenharSentido(dst *ebiten.Image, p *sim.Pista) {
	a, b := p.Centro[0], p.Centro[min(4, len(p.Centro)-1)]
	ax, ay := MundoParaTela(a.X, a.Y)
	bx, by := MundoParaTela(b.X, b.Y)
	vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 3, corInicio, true)
	vector.FillCircle(dst, float32(ax), float32(ay), 6, corInicio, true)
}
