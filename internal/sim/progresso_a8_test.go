package sim

import (
	"fmt"
	"math"
	"testing"
)

// percorrer leva o carro pelo circuito fechado q, do ponto i0 até o ponto
// i0+pontos (dando a volta quantas vezes precisar), em ticks de passo px,
// chamando AtualizarProgresso a cada tick.
//
// Com nasLinhas, cada trecho q[i] -> q[i+1] é dividido em ticks iguais e o
// último termina exatamente em q[i+1], que fica sobre o checkpoint i+1. Sem,
// os ticks andam passo px ao longo do caminho e terminam onde der; só um tick
// que cortaria a quina por fora da pista (rente à parede de dentro) termina no
// vértice, como a colisão faria um carro de verdade contornar.
func percorrer(c *Carro, p *Pista, q []Vec2, i0, pontos int, passo float64, nasLinhas bool) {
	n := len(q)
	tick := func(ate Vec2) {
		de := c.Pos
		c.Pos = ate
		c.AtualizarProgresso(de, p)
	}
	k, s := 0, 0.0 // trecho atual (q[i0+k] -> q[i0+k+1]) e quanto dele já foi andado
	for k < pontos {
		a, b := q[(i0+k)%n], q[(i0+k+1)%n]
		l := b.Sub(a).Len()
		if nasLinhas {
			m := int(math.Ceil(l / passo))
			for i := 1; i < m; i++ {
				tick(a.Add(b.Sub(a).Scale(float64(i) / float64(m))))
			}
			tick(b)
			k++
			continue
		}
		// Ponto passo px à frente ao longo do caminho.
		kk, ss, falta := k, s, passo
		for kk < pontos {
			ll := q[(i0+kk+1)%n].Sub(q[(i0+kk)%n]).Len()
			if ss+falta <= ll {
				ss += falta
				break
			}
			falta -= ll - ss
			kk, ss = kk+1, 0
		}
		if kk >= pontos { // o caminho acaba antes de um passo inteiro
			tick(q[(i0+pontos)%n])
			return
		}
		aa, bb := q[(i0+kk)%n], q[(i0+kk+1)%n]
		alvo := aa.Add(bb.Sub(aa).Scale(ss / bb.Sub(aa).Len()))
		if kk > k && cruzaParede(p, c.Pos, alvo) {
			tick(b) // cortaria a quina por fora: termina no vértice
			k, s = k+1, 0
			continue
		}
		tick(alvo)
		k, s = kk, ss
	}
}

// Achado da onda 3: com ticks que terminam em cima de um checkpoint (ou a um
// arredondamento dele), cruzaCheckpoint recusava o tick seguinte e o progresso
// travava para sempre. Em todas as formas de pista, a qualquer distância das
// paredes, com passos de 6,7 (VMax/60), 15 e 30 px, terminando ou não sobre as
// linhas, duas voltas e pouco contam as voltas certas.
func TestVoltasComPassosETicksSobreALinha(t *testing.T) {
	falhas := 0
	for _, p := range pistasDeForma(t) {
		n := len(p.Checkpoints)
		for nome, q := range caminhosDaPista(p) {
			for _, passo := range []float64{6.7, 15, 30} {
				for _, nasLinhas := range []bool{false, true} {
					c := carroEm(q[n-3], 0)
					percorrer(c, p, q, n-3, 3*n+3+5, passo, nasLinhas) // larga, 3 voltas e 5 pontos
					caso := fmt.Sprintf("%s, %s, passo %v, nas linhas %v", p.Nome, nome, passo, nasLinhas)
					if c.Progresso.Voltas != 3 {
						falhas++
						t.Errorf("%s: Voltas = %d, quer 3 (progresso %+v)", caso, c.Progresso.Voltas, c.Progresso)
						continue
					}
					want := 3*p.Comprimento + 5*p.Passo
					if got := c.Progresso.Distancia; math.Abs(got-want) > 0.1*p.Comprimento {
						t.Errorf("%s: Distancia = %.1f, quer ≈ %.1f", caso, got, want)
					}
				}
			}
		}
	}
	if falhas > 0 {
		t.Logf("%d combinações travaram", falhas)
	}
}

// O carro que termina um tick exatamente sobre o checkpoint (ou a um
// arredondamento dele, de qualquer lado) cruza no tick seguinte; cruzar de
// novo, ou para trás, não conta. O t devolvido fica sempre em [0, 1].
func TestCruzaCheckpointSobreALinha(t *testing.T) {
	cp := Segmento{Vec2{100.1, 345.7}, Vec2{133.3, 271.9}} // diagonal, sem números redondos
	frente := cp.B.Sub(cp.A).Norm().Rotate(math.Pi / 2)
	lateral := cp.B.Sub(cp.A).Norm()
	for _, u := range []float64{0, 0.3, 0.5, 1} {
		for _, ruido := range []float64{-1e-12, -1e-9, 0, 1e-12, 1e-9, 5e-7} {
			sobre := cp.A.Add(cp.B.Sub(cp.A).Scale(u)).Add(frente.Scale(ruido))
			if u == 1 || u == 0 {
				sobre = sobre.Sub(lateral.Scale(1e-7 * (2*u - 1))) // ponta: um tiquinho para dentro
			}
			antes := sobre.Sub(frente.Scale(6.7))
			depois := sobre.Add(frente.Scale(6.7))
			_, ok1 := cruzaCheckpoint(antes, sobre, cp)
			_, ok2 := cruzaCheckpoint(sobre, depois, cp)
			if ok1 == ok2 {
				t.Errorf("u %v, ruído %g: chegar na linha %v, sair dela %v; quer exatamente um dos dois", u, ruido, ok1, ok2)
			}
			for _, cam := range [][2]Vec2{{antes, sobre}, {sobre, depois}, {antes, depois}} {
				if tt, ok := cruzaCheckpoint(cam[0], cam[1], cp); ok && (tt < 0 || tt > 1) {
					t.Errorf("u %v, ruído %g: t = %v fora de [0,1]", u, ruido, tt)
				}
				if _, ok := cruzaCheckpoint(cam[1], cam[0], cp); ok {
					t.Errorf("u %v, ruído %g: de ré contou", u, ruido)
				}
			}
		}
	}
	// Fora do checkpoint (além das pontas) não conta.
	for _, u := range []float64{-0.01, 1.01} {
		q := cp.A.Add(cp.B.Sub(cp.A).Scale(u))
		if _, ok := cruzaCheckpoint(q.Sub(frente), q.Add(frente), cp); ok {
			t.Errorf("u %v: cruzou fora do checkpoint", u)
		}
	}
}
