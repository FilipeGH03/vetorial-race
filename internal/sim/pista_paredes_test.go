package sim

import (
	"fmt"
	"math"
	"sort"
	"testing"
)

// pistasDeForma são pistas válidas com quinas de vários tipos, para conferir
// que as paredes ficam limpas e os checkpoints não se cruzam.
func pistasDeForma(t *testing.T) []*Pista {
	t.Helper()
	ps := []*Pista{carregarOval(t)}
	casos := []struct {
		nome    string
		largura float64
		pontos  []Vec2
	}{
		{"quadrado 200", 200, []Vec2{{0, 0}, {800, 0}, {800, 800}, {0, 800}}},
		{"quadrado 40", 40, []Vec2{{0, 0}, {800, 0}, {800, 800}, {0, 800}}},
		{"quadrado horário", 120, []Vec2{{0, 0}, {0, 800}, {800, 800}, {800, 0}}},
		{"grampo", 90, []Vec2{{0, 0}, {600, 0}, {600, 100}, {0, 100}}},
		{"quebrada", 73.25, []Vec2{{100.125, 100}, {500.3, 120.7}, {610, 333.333333333}, {420.5, 560}, {130, 480.1}}},
		{"L", 100, []Vec2{{0, 0}, {900, 0}, {900, 400}, {400, 400}, {400, 900}, {0, 900}}},
		{"oval largo", 200, []Vec2{{200, 300}, {900, 300}, {1100, 500}, {900, 700}, {200, 700}, {0, 500}}},
	}
	for _, c := range casos {
		p, err := NovaPista(c.nome, c.largura, c.pontos)
		if err != nil {
			t.Fatalf("%s: NovaPista: %v", c.nome, err)
		}
		ps = append(ps, p)
	}
	return ps
}

// distPontoSeg é a distância do ponto p ao segmento ab.
func distPontoSeg(p, a, b Vec2) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 == 0 {
		return p.Sub(a).Len()
	}
	f := math.Max(0, math.Min(1, p.Sub(a).Dot(ab)/l2))
	return p.Sub(a.Add(ab.Scale(f))).Len()
}

// distSegSeg é a distância entre os segmentos ab e cd.
func distSegSeg(a, b, c, d Vec2) float64 {
	if segmentosSeTocam(a, b, c, d) {
		return 0
	}
	return math.Min(math.Min(distPontoSeg(a, c, d), distPontoSeg(b, c, d)),
		math.Min(distPontoSeg(c, a, b), distPontoSeg(d, a, b)))
}

// distParedeCentro é a menor distância entre um trecho da parede w e um trecho
// da linha central c (ambos circuitos fechados).
func distParedeCentro(w, c []Vec2) float64 {
	min := math.Inf(1)
	for i := range w {
		for j := range c {
			d := distSegSeg(w[i], w[(i+1)%len(w)], c[j], c[(j+1)%len(c)])
			min = math.Min(min, d)
		}
	}
	return min
}

func TestParedesSaoPoligonosSimples(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		for _, w := range []struct {
			nome string
			p    []Vec2
		}{{"esq", p.ParedeEsq}, {"dir", p.ParedeDir}} {
			if len(w.p) != len(p.Centro) {
				t.Errorf("%s: parede %s tem %d pontos, want %d", p.Nome, w.nome, len(w.p), len(p.Centro))
			}
			if i, j, ok := cruzaPropria(semRepetidos(w.p)); ok {
				t.Errorf("%s: parede %s cruza a si mesma (trechos %d e %d)", p.Nome, w.nome, i, j)
			}
		}
		if i, j, ok := cruzaEntre(p.ParedeEsq, p.ParedeDir); ok {
			t.Errorf("%s: parede esq (trecho %d) cruza a dir (trecho %d)", p.Nome, i, j)
		}
	}
}

func TestParedesNaoInvademAPista(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		meia := p.Largura / 2
		for _, w := range []struct {
			nome string
			p    []Vec2
		}{{"esq", p.ParedeEsq}, {"dir", p.ParedeDir}} {
			d := distParedeCentro(w.p, p.Centro)
			t.Logf("%s: parede %s a %.2f px da linha central (meia largura %v)", p.Nome, w.nome, d, meia)
			if d < meia-1e-6 {
				t.Errorf("%s: parede %s chega a %.2f px da linha central, meia largura %v", p.Nome, w.nome, d, meia)
			}
		}
	}
}

func TestCheckpointsNaoSeCruzam(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		cps := p.Checkpoints
		for i := range cps {
			if cps[i].A != p.ParedeEsq[i] || cps[i].B != p.ParedeDir[i] {
				t.Errorf("%s: checkpoint %d não vai de ParedeEsq[%d] a ParedeDir[%d]", p.Nome, i, i, i)
			}
			for j := i + 1; j < len(cps); j++ {
				a, b := cps[i], cps[j]
				comum := a.A == b.A || a.B == b.B
				if !comum && segmentosSeTocam(a.A, a.B, b.A, b.B) {
					t.Errorf("%s: checkpoints %d e %d se cruzam", p.Nome, i, j)
				}
			}
		}
	}
}

// toqueComFolga diz se ab toca cd e devolve a fração de ab onde toca. Tem folga
// de arredondamento: os caminhos de teste passam exatamente pelos checkpoints.
func toqueComFolga(a, b, c, d Vec2) (float64, bool) {
	const eps = 1e-9
	r, s := b.Sub(a), d.Sub(c)
	den := r.Cross(s)
	if den == 0 {
		return 0, false
	}
	t := c.Sub(a).Cross(s) / den
	u := c.Sub(a).Cross(r) / den
	return t, t >= -eps && t <= 1+eps && u >= -eps && u <= 1+eps
}

// sequenciaDeCheckpoints anda pelo circuito fechado q e devolve os índices dos
// checkpoints na ordem em que são tocados (sem repetir o mesmo em seguida).
func sequenciaDeCheckpoints(q []Vec2, cps []Segmento) []int {
	var seq []int
	for i := range q {
		a, b := q[i], q[(i+1)%len(q)]
		type toque struct {
			t float64
			k int
		}
		var ts []toque
		for k, cp := range cps {
			if t, ok := toqueComFolga(a, b, cp.A, cp.B); ok {
				ts = append(ts, toque{t, k})
			}
		}
		sort.Slice(ts, func(i, j int) bool { return ts[i].t < ts[j].t })
		for _, x := range ts {
			if len(seq) == 0 || seq[len(seq)-1] != x.k {
				seq = append(seq, x.k)
			}
		}
	}
	for len(seq) > 1 && seq[len(seq)-1] == seq[0] {
		seq = seq[:len(seq)-1]
	}
	return seq
}

// Um carro que anda pela pista a qualquer distância das paredes cruza cada
// checkpoint uma vez por volta, na ordem 0, 1, 2, ... O caminho de teste passa
// pelo ponto a uma fração f de cada checkpoint (f perto de 0 é rente à parede
// esquerda, perto de 1 rente à direita); a linha central também é testada.
func TestCheckpointsCruzadosEmOrdem(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		cps := p.Checkpoints
		n := len(cps)
		caminhos := map[string][]Vec2{"linha central": p.Centro}
		for _, f := range []float64{0.02, 0.2, 0.5, 0.8, 0.98} {
			q := make([]Vec2, n)
			for i, cp := range cps {
				q[i] = cp.A.Add(cp.B.Sub(cp.A).Scale(f))
			}
			caminhos[fmt.Sprintf("f=%v", f)] = q
		}
		for nome, q := range caminhos {
			seq := sequenciaDeCheckpoints(q, cps)
			ok := len(seq) == n
			for j := 0; ok && j < len(seq); j++ {
				ok = seq[(j+1)%len(seq)] == (seq[j]+1)%n
			}
			if !ok {
				t.Errorf("%s, %s: checkpoints tocados fora de ordem: %v", p.Nome, nome, seq)
			}
		}
	}
}
