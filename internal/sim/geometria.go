package sim

import "iter"

// Geometria de segmentos usada pela pista (validação, paredes), pela colisão
// e pelos sensores. Todas as funções aceitam segmentos de comprimento zero
// (as paredes podem repetir pontos seguidos, contrato 1.2): um segmento de A
// a A é o ponto A.

// orientacao é o sinal de (q-p) × (r-p): >0 se r está à esquerda de p→q.
func orientacao(p, q, r Vec2) float64 { return q.Sub(p).Cross(r.Sub(p)) }

// noRetangulo diz se r (colinear com p e q) está dentro do retângulo de p a q.
func noRetangulo(p, q, r Vec2) bool {
	return min(p.X, q.X) <= r.X && r.X <= max(p.X, q.X) &&
		min(p.Y, q.Y) <= r.Y && r.Y <= max(p.Y, q.Y)
}

// segmentosSeTocam diz se os segmentos p1p2 e q1q2 têm algum ponto em comum
// (pontas incluídas).
func segmentosSeTocam(p1, p2, q1, q2 Vec2) bool {
	d1 := orientacao(q1, q2, p1)
	d2 := orientacao(q1, q2, p2)
	d3 := orientacao(p1, p2, q1)
	d4 := orientacao(p1, p2, q2)
	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	return (d1 == 0 && noRetangulo(q1, q2, p1)) ||
		(d2 == 0 && noRetangulo(q1, q2, p2)) ||
		(d3 == 0 && noRetangulo(p1, p2, q1)) ||
		(d4 == 0 && noRetangulo(p1, p2, q2))
}

// interseccaoRetas resolve p + t·r = q + u·s: t e u são os parâmetros do ponto
// em que as retas se cruzam, em cada uma (0 a 1 dentro do segmento, quando r e
// s são as diferenças das pontas). ok = false se as retas são paralelas ou um
// dos vetores tem comprimento zero (o determinante é 0).
func interseccaoRetas(p, r, q, s Vec2) (t, u float64, ok bool) {
	den := r.Cross(s)
	if den == 0 {
		return 0, 0, false
	}
	qp := q.Sub(p)
	return qp.Cross(s) / den, qp.Cross(r) / den, true
}

// pontoDeCruzamento devolve o ponto em que os segmentos p1p2 e q1q2 se cruzam
// (pontas incluídas). Segmentos paralelos (ou de comprimento zero) não se cruzam.
func pontoDeCruzamento(p1, p2, q1, q2 Vec2) (Vec2, bool) {
	r := p2.Sub(p1)
	t, u, ok := interseccaoRetas(p1, r, q1, q2.Sub(q1))
	if !ok {
		return Vec2{}, false
	}
	if t < 0 || t > 1 || u < 0 || u > 1 {
		return Vec2{}, false
	}
	return p1.Add(r.Scale(t)), true
}

// folgaPonta é quanto (em fração do comprimento do trecho) a colisão e os
// sensores esticam cada trecho de parede além das pontas. Sem ela, um caminho
// ou raio que passa exatamente por um vértice da parede pode escapar dos dois
// trechos vizinhos por erro de arredondamento (num dá u = 1+ε, no outro u = -ε).
const folgaPonta = 1e-9

// cruzaTrecho diz se o caminho p1p2 toca o trecho de parede q1q2, com a folga
// folgaPonta nas pontas do trecho. Trecho de comprimento zero vale como ponto.
func cruzaTrecho(p1, p2, q1, q2 Vec2) bool {
	if segmentosSeTocam(p1, p2, q1, q2) {
		return true
	}
	t, u, ok := interseccaoRetas(p1, p2.Sub(p1), q1, q2.Sub(q1))
	if !ok {
		return false
	}
	return t >= 0 && t <= 1 && u >= -folgaPonta && u <= 1+folgaPonta
}

// distanciaRaio devolve a distância, ao longo do raio que sai de o na direção
// unitária d, até o primeiro ponto do segmento ab. ok = false se o raio não
// toca o segmento (com a folga folgaPonta nas pontas). Se o já está sobre o
// segmento, a distância é 0. Segmento colinear com o raio dá a distância até a
// ponta mais próxima à frente; segmento de comprimento zero (a = b) só é tocado
// se o ponto estiver sobre o raio.
func distanciaRaio(o, d, a, b Vec2) (float64, bool) {
	oa := a.Sub(o)
	// t é ao longo do raio; u, ao longo do segmento, de 0 (a) a 1 (b).
	if t, u, ok := interseccaoRetas(o, d, a, b.Sub(a)); ok {
		if t < 0 || u < -folgaPonta || u > 1+folgaPonta {
			return 0, false
		}
		return t, true
	}
	// Paralelo, ou segmento de comprimento zero: só toca se for colinear.
	if oa.Cross(d) != 0 || b.Sub(o).Cross(d) != 0 {
		return 0, false
	}
	ta, tb := oa.Dot(d), b.Sub(o).Dot(d)
	switch {
	case ta < 0 && tb < 0:
		return 0, false // atrás do raio
	case ta <= 0 || tb <= 0:
		return 0, true // a origem está sobre o segmento
	default:
		return min(ta, tb), true
	}
}

// caixasSeparadas diz se a caixa (alinhada aos eixos) do segmento ab não toca a
// caixa de lo a hi. É só um filtro rápido antes das contas exatas.
func caixasSeparadas(a, b, lo, hi Vec2) bool {
	return max(a.X, b.X) < lo.X || min(a.X, b.X) > hi.X ||
		max(a.Y, b.Y) < lo.Y || min(a.Y, b.Y) > hi.Y
}

// distanciaPontoSegmento é a distância do ponto q ao segmento ab (a = b vale
// como o ponto a).
func distanciaPontoSegmento(q, a, b Vec2) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 == 0 {
		return q.Sub(a).Len()
	}
	t := max(0, min(1, q.Sub(a).Dot(ab)/l2))
	return q.Sub(a.Add(ab.Scale(t))).Len()
}

// distanciaEntreSegmentos é a menor distância entre os segmentos ab e cd (0 se
// eles se tocam).
func distanciaEntreSegmentos(a, b, c, d Vec2) float64 {
	if segmentosSeTocam(a, b, c, d) {
		return 0
	}
	return min(distanciaPontoSegmento(a, c, d), distanciaPontoSegmento(b, c, d),
		distanciaPontoSegmento(c, a, b), distanciaPontoSegmento(d, a, b))
}

// trechosParedes percorre todos os trechos das duas paredes da pista (esquerda
// e depois direita), cada parede como polígono fechado (o último ponto liga no
// primeiro). Os trechos podem ter comprimento zero. Pista nil não tem trechos.
func trechosParedes(p *Pista) iter.Seq[Segmento] {
	return func(yield func(Segmento) bool) {
		if p == nil {
			return
		}
		for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
			n := len(w)
			for i := 0; i < n; i++ {
				if !yield(Segmento{A: w[i], B: w[(i+1)%n]}) {
					return
				}
			}
		}
	}
}
