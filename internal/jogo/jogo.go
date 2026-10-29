package jogo

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"vetorial-race/assets"
)

// Jogo implementa ebiten.Game. Nesta fase mostra só o carro parado.
type Jogo struct {
	carro *ebiten.Image
	// posição do carro em coordenadas de mundo (y para cima) e ângulo anti-horário.
	posX, posY, angulo float64
}

// Novo carrega os assets e devolve o jogo.
func Novo() (*Jogo, error) {
	img, err := png.Decode(bytes.NewReader(assets.CarroAzulPNG))
	if err != nil {
		return nil, fmt.Errorf("decodificar carro_azul.png: %w", err)
	}
	return &Jogo{
		carro: ebiten.NewImageFromImage(img),
		posX:  LarguraTela / 2,
		posY:  AlturaTela / 2,
	}, nil
}

// Update roda a 60 TPS.
func (j *Jogo) Update() error { return nil }

// Draw desenha o quadro.
func (j *Jogo) Draw(tela *ebiten.Image) {
	tela.Fill(color.RGBA{R: 0x2b, G: 0x2f, B: 0x36, A: 0xff})

	b := j.carro.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Rotate(AnguloTela(j.angulo))
	tx, ty := MundoParaTela(j.posX, j.posY)
	op.GeoM.Translate(tx, ty)
	tela.DrawImage(j.carro, op)

	ebitenutil.DebugPrint(tela, fmt.Sprintf("FPS: %.1f  TPS: %.1f", ebiten.ActualFPS(), ebiten.ActualTPS()))
}

// Layout fixa a resolução lógica em 1280x720.
func (j *Jogo) Layout(_, _ int) (int, int) { return LarguraTela, AlturaTela }
