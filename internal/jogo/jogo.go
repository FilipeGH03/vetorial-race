package jogo

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"vetorial-race/assets"
	"vetorial-race/internal/sim"
)

// Jogo implementa ebiten.Game: o humano dirige um carro numa pista carregada.
// A física e a colisão moram em internal/sim; aqui só há entrada e desenho.
type Jogo struct {
	Pista *sim.Pista
	Carro sim.Carro
	Cfg   sim.Config // ConfigPadrao com BaterMata = false (D2, D13)

	imgCarro *ebiten.Image // sprite do carro (nil nos testes, que não desenham)
	imgPista *ebiten.Image // pista desenhada uma vez
}

// Novo carrega a pista de caminhoPista (ex.: pistas/oval.json) e o sprite do
// carro, e põe o carro na largada.
func Novo(caminhoPista string) (*Jogo, error) {
	p, err := sim.CarregarPista(caminhoPista)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(assets.CarroAzulPNG))
	if err != nil {
		return nil, fmt.Errorf("decodificar carro_azul.png: %w", err)
	}
	j := NovoComPista(p)
	j.imgCarro = ebiten.NewImageFromImage(img)
	return j, nil
}

// NovoComPista monta o jogo sem sprite nem janela (usado por Novo e pelos testes).
func NovoComPista(p *sim.Pista) *Jogo {
	cfg := sim.ConfigPadrao()
	cfg.BaterMata = false
	j := &Jogo{Pista: p, Cfg: cfg}
	j.Largada()
	return j
}

// Largada põe o carro parado na largada (início da linha central), apontado na
// direção da pista.
func (j *Jogo) Largada() {
	c := sim.NovoCarro(0, j.Pista.Centro[0])
	c.DefinirAngulo(j.Pista.Centro[1].Sub(j.Pista.Centro[0]).Angle())
	j.Carro = c
}

// Passo roda um tick da simulação com as teclas dadas: Controle (contrato 1.1) e
// depois Avancar, que inclui a colisão com as paredes.
func (j *Jogo) Passo(t Teclas) {
	j.Carro.Avancar(ControleDeTeclas(t), j.Pista, j.Cfg)
}

// Update roda a 60 TPS.
func (j *Jogo) Update() error {
	j.Passo(TeclasDe(ebiten.IsKeyPressed))
	return nil
}

// Draw desenha o quadro.
func (j *Jogo) Draw(tela *ebiten.Image) {
	if j.imgPista == nil {
		j.imgPista = ebiten.NewImage(LarguraTela, AlturaTela)
		desenharPista(j.imgPista, j.Pista)
	}
	tela.DrawImage(j.imgPista, nil)

	// Carro: sprite com a frente para +x, centrado na posição, girado pelo ângulo
	// da resultante (c.Angulo()) e com comprimento de 40% da largura da pista (D14).
	b := j.imgCarro.Bounds()
	escala := ComprimentoCarro(j.Pista) / float64(b.Dx())
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(escala, escala)
	op.GeoM.Rotate(AnguloTela(j.Carro.Angulo()))
	tx, ty := MundoParaTela(j.Carro.Pos.X, j.Carro.Pos.Y)
	op.GeoM.Translate(tx, ty)
	tela.DrawImage(j.imgCarro, op)

	ebitenutil.DebugPrint(tela, fmt.Sprintf("FPS: %.1f  TPS: %.1f\nWASD/setas: dirigir  vel: (%.0f, %.0f)",
		ebiten.ActualFPS(), ebiten.ActualTPS(), j.Carro.Vel.X, j.Carro.Vel.Y))
}

// Layout fixa a resolução lógica em 1280x720.
func (j *Jogo) Layout(_, _ int) (int, int) { return LarguraTela, AlturaTela }
