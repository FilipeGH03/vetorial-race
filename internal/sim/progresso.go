package sim

import (
	"math"
	"sort"
)

// AtualizarProgresso põe em c.Progresso o que o carro andou no tick: o caminho
// do tick é o segmento de "de" (a posição do começo do tick) até c.Pos. Quem
// roda o tick chama Avancar e depois AtualizarProgresso com a posição de antes.
//
// Um checkpoint só conta se é o Proximo, se o caminho o cruza no sentido da
// pista (da parede esquerda para a direita, olhando de dentro para fora; ver
// Pista.Checkpoints) e se vem depois dos outros que o mesmo caminho já cruzou.
// Andar de ré, cruzar de volta ou pular checkpoints não conta; um caminho
// longo que cruza vários em ordem conta todos.
//
// O checkpoint 0 é a largada/chegada. O carro nasce com Proximo = 0 atrás da
// linha: a primeira vez que a cruza só abre a volta 1 (Voltas continua 0);
// depois de cruzar o último checkpoint, o Proximo volta a 0 e cruzar a linha
// completa a volta.
//
// Distancia é a posição do carro ao longo da linha central desde a largada,
// somando as voltas: o comprimento de Voltas voltas mais a projeção da posição
// no trecho da linha central que sai do último checkpoint cruzado. Nunca
// diminui, e fica em 0 até a largada ser cruzada. Carro morto, pista nil ou sem
// checkpoints não mudam o progresso.
func (c *Carro) AtualizarProgresso(de Vec2, p *Pista) {
	if !c.Vivo || p == nil {
		return
	}
	n := len(p.Checkpoints)
	if n == 0 || len(p.Centro) != n {
		return
	}
	pr := &c.Progresso
	// Proximo fora de [0, n) (carro criado à mão ou pista trocada): recomeça.
	if pr.Proximo < 0 || pr.Proximo >= n {
		pr.Proximo = 0
	}

	// Cruza, em ordem, os checkpoints que o caminho do tick atravessa.
	tMin := 0.0
	for k := 0; k < n; k++ {
		t, ok := cruzaCheckpoint(de, c.Pos, p.Checkpoints[pr.Proximo])
		if !ok || t < tMin {
			break
		}
		tMin = t
		if pr.Proximo == 0 && pr.Distancia > float64(pr.Voltas)*p.Comprimento+p.Comprimento/2 {
			pr.Voltas++ // chegada: o carro já deu a volta
		}
		pr.Proximo = (pr.Proximo + 1) % n
	}

	if pr.Proximo == 0 && pr.Distancia <= float64(pr.Voltas)*p.Comprimento+p.Comprimento/2 {
		return // ainda não largou: Distancia fica em 0
	}
	// j é o último checkpoint cruzado; a projeção é no trecho j -> j+1 da linha central.
	j := (pr.Proximo - 1 + n) % n
	a, b := p.Centro[j], p.Centro[(j+1)%n]
	s := math.Max(0, math.Min(p.Passo, c.Pos.Sub(a).Dot(b.Sub(a).Norm())))
	d := float64(pr.Voltas)*p.Comprimento + float64(j)*p.Passo + s
	if d > pr.Distancia {
		pr.Distancia = d
	}
}

// cruzaCheckpoint diz se o caminho de "de" até "ate" cruza o checkpoint cp no
// sentido da pista, e em que fração t do caminho (de 0 a 1). O sentido da pista
// é o do lado esquerdo do checkpoint (cp.A, na parede esquerda) para o direito:
// o carro sai do lado de trás (ou em cima da linha) e termina estritamente na
// frente.
func cruzaCheckpoint(de, ate Vec2, cp Segmento) (float64, bool) {
	if orientacao(cp.A, cp.B, de) > 0 || orientacao(cp.A, cp.B, ate) <= 0 {
		return 0, false
	}
	r, s := ate.Sub(de), cp.B.Sub(cp.A)
	den := r.Cross(s)
	if den == 0 {
		return 0, false
	}
	ad := cp.A.Sub(de)
	t := ad.Cross(s) / den
	u := ad.Cross(r) / den
	if t < 0 || t > 1 || u < -folgaPonta || u > 1+folgaPonta {
		return 0, false
	}
	return t, true
}

// Classificar ordena os carros do primeiro ao último e devolve os índices de
// carros nessa ordem: mais Voltas primeiro, depois maior Distancia; empate total
// fica na ordem original (a classificação é estável). Não altera os carros.
func Classificar(carros []Carro) []int {
	ordem := make([]int, len(carros))
	for i := range ordem {
		ordem[i] = i
	}
	sort.SliceStable(ordem, func(a, b int) bool {
		return carros[ordem[a]].Progresso.MaisAdiante(carros[ordem[b]].Progresso)
	})
	return ordem
}

// MaisAdiante diz se p está estritamente à frente de o: mais Voltas ou, com as
// mesmas Voltas, maior Distancia.
func (p Progresso) MaisAdiante(o Progresso) bool {
	if p.Voltas != o.Voltas {
		return p.Voltas > o.Voltas
	}
	return p.Distancia > o.Distancia
}
