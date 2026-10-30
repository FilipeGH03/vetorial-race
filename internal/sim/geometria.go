package sim

import "math"

// Geometria de segmentos usada pela pista (validação, paredes), pela colisão
// e pelos sensores. Todas as funções aceitam segmentos de comprimento zero
// (as paredes podem repetir pontos seguidos, contrato 1.2): um segmento de A
// a A é o ponto A.

// orientacao é o sinal de (q-p) × (r-p): >0 se r está à esquerda de p→q.
func orientacao(p, q, r Vec2) float64 { return q.Sub(p).Cross(r.Sub(p)) }

// noRetangulo diz se r (colinear com p e q) está dentro do retângulo de p a q.
func noRetangulo(p, q, r Vec2) bool {
	return math.Min(p.X, q.X) <= r.X && r.X <= math.Max(p.X, q.X) &&
		math.Min(p.Y, q.Y) <= r.Y && r.Y <= math.Max(p.Y, q.Y)
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

// pontoDeCruzamento devolve o ponto em que os segmentos p1p2 e q1q2 se cruzam
// (pontas incluídas). Segmentos paralelos (ou de comprimento zero) não se cruzam.
func pontoDeCruzamento(p1, p2, q1, q2 Vec2) (Vec2, bool) {
	r, s := p2.Sub(p1), q2.Sub(q1)
	den := r.Cross(s)
	if den == 0 {
		return Vec2{}, false
	}
	qp := q1.Sub(p1)
	t := qp.Cross(s) / den
	u := qp.Cross(r) / den
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
	r, s := p2.Sub(p1), q2.Sub(q1)
	den := r.Cross(s)
	if den == 0 {
		return false
	}
	qp := q1.Sub(p1)
	t := qp.Cross(s) / den
	u := qp.Cross(r) / den
	return t >= 0 && t <= 1 && u >= -folgaPonta && u <= 1+folgaPonta
}

// distanciaRaio devolve a distância, ao longo do raio que sai de o na direção
// unitária d, até o primeiro ponto do segmento ab. ok = false se o raio não
// toca o segmento (com a folga folgaPonta nas pontas). Se o já está sobre o
// segmento, a distância é 0. Segmento colinear com o raio dá a distância até a
// ponta mais próxima à frente; segmento de comprimento zero (a = b) só é tocado
// se o ponto estiver sobre o raio.
func distanciaRaio(o, d, a, b Vec2) (float64, bool) {
	s := b.Sub(a)
	oa := a.Sub(o)
	den := d.Cross(s)
	if den != 0 {
		t := oa.Cross(s) / den // ao longo do raio
		u := oa.Cross(d) / den // ao longo do segmento, de 0 (a) a 1 (b)
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
		return math.Min(ta, tb), true
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
	t := math.Max(0, math.Min(1, q.Sub(a).Dot(ab)/l2))
	return q.Sub(a.Add(ab.Scale(t))).Len()
}

// distanciaEntreSegmentos é a menor distância entre os segmentos ab e cd (0 se
// eles se tocam).
func distanciaEntreSegmentos(a, b, c, d Vec2) float64 {
	if segmentosSeTocam(a, b, c, d) {
		return 0
	}
	return math.Min(math.Min(distanciaPontoSegmento(a, c, d), distanciaPontoSegmento(b, c, d)),
		math.Min(distanciaPontoSegmento(c, a, b), distanciaPontoSegmento(d, a, b)))
}
