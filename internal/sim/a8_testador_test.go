package sim

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// A largada em todas as formas de pista, com várias quantidades de carros
// (inclusive mais do que a grade de treino costuma usar), respeita a D16, e a
// pista não é alterada por Largada.
func TestA8LargadaTodasPistasMuitosCarros(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		antes := fmt.Sprint(p.Centro, p.ParedeEsq, p.ParedeDir, p.Checkpoints, p.Comprimento, p.Passo)
		for _, n := range []int{1, 2, 5, 9, 12, 20, 30} {
			conferirLargada(t, p, n, Largada(p, n))
		}
		if depois := fmt.Sprint(p.Centro, p.ParedeEsq, p.ParedeDir, p.Checkpoints, p.Comprimento, p.Passo); antes != depois {
			t.Errorf("%s: Largada alterou a pista", p.Nome)
		}
	}
}

// Larguras de pista de 40 a 200 px (limites do contrato) numa mesma forma: a
// folga de 8 px das paredes e o resto da D16 valem em todas.
func TestA8LargadaTodasAsLarguras(t *testing.T) {
	forma := []Vec2{{200, 300}, {600, 300}, {700, 400}, {600, 500}, {200, 500}, {100, 400}}
	for _, w := range []float64{40, 41.5, 60, 90, 120, 160, 199, 200} {
		// Pista estreita e larga na mesma forma pode cruzar as paredes: pula as que a A6/B1 recusam.
		p, err := NovaPista(fmt.Sprintf("oval w%v", w), w, forma)
		if err != nil {
			t.Logf("largura %v: NovaPista recusou (%v)", w, err)
			continue
		}
		quantidades := []int{1, 3, 6, 8}
		if w >= 90 {
			quantidades = append(quantidades, 16) // 16 carros numa pista de 40 px não é uso real
		}
		for _, n := range quantidades {
			conferirLargada(t, p, n, Largada(p, n))
		}
	}
}

// Só com Avancar, acelerando para onde o carro aponta (componentes do ângulo
// que ele recebeu da largada), os 6 carros da largada do oval cruzam a largada
// sem bater, o progresso vira Proximo 1 e Voltas 0.
func TestA8LargadaPrimeiroTickParaFrenteCruza(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	for _, c0 := range Largada(p, 6) {
		c := c0
		var ctl Controle
		if cx := math.Cos(c.Angulo()); math.Abs(cx) > 0.3 {
			ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: EixoX, Sinal: int8(math.Copysign(1, cx))})
		}
		if sy := math.Sin(c.Angulo()); math.Abs(sy) > 0.3 {
			ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: EixoY, Sinal: int8(math.Copysign(1, sy))})
		}
		for tick := 0; tick < 2*60 && c.Progresso.Proximo == 0; tick++ {
			c.Avancar(ctl, p, cfg)
			if c.Batido {
				t.Fatalf("carro %d bateu no tick %d em %v", c.ID, tick, c.Pos)
			}
		}
		if c.Progresso.Proximo != 1 || c.Progresso.Voltas != 0 {
			t.Errorf("carro %d: progresso %+v depois de acelerar para frente, quer Proximo 1 e Voltas 0", c.ID, c.Progresso)
		}
	}
}

// Invariantes do progresso quando só Avancar roda, com controles aleatórios (com
// seed) a partir da largada, em todas as formas de pista, com e sem BaterMata:
// Voltas e Distancia nunca diminuem, Proximo fica em [0, n), um tick que bate
// ou um carro morto não mexe no progresso, e Distancia acompanha as Voltas.
func TestA8AvancarInvariantesAleatorios(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		n := len(p.Checkpoints)
		for _, mata := range []bool{false, true} {
			cfg := ConfigPadrao()
			cfg.BaterMata = mata
			rng := rand.New(rand.NewSource(42))
			carros := Largada(p, 6)
			for i := range carros {
				c := &carros[i]
				var ult Progresso
				for tick := 0; tick < 1500; tick++ {
					var ctl Controle
					for eixo := Eixo(0); eixo < 2; eixo++ {
						switch rng.Intn(3) {
						case 0:
							ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: eixo, Sinal: 1})
						case 1:
							ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: eixo, Sinal: -1})
						}
					}
					antes := c.Progresso
					c.Avancar(ctl, p, cfg)
					pr := c.Progresso
					if pr.Proximo < 0 || pr.Proximo >= n {
						t.Fatalf("%s mata=%v carro %d tick %d: Proximo %d fora de [0,%d)", p.Nome, mata, i, tick, pr.Proximo, n)
					}
					if pr.Voltas < ult.Voltas || pr.Distancia < ult.Distancia {
						t.Fatalf("%s mata=%v carro %d tick %d: progresso regrediu de %+v para %+v", p.Nome, mata, i, tick, ult, pr)
					}
					if c.Batido && pr != antes {
						t.Fatalf("%s mata=%v carro %d tick %d: bateu e o progresso mudou de %+v para %+v", p.Nome, mata, i, tick, antes, pr)
					}
					if pr.Voltas > ult.Voltas+1 {
						t.Fatalf("%s carro %d tick %d: pulou de %d para %d voltas num tick", p.Nome, i, tick, ult.Voltas, pr.Voltas)
					}
					if pr.Distancia > 0 && pr.Distancia < float64(pr.Voltas)*p.Comprimento-p.Comprimento/2 {
						t.Fatalf("%s carro %d tick %d: Distancia %.1f incompatível com %d voltas", p.Nome, i, tick, pr.Distancia, pr.Voltas)
					}
					if math.IsNaN(pr.Distancia) || math.IsInf(pr.Distancia, 0) {
						t.Fatalf("%s carro %d tick %d: Distancia %v", p.Nome, i, tick, pr.Distancia)
					}
					ult = pr
				}
				if mata && !c.Vivo {
					morto := c.Progresso
					for k := 0; k < 10; k++ {
						c.Avancar(Controle{Acoes: []Acao{{Ativa: true, Eixo: EixoX, Sinal: 1}}}, p, cfg)
					}
					if c.Progresso != morto {
						t.Errorf("%s carro %d: carro morto mudou o progresso %+v -> %+v", p.Nome, i, morto, c.Progresso)
					}
				}
			}
		}
	}
}

// Dirigindo de verdade (só Avancar) desde a largada, cada carro das pistas de
// forma sem quina fechada demais completa pelo menos uma volta e a contagem não
// trava; nas demais o teste só confere que a Distancia nunca regride.
func TestA8AvancarDaVoltaEmPistasDeForma(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		c := Largada(p, 1)[0]
		ult := 0.0
		for tick := 0; tick < 90*60 && c.Progresso.Voltas < 1; tick++ {
			c.Avancar(motorista(p, &c), p, cfg)
			if c.Progresso.Distancia < ult {
				t.Fatalf("%s tick %d: Distancia caiu de %.1f para %.1f", p.Nome, tick, ult, c.Progresso.Distancia)
			}
			ult = c.Progresso.Distancia
		}
		t.Logf("%s: progresso %+v (comprimento %.0f)", p.Nome, c.Progresso, p.Comprimento)
	}
}
