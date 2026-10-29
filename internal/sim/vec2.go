// Package sim é a simulação do jogo: física do carro, pista, colisão e laser.
// Não importa Ebitengine nem nada de desenho: roda sem janela (o treino usa).
// Coordenadas matemáticas: y para cima, ângulo anti-horário a partir de +x.
package sim

import "math"

// Vec2 é um vetor (ou ponto) do plano, em pixels de mundo ou px/s.
type Vec2 struct{ X, Y float64 }

// Add devolve v + o.
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Sub devolve v - o.
func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }

// Scale devolve v * k.
func (v Vec2) Scale(k float64) Vec2 { return Vec2{v.X * k, v.Y * k} }

// Dot é o produto escalar v · o.
func (v Vec2) Dot(o Vec2) float64 { return v.X*o.X + v.Y*o.Y }

// Cross é o componente z do produto vetorial v × o (positivo se o está
// à esquerda de v, no sentido anti-horário).
func (v Vec2) Cross(o Vec2) float64 { return v.X*o.Y - v.Y*o.X }

// Len é o módulo |v|.
func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

// Norm devolve o vetor unitário na direção de v; o vetor nulo devolve o nulo.
func (v Vec2) Norm() Vec2 {
	n := v.Len()
	if n == 0 {
		return Vec2{}
	}
	return Vec2{v.X / n, v.Y / n}
}

// Angle é atan2(Y, X), em radianos, de -π a π. O vetor nulo dá 0.
func (v Vec2) Angle() float64 { return math.Atan2(v.Y, v.X) }

// Rotate gira v por a radianos no sentido anti-horário.
func (v Vec2) Rotate(a float64) Vec2 {
	s, c := math.Sincos(a)
	return Vec2{v.X*c - v.Y*s, v.X*s + v.Y*c}
}
