package jogo

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/sim"
)

var (
	corFundo   = color.RGBA{R: 0x2b, G: 0x2f, B: 0x36, A: 0xff}
	corAsfalto = color.RGBA{R: 0x4a, G: 0x4f, B: 0x58, A: 0xff}
	corParede  = color.RGBA{R: 0xf2, G: 0xf2, B: 0xf2, A: 0xff}
	corLargada = color.RGBA{R: 0xff, G: 0xd5, B: 0x4a, A: 0xff}
)

// FracaoCarro é o comprimento do carro na tela em relação à largura da pista (D14).
const FracaoCarro = 0.40

// ComprimentoCarro é o comprimento do carro desenhado, em px de tela: 40% da
// largura da pista (D14).
func ComprimentoCarro(p *sim.Pista) float64 { return FracaoCarro * p.Largura }

// desenharPista desenha o fundo, o asfalto, as paredes e a linha de largada em dst.
// As coordenadas da pista são de mundo (y para cima); aqui viram coordenadas de tela.
func desenharPista(dst *ebiten.Image, p *sim.Pista) {
	dst.Fill(corFundo)

	// Asfalto: o anel entre as duas paredes (preenchimento par-ímpar).
	var anel vector.Path
	for _, parede := range [2][]sim.Vec2{p.ParedeEsq, p.ParedeDir} {
		for i, q := range parede {
			x, y := MundoParaTela(q.X, q.Y)
			if i == 0 {
				anel.MoveTo(float32(x), float32(y))
			} else {
				anel.LineTo(float32(x), float32(y))
			}
		}
		anel.Close()
	}
	var cor ebiten.ColorScale
	cor.ScaleWithColor(corAsfalto)
	vector.FillPath(dst, &anel, &vector.FillOptions{FillRule: vector.FillRuleEvenOdd},
		&vector.DrawPathOptions{AntiAlias: true, ColorScale: cor})

	// Paredes.
	for _, parede := range [2][]sim.Vec2{p.ParedeEsq, p.ParedeDir} {
		n := len(parede)
		for i := range parede {
			a, b := parede[i], parede[(i+1)%n]
			ax, ay := MundoParaTela(a.X, a.Y)
			bx, by := MundoParaTela(b.X, b.Y)
			vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 3, corParede, true)
		}
	}

	// Linha de largada/chegada: o checkpoint 0.
	if len(p.Checkpoints) > 0 {
		c := p.Checkpoints[0]
		ax, ay := MundoParaTela(c.A.X, c.A.Y)
		bx, by := MundoParaTela(c.B.X, c.B.Y)
		vector.StrokeLine(dst, float32(ax), float32(ay), float32(bx), float32(by), 4, corLargada, true)
	}
}
