package sim

import (
	"fmt"
	"math"
	"testing"
)

// folgaParedes é a menor distância de q a qualquer trecho das paredes.
func folgaParedes(p *Pista, q Vec2) float64 {
	m := math.Inf(1)
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for i := range w {
			m = math.Min(m, distanciaPontoSegmento(q, w[i], w[(i+1)%len(w)]))
		}
	}
	return m
}

// conferirLargada confere as regras da D16 para os carros de Largada(p, n).
func conferirLargada(t *testing.T, p *Pista, n int, carros []Carro) {
	t.Helper()
	caso := fmt.Sprintf("%s, %d carros", p.Nome, n)
	if len(carros) != n {
		t.Fatalf("%s: %d carros", caso, len(carros))
	}
	cp0 := p.Checkpoints[0]
	vistos := map[Vec2]bool{}
	for i, c := range carros {
		if c.ID != i || !c.Vivo || c.Batido || c.Vel != (Vec2{}) || c.Progresso != (Progresso{}) {
			t.Errorf("%s: carro %d = %+v, quer ID %d, vivo, parado e sem progresso", caso, i, c, i)
		}
		if vistos[c.Pos] {
			t.Errorf("%s: dois carros em %v", caso, c.Pos)
		}
		vistos[c.Pos] = true
		if !naPista(p, c.Pos) {
			t.Errorf("%s: carro %d fora da pista em %v", caso, i, c.Pos)
		}
		if f := folgaParedes(p, c.Pos); f < FolgaLargada {
			t.Errorf("%s: carro %d a %.2f px da parede, quer ≥ %v", caso, i, f, FolgaLargada)
		}
		// Atrás da linha de largada (a 1ª fila está perto dela).
		if d := distanciaPontoSegmento(c.Pos, cp0.A, cp0.B); d < p.Largura && orientacao(cp0.A, cp0.B, c.Pos) >= 0 {
			t.Errorf("%s: carro %d em %v está à frente da largada", caso, i, c.Pos)
		}
		// Apontado na direção da pista: no máximo 50° da direção de algum trecho
		// da linha central a até meia largura do carro (numa quina são vários).
		m := len(p.Centro)
		dir := Vec2{math.Cos(c.Angulo()), math.Sin(c.Angulo())}
		alinhado := false
		for k := range p.Centro {
			a, b := p.Centro[k], p.Centro[(k+1)%m]
			if distanciaPontoSegmento(c.Pos, a, b) <= p.Largura/2 && dir.Dot(b.Sub(a).Norm()) >= math.Cos(50*math.Pi/180) {
				alinhado = true
				break
			}
		}
		if !alinhado {
			t.Errorf("%s: carro %d em %v apontado para %.0f°, fora da direção da pista ali", caso, i, c.Pos, c.Angulo()*180/math.Pi)
		}
	}
}

// emFrente anda com o carro em linha reta na direção do seu ângulo, em ticks de
// passo px (com AtualizarProgresso), até o progresso mudar, a parede ser tocada
// ou andar maximo px. Devolve quanto andou e se tocou a parede.
func emFrente(c *Carro, p *Pista, passo, maximo float64) (float64, bool) {
	d := Vec2{math.Cos(c.Angulo()), math.Sin(c.Angulo())}
	inicio := c.Progresso
	andou := 0.0
	for andou < maximo && c.Progresso == inicio {
		de := c.Pos
		c.Pos = c.Pos.Add(d.Scale(passo))
		andou += passo
		if cruzaParede(p, de, c.Pos) {
			return andou, true
		}
		c.AtualizarProgresso(de, p)
	}
	return andou, false
}

// Largada(oval, 6): 6 carros entre as paredes com folga ≥ 8 px, atrás da linha,
// e o 1º tick para a frente que muda o progresso é a travessia da largada.
func TestLargadaOval6(t *testing.T) {
	p := carregarOval(t)
	carros := Largada(p, 6)
	conferirLargada(t, p, 6, carros)
	cp0 := p.Checkpoints[0]
	perto := 0
	for i := range carros {
		c := carros[i]
		d := distanciaPontoSegmento(c.Pos, cp0.A, cp0.B)
		if d <= RecuoLargada+5 {
			perto++
		}
		andou, bateu := emFrente(&c, p, passoVMax(), 200)
		if bateu {
			t.Fatalf("carro %d bateu na parede antes da largada (andou %.1f px)", i, andou)
		}
		if c.Progresso.Proximo != 1 || c.Progresso.Voltas != 0 || c.Progresso.Distancia > p.Passo {
			t.Errorf("carro %d: depois de andar %.1f px o progresso é %+v, quer Proximo 1, Voltas 0", i, andou, c.Progresso)
		}
		t.Logf("carro %d em (%.1f, %.1f), %.1f px da largada, %.0f°", i, carros[i].Pos.X, carros[i].Pos.Y, d, carros[i].Angulo()*180/math.Pi)
	}
	if perto == 0 {
		t.Errorf("nenhum carro a ~%v px da largada", RecuoLargada)
	}
}

// passoVMax é o deslocamento de um tick em velocidade máxima (400 px/s · 1/60 s).
func passoVMax() float64 { c := ConfigPadrao(); return c.VMax * c.DT }

// Em todas as formas de pista e de 1 a 8 carros, a largada respeita a D16 e
// cada carro, dirigido só com Avancar, conta a largada como o 1º checkpoint
// (Proximo 1, Voltas 0) logo no começo.
func TestLargadaEmTodasAsPistas(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		for n := 1; n <= 8; n++ {
			carros := Largada(p, n)
			conferirLargada(t, p, n, carros)
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

// Seis carros não cabem lado a lado no oval (0,2·90 = 18 px entre vizinhos em
// 74 px livres): vão em duas filas de 3, a de trás mais longe da largada.
func TestLargadaEmFilas(t *testing.T) {
	p := carregarOval(t)
	carros := Largada(p, 6)
	cp0 := p.Checkpoints[0]
	d := func(i int) float64 { return distanciaPontoSegmento(carros[i].Pos, cp0.A, cp0.B) }
	for i := 0; i < 3; i++ {
		if d(i+3) < d(i)+0.4*p.Largura {
			t.Errorf("carro %d (2ª fila) a %.1f px da largada, carro %d (1ª fila) a %.1f", i+3, d(i+3), i, d(i))
		}
		for j := 0; j < i; j++ {
			if dd := carros[i].Pos.Sub(carros[j].Pos).Len(); dd < 0.2*p.Largura {
				t.Errorf("carros %d e %d da mesma fila a %.1f px", i, j, dd)
			}
		}
	}
	// Quatro cabem lado a lado.
	um := Largada(p, 4)
	for i := 1; i < 4; i++ {
		if math.Abs(distanciaPontoSegmento(um[i].Pos, cp0.A, cp0.B)-distanciaPontoSegmento(um[0].Pos, cp0.A, cp0.B)) > 0.2*p.Largura {
			t.Errorf("4 carros deviam ficar numa fila só: %v e %v", um[0].Pos, um[i].Pos)
		}
	}
}

func TestLargadaBorda(t *testing.T) {
	p := carregarOval(t)
	if c := Largada(nil, 3); c != nil {
		t.Errorf("pista nil: %v", c)
	}
	if c := Largada(&Pista{}, 3); c != nil {
		t.Errorf("pista vazia: %v", c)
	}
	if c := Largada(p, 0); c != nil {
		t.Errorf("0 carros: %v", c)
	}
	if c := Largada(p, -1); c != nil {
		t.Errorf("-1 carros: %v", c)
	}
	// Mesma pista, mesma largada (sem relógio nem sorteio).
	a, b := Largada(p, 6), Largada(p, 6)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("largada não é determinística: %+v e %+v", a[i], b[i])
		}
	}
	// Muitos carros (treino) também ficam na pista e atrás da linha.
	conferirLargada(t, p, 50, Largada(p, 50))
}

// motorista persegue o ponto da linha central 2 pontos à frente do mais perto
// do carro, com velocidade alvo de 150 px/s: o suficiente para dar voltas.
func motorista(p *Pista, c *Carro) Controle {
	n := len(p.Centro)
	melhor, dist := 0, math.Inf(1)
	for i, q := range p.Centro {
		if d := q.Sub(c.Pos).Len(); d < dist {
			melhor, dist = i, d
		}
	}
	erro := p.Centro[(melhor+2)%n].Sub(c.Pos).Norm().Scale(150).Sub(c.Vel)
	var ctl Controle
	for _, e := range []struct {
		eixo Eixo
		v    float64
	}{{EixoX, erro.X}, {EixoY, erro.Y}} {
		if math.Abs(e.v) > 10 {
			s := int8(1)
			if e.v < 0 {
				s = -1
			}
			ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e.eixo, Sinal: s})
		}
	}
	return ctl
}

// Só com Avancar (sem chamar AtualizarProgresso), a partir da largada, a
// Distancia cresce ao longo de uma volta no oval e a volta é contada (D15).
func TestAvancarSozinhoContaProgresso(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	c := Largada(p, 1)[0]
	ult, cresceu, segundos := 0.0, 0, 0
	for tick := 1; tick <= 60*60 && c.Progresso.Voltas < 1; tick++ {
		c.Avancar(motorista(p, &c), p, cfg)
		if c.Progresso.Distancia < ult {
			t.Fatalf("tick %d: Distancia caiu de %.1f para %.1f", tick, ult, c.Progresso.Distancia)
		}
		if tick%60 == 0 {
			segundos++
			if c.Progresso.Distancia > ult {
				cresceu++
			}
			ult = c.Progresso.Distancia
		}
	}
	t.Logf("progresso %+v em %d s (Distancia cresceu em %d)", c.Progresso, segundos, cresceu)
	if c.Progresso.Voltas != 1 {
		t.Fatalf("Avancar não contou a volta em 60 s: %+v", c.Progresso)
	}
	if c.Progresso.Distancia < p.Comprimento-p.Passo || c.Progresso.Distancia > p.Comprimento+2*p.Passo {
		t.Errorf("Distancia ao fechar a volta = %.1f, quer ≈ %.1f", c.Progresso.Distancia, p.Comprimento)
	}
	if cresceu < segundos-1 {
		t.Errorf("Distancia só cresceu em %d de %d segundos", cresceu, segundos)
	}
}

// Os 6 carros da largada, dirigidos só com Avancar, cruzam a largada logo no
// começo (sem contar volta) e todos fecham uma volta.
func TestLargadaSeisCarrosDaoAVolta(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	carros := Largada(p, 6)
	for i := range carros {
		c := &carros[i]
		largou := -1
		for tick := 1; tick <= 60*60 && c.Progresso.Voltas < 1; tick++ {
			c.Avancar(motorista(p, c), p, cfg)
			if largou < 0 && c.Progresso.Proximo != 0 {
				largou = tick
				if c.Progresso.Proximo != 1 || c.Progresso.Voltas != 0 {
					t.Errorf("carro %d: 1º checkpoint contado deu %+v, quer a largada", i, c.Progresso)
				}
			}
		}
		if largou < 0 || largou > 3*60 {
			t.Errorf("carro %d: cruzou a largada no tick %d, quer nos 3 primeiros segundos", i, largou)
		}
		if c.Progresso.Voltas != 1 {
			t.Errorf("carro %d: não fechou a volta: %+v", i, c.Progresso)
		}
	}
}
