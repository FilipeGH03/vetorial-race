package jogo

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/sim"
)

// Tela é a tela do App em que o jogador está.
type Tela int8

const (
	TelaMenu    Tela = iota // escolher pista, modo, voltas e carros
	TelaPartida             // corrida em andamento
	TelaFim                 // vencedor e classificação
)

var (
	corLaser  = color.RGBA{R: 0xff, G: 0x1e, B: 0x1e, A: 0xc8}
	corAviso  = color.RGBA{R: 0xff, G: 0xd5, B: 0x4a, A: 0xff}
	corVoce   = color.RGBA{R: 0x7f, G: 0xc8, B: 0xff, A: 0xff}
	corPainel = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x90}
)

// App é o ebiten.Game completo: menu, partida (humano + IA) e tela do vencedor.
// A lógica de cada tela está em Menu e Partida (testáveis sem janela); aqui só
// há entrada e desenho.
type App struct {
	Tela    Tela
	Menu    *Menu
	Partida *Partida
	Opcoes  Opcoes // da última partida, para jogar de novo

	imgCarro *ebiten.Image // sprite do carro (nil nos testes, que não desenham)
	imgPista *ebiten.Image // pista da partida desenhada uma vez
	imgTexto *ebiten.Image // rascunho para texto ampliado
}

// NovoApp abre o menu sobre as pistas de dir. preferida (pode ser "") é a pista
// já escolhida.
func NovoApp(dir, preferida string) (*App, error) {
	img, err := decodificarCarro()
	if err != nil {
		return nil, err
	}
	a := NovoAppSemJanela(NovoMenu(ListarPistas(dir), preferida))
	a.imgCarro = ebiten.NewImageFromImage(img)
	return a, nil
}

// NovoAppSemJanela monta o App sem sprite (para os testes).
func NovoAppSemJanela(m *Menu) *App { return &App{Tela: TelaMenu, Menu: m} }

// Iniciar monta a partida com as escolhas do menu e passa para a corrida. Se a
// pista é recusada (inválida, não cabe em 1280x720), fica no menu com o aviso.
func (a *App) Iniciar() error {
	return a.iniciarCom(a.Menu.Opcoes())
}

func (a *App) iniciarCom(o Opcoes) error {
	p, err := NovaPartida(o)
	if err != nil {
		a.Tela = TelaMenu
		a.Menu.Aviso = err.Error()
		return err
	}
	a.Opcoes, a.Partida, a.imgPista = o, p, nil
	a.Tela = TelaPartida
	return nil
}

// Voltar volta ao menu.
func (a *App) Voltar() { a.Tela, a.Partida, a.imgPista = TelaMenu, nil, nil }

// PassoPartida roda um tick da partida com as teclas dadas e passa para a tela
// do vencedor quando a corrida termina.
func (a *App) PassoPartida(t Teclas) {
	if a.Partida == nil {
		return
	}
	a.Partida.Passo(t)
	if a.Partida.Corrida.Terminada {
		a.Tela = TelaFim
	}
}

// Update roda a 60 TPS.
func (a *App) Update() error {
	switch a.Tela {
	case TelaMenu:
		m := a.Menu
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
			return ebiten.Termination
		case teclaApertada(ebiten.KeyArrowUp, ebiten.KeyW):
			m.Cima()
		case teclaApertada(ebiten.KeyArrowDown, ebiten.KeyS):
			m.Baixo()
		case teclaApertada(ebiten.KeyArrowLeft, ebiten.KeyA):
			m.Esquerda()
		case teclaApertada(ebiten.KeyArrowRight, ebiten.KeyD):
			m.Direita()
		case teclaApertada(ebiten.KeyEnter, ebiten.KeySpace):
			_ = a.Iniciar() // o erro vai para Menu.Aviso
		}
	case TelaPartida:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.Voltar()
			return nil
		}
		a.PassoPartida(TeclasDe(ebiten.IsKeyPressed))
	case TelaFim:
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyR):
			_ = a.iniciarCom(a.Opcoes)
		case teclaApertada(ebiten.KeyEnter, ebiten.KeyEscape):
			a.Voltar()
		}
	}
	return nil
}

func teclaApertada(teclas ...ebiten.Key) bool {
	for _, k := range teclas {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// Layout fixa a resolução lógica em 1280x720.
func (a *App) Layout(_, _ int) (int, int) { return LarguraTela, AlturaTela }

// Draw desenha a tela atual.
func (a *App) Draw(tela *ebiten.Image) {
	switch a.Tela {
	case TelaMenu:
		a.desenharMenu(tela)
	case TelaPartida:
		a.desenharPartida(tela)
	case TelaFim:
		a.desenharPartida(tela)
		a.desenharFim(tela)
	}
}

func (a *App) desenharMenu(tela *ebiten.Image) {
	tela.Fill(corFundo)
	a.textoGrande(tela, "CORRIDA VETORIAL", 80, 70, 4, corLargada)
	m := a.Menu
	for i, linha := range m.Linhas() {
		y := 200 + 60*i
		cor := color.Color(color.White)
		if Campo(i) == m.Selecao {
			cor = corLargada
			a.textoGrande(tela, ">", 70, y, 3, cor)
		}
		a.textoGrande(tela, linha, 110, y, 3, cor)
	}
	ajuda := "Cima/Baixo: escolher linha   Esquerda/Direita: mudar   Enter: correr   Esc: sair"
	texto(tela, ajuda, 80, 480)
	texto(tela, "Dentro da corrida: WASD/setas dirigem (x e y), Esc volta ao menu", 80, 500)
	if m.Aviso != "" {
		a.textoGrande(tela, m.Aviso, 80, 560, 1.5, corErro)
	}
}

func (a *App) desenharPartida(tela *ebiten.Image) {
	p := a.Partida
	if p == nil {
		return
	}
	r := p.Corrida
	if a.imgPista == nil {
		a.imgPista = ebiten.NewImage(LarguraTela, AlturaTela)
		desenharPista(a.imgPista, r.Pista)
	}
	tela.DrawImage(a.imgPista, nil)

	// Laser da Eliminação: faixa vermelha de parede a parede (D6).
	if _, ativo := r.Laser(); ativo {
		f := r.LaserFaixa()
		ax, ay := MundoParaTela(f.A.X, f.A.Y)
		bx, by := MundoParaTela(f.B.X, f.B.Y)
		vector.StrokeLine(tela, float32(ax), float32(ay), float32(bx), float32(by), 10, corLaser, true)
	}

	// Primeiro os outros carros, o jogador por cima.
	for i := len(r.Carros) - 1; i >= 0; i-- {
		a.desenharCarro(tela, r, i)
	}
	if r.Humano >= 0 && r.Carros[r.Humano].Vivo {
		desenharHUD(tela, &r.Carros[r.Humano])
	}
	a.desenharPainel(tela, p)
}

// matizCarro é a rotação de cor (radianos) do sprite azul para o carro i; o do
// jogador mantém o azul original.
func matizCarro(i int) float64 { return float64(i) * 0.9 }

func (a *App) desenharCarro(tela *ebiten.Image, r *sim.Corrida, i int) {
	if a.imgCarro == nil {
		return
	}
	c := &r.Carros[i]
	b := a.imgCarro.Bounds()
	escala := ComprimentoCarro(r.Pista) / float64(b.Dx())
	var op colorm.DrawImageOptions
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(escala, escala)
	op.GeoM.Rotate(AnguloTela(c.Angulo()))
	tx, ty := MundoParaTela(c.Pos.X, c.Pos.Y)
	op.GeoM.Translate(tx, ty)
	var cm colorm.ColorM
	if i != r.Humano {
		cm.ChangeHSV(matizCarro(i), 1, 1)
	}
	if !c.Vivo {
		cm.Scale(1, 1, 1, 0.3) // eliminado: esmaecido
	}
	colorm.DrawImage(tela, a.imgCarro, cm, &op)
	rotulo := NomeCarro(i)
	if i == r.Humano {
		rotulo = "VOCE"
	}
	texto(tela, rotulo, int(tx)-len(rotulo)*3, int(ty)-int(ComprimentoCarro(r.Pista)/2)-18)
}

// desenharPainel mostra modo, tempo, posição, avisos e o painel x, y, módulo,
// ângulo do jogador (R13).
func (a *App) desenharPainel(tela *ebiten.Image, p *Partida) {
	r := p.Corrida
	pos := 0
	for k, i := range r.Classificacao() {
		if i == r.Humano {
			pos = k + 1
		}
	}
	cab := fmt.Sprintf("%s  tempo %.1f s  posicao %d/%d", p.Pista, r.Tempo(), pos, len(r.Carros))
	if r.Modo == sim.ModoVoltas {
		volta := min(r.Carros[Humano].Progresso.Voltas+1, r.Voltas)
		cab += fmt.Sprintf("  volta %d/%d", volta, r.Voltas)
	} else {
		vivos := 0
		for _, c := range r.Carros {
			if c.Vivo {
				vivos++
			}
		}
		cab += fmt.Sprintf("  vivos %d", vivos)
	}
	texto(tela, cab, 8, 4)
	y := 24
	if av := p.AvisoLaser(); av != "" {
		a.textoGrande(tela, av, 8, y, 1.5, corAviso)
		y += 28
	}
	if p.Aviso != "" {
		a.textoGrande(tela, p.Aviso, 8, y, 1, corAviso)
	}
	if r.Humano >= 0 {
		vector.FillRect(tela, LarguraTela-170, 4, 166, 72, corPainel, false)
		texto(tela, TextoPainel(r.Carros[r.Humano].Vel), LarguraTela-166, 6)
	}
}

func (a *App) desenharFim(tela *ebiten.Image) {
	p := a.Partida
	vector.FillRect(tela, 240, 120, 800, 480, color.RGBA{A: 0xd0}, false)
	cor := color.Color(color.White)
	if p.Corrida.Vencedor == Humano {
		cor = corVoce
	}
	a.textoGrande(tela, p.TextoVencedor(), 280, 150, 4, cor)
	for i, linha := range p.Resultado() {
		a.textoGrande(tela, linha, 280, 250+34*i, 2, color.White)
	}
	texto(tela, "R: jogar de novo   Enter: menu", 280, 560)
}

// texto escreve s (sem acentos: a fonte de depuração só tem ASCII) em (x, y).
func texto(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, semAcento(s), x, y)
}

// textoGrande escreve s ampliado por escala, na cor dada, usando um rascunho.
func (a *App) textoGrande(dst *ebiten.Image, s string, x, y int, escala float64, cor color.Color) {
	s = semAcento(s)
	if a.imgTexto == nil {
		a.imgTexto = ebiten.NewImage(LarguraTela, 16)
	}
	a.imgTexto.Clear()
	ebitenutil.DebugPrintAt(a.imgTexto, s, 0, 0)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(escala, escala)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(cor)
	dst.DrawImage(a.imgTexto, &op)
}

var acentos = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "ê", "e", "è", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o",
	"ú", "u", "ü", "u", "ç", "c",
	"Á", "A", "Â", "A", "Ã", "A", "É", "E", "Ê", "E", "Í", "I", "Ó", "O", "Ô", "O", "Õ", "O", "Ú", "U", "Ç", "C",
)

func semAcento(s string) string { return acentos.Replace(s) }
