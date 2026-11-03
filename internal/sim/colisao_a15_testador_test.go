package sim

import (
	"math"
	"math/rand"
	"testing"
)

// Vários ticks seguidos acelerando contra a parede (D28): o carro desliza, nunca
// atravessa, nunca fica NaN e nunca passa de VMax.
func TestDeslizarVariosTicksNuncaSaiDaPista(t *testing.T) {
	cfg := ConfigPadrao()
	for _, seed := range []int64{1, 2} {
		rng := rand.New(rand.NewSource(seed))
		for _, p := range pistasDeForma(t) {
			for k := 0; k < 100; k++ {
				c := carroEm(pontoNaPista(rng, p), 0)
				ctl := Controle{Acoes: []Acao{
					{Ativa: true, Eixo: EixoX, Sinal: int8(1 - 2*rng.Intn(2))},
					{Ativa: true, Eixo: EixoY, Sinal: int8(1 - 2*rng.Intn(2))},
				}}
				for i := 0; i < 400; i++ {
					c.Avancar(ctl, p, cfg)
					if math.IsNaN(c.Pos.X) || math.IsNaN(c.Vel.X) || math.IsNaN(c.Vel.Y) {
						t.Fatalf("%s: NaN no tick %d", p.Nome, i)
					}
					if !naPista(p, c.Pos) {
						t.Fatalf("%s: saiu da pista em %v no tick %d", p.Nome, c.Pos, i)
					}
					if c.Vel.Len() > cfg.VMax+1e-9 {
						t.Fatalf("%s: |Vel| = %v > VMax", p.Nome, c.Vel.Len())
					}
				}
			}
		}
	}
}

// Com BaterMata o carro morre ao bater, sem deslizar.
func TestBaterMataContinuaMatando(t *testing.T) {
	p := pistaDeParedes(quadro(-1000, -1000, 1000, 10), nil)
	cfg := cfgSemAtrito()
	cfg.BaterMata = true
	c := carroEm(Vec2{0, 9.9}, 0)
	c.Vel = Vec2{300, 100}
	c.Avancar(Controle{}, p, cfg)
	if !c.Batido || c.Vivo {
		t.Errorf("Batido = %v, Vivo = %v; quer true e false", c.Batido, c.Vivo)
	}
}
