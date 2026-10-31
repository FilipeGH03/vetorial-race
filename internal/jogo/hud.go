package jogo

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/sim"
)

// EscalaSeta é quantos px de tela valem 1 px/s de velocidade (VMax = 400 vira 100 px).
const EscalaSeta = 0.25

var (
	corSetaX   = color.RGBA{R: 0xe5, G: 0x39, B: 0x35, A: 0xff} // vermelha
	corSetaY   = color.RGBA{R: 0x43, G: 0xa0, B: 0x47, A: 0xff} // verde
	corSetaRes = color.RGBA{R: 0xff, G: 0xeb, B: 0x3b, A: 0xff} // amarela
)

// Seta é um segmento em coordenadas de tela (y para baixo).
type Seta struct{ X0, Y0, X1, Y1 float64 }

// SetasHUD converte Vel em três setas na tela, todas saindo do carro (pos, em
// mundo): x (horizontal), y (vertical) e a resultante (a soma). O y é invertido
// (D3): vel.Y > 0 aponta para cima na tela, ou seja, Y1 < Y0.
func SetasHUD(pos, vel sim.Vec2, escala float64) (x, y, res Seta) {
	ox, oy := MundoParaTela(pos.X, pos.Y)
	dx, dy := vel.X*escala, -vel.Y*escala
	x = Seta{ox, oy, ox + dx, oy}
	y = Seta{ox, oy, ox, oy + dy}
	res = Seta{ox, oy, ox + dx, oy + dy}
	return
}

// TextoPainel devolve o texto do painel: x, y, módulo e ângulo (graus, 0 a 360,
// anti-horário a partir de +x) de Vel. Parado, o ângulo é 0.
func TextoPainel(vel sim.Vec2) string {
	mod := vel.Len()
	ang := 0.0
	if mod > 0 {
		ang = math.Atan2(vel.Y, vel.X) * 180 / math.Pi
		if ang < 0 {
			ang += 360
		}
	}
	return fmt.Sprintf("x: %7.1f\ny: %7.1f\n|v|: %6.1f\nang: %6.1f graus", vel.X, vel.Y, mod, ang)
}

func desenharSeta(dst *ebiten.Image, s Seta, cor color.Color, esp float32) {
	dx, dy := s.X1-s.X0, s.Y1-s.Y0
	comp := math.Hypot(dx, dy)
	if comp < 1 {
		return
	}
	vector.StrokeLine(dst, float32(s.X0), float32(s.Y0), float32(s.X1), float32(s.Y1), esp, cor, true)
	// Ponta: dois traços curtos para trás, a ±25 graus.
	ux, uy := dx/comp, dy/comp
	tam := math.Min(10, comp/2)
	for _, a := range [2]float64{0.44, -0.44} {
		c, sn := math.Cos(a), math.Sin(a)
		bx := -(ux*c - uy*sn) * tam
		by := -(ux*sn + uy*c) * tam
		vector.StrokeLine(dst, float32(s.X1), float32(s.Y1), float32(s.X1+bx), float32(s.Y1+by), esp, cor, true)
	}
}

// desenharHUD desenha as três setas no carro e o painel no canto da tela.
func desenharHUD(dst *ebiten.Image, c *sim.Carro) {
	x, y, r := SetasHUD(c.Pos, c.Vel, EscalaSeta)
	desenharSeta(dst, x, corSetaX, 2)
	desenharSeta(dst, y, corSetaY, 2)
	desenharSeta(dst, r, corSetaRes, 3)
}
