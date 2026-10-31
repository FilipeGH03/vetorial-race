package sim

import (
	"fmt"
	"math"
	"testing"
)

// girar devolve os pontos com o início deslocado k posições: o checkpoint 0
// (largada) cai noutro ponto da mesma pista.
func girar(pts []Vec2, k int) []Vec2 {
	m := len(pts)
	r := make([]Vec2, m)
	for i := range pts {
		r[i] = pts[(i+k)%m]
	}
	return r
}

// deslocamentos devolve as rotações testadas para uma linha central de m
// pontos: 1, 2 e de m/7 em m/7 até m-3.
func deslocamentos(m int) []int {
	ks := []int{1, 2}
	for k := max(3, m/7); k <= m-3; k += max(1, m/7) {
		ks = append(ks, k)
	}
	return append(ks, m-3)
}

// conferirDirecaoLargada confere que cada carro aponta a ≤ 50° da direção do
// trecho da linha central mais perto dele (a pista onde ele está) e não bate
// na parede ao andar 3 larguras de carro (24 px) para a frente.
func conferirDirecaoLargada(t *testing.T, p *Pista, n int, carros []Carro) {
	t.Helper()
	m := len(p.Centro)
	for i, c := range carros {
		dir := Vec2{math.Cos(c.Angulo()), math.Sin(c.Angulo())}
		// Os trechos da linha central mais perto do carro (vários se ele está na
		// quina, onde empatam): ao menos um deve estar a ≤ 50° da direção dele.
		dist := math.Inf(1)
		for k := range p.Centro {
			dist = math.Min(dist, distanciaPontoSegmento(c.Pos, p.Centro[k], p.Centro[(k+1)%m]))
		}
		melhorCos := -1.0
		for k := range p.Centro {
			a, b := p.Centro[k], p.Centro[(k+1)%m]
			if distanciaPontoSegmento(c.Pos, a, b) <= dist+0.01 {
				melhorCos = math.Max(melhorCos, dir.Dot(b.Sub(a).Norm()))
			}
		}
		if melhorCos < math.Cos(50*math.Pi/180) {
			t.Errorf("%s, %d carros: carro %d em %v aponta %.0f°, a %.0f° da pista ali",
				p.Nome, n, i, c.Pos, c.Angulo()*180/math.Pi, math.Acos(math.Max(-1, melhorCos))*180/math.Pi)
		}
		if cruzaParede(p, c.Pos, c.Pos.Add(dir.Scale(24))) {
			t.Errorf("%s, %d carros: carro %d em %v bate na parede a menos de 24 px à frente", p.Nome, n, i, c.Pos)
		}
	}
}

// A largada com a linha deslocada (Centro girado 1, 2, m/7..m-3 pontos) e no
// retângulo 1280x720 de largura 60: todo carro a ≤ 50° da direção da pista,
// ≥ 8 px das paredes, atrás da linha, e a 1ª travessia é o checkpoint 0.
func TestLargadaEmTodasAsPistasDeslocada(t *testing.T) {
	cfg := ConfigPadrao()
	retangulo := []Vec2{{30, 30}, {1250, 30}, {1250, 690}, {30, 690}}
	base := pistasDeForma(t)
	pr, err := NovaPista("retangulo 1280x720", 60, retangulo)
	if err != nil {
		t.Fatalf("NovaPista(retângulo): %v", err)
	}
	base = append(base, pr)
	for _, b := range base {
		m := len(b.Centro)
		for _, k := range append([]int{0}, deslocamentos(m)...) {
			p, err := NovaPista(fmt.Sprintf("%s +%d", b.Nome, k), b.Largura, girar(b.Centro, k))
			if err != nil {
				t.Fatalf("%s +%d: NovaPista: %v", b.Nome, k, err)
			}
			for _, n := range []int{1, 3, 6, 8} {
				carros := Largada(p, n)
				conferirLargada(t, p, n, carros)
				conferirDirecaoLargada(t, p, n, carros)
				for i := range carros {
					c := &carros[i]
					tick := 1
					for ; tick <= 5*60 && c.Progresso.Proximo == 0; tick++ {
						c.Avancar(motorista(p, c), p, cfg)
					}
					if c.Progresso.Proximo != 1 || c.Progresso.Voltas != 0 {
						t.Errorf("%s, %d carros: carro %d em %d ticks ficou com %+v, quer Proximo 1 e Voltas 0",
							p.Nome, n, i, tick, c.Progresso)
					}
				}
			}
		}
	}
}
