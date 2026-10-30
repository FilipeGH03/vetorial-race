package sim

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

var ovalPontos = []Vec2{{200, 300}, {600, 300}, {700, 400}, {600, 500}, {200, 500}, {100, 400}}

// O oval do contrato tem 800 + 400·√2 ≈ 1365,7 px: 68 pontos reamostrados, um
// checkpoint por ponto, e a mesma contagem em todo lugar.
func TestOvalTem68Checkpoints(t *testing.T) {
	p := carregarOval(t)
	if len(p.Centro) != 68 || len(p.Checkpoints) != 68 || len(p.ParedeEsq) != 68 || len(p.ParedeDir) != 68 {
		t.Errorf("centro=%d checkpoints=%d esq=%d dir=%d, want 68 em todos", len(p.Centro), len(p.Checkpoints), len(p.ParedeEsq), len(p.ParedeDir))
	}
}

// Carregar duas vezes o mesmo arquivo dá a mesma pista (sem aleatoriedade).
func TestCarregarEDeterministico(t *testing.T) {
	a, b := carregarOval(t), carregarOval(t)
	if !reflect.DeepEqual(a, b) {
		t.Error("duas leituras do oval deram pistas diferentes")
	}
}

// O ponto em que o desenho começa muda só a largada: a pista continua válida
// e com o mesmo comprimento e o mesmo número de checkpoints.
func TestPistaValidaComQualquerPontoInicial(t *testing.T) {
	ref, err := NovaPista("oval", 90, ovalPontos)
	if err != nil {
		t.Fatal(err)
	}
	for r := range ovalPontos {
		q := append(append([]Vec2{}, ovalPontos[r:]...), ovalPontos[:r]...)
		p, err := NovaPista("rot", 90, q)
		if err != nil {
			t.Errorf("início no ponto %d: %v", r, err)
			continue
		}
		if len(p.Checkpoints) != len(ref.Checkpoints) || !perto(p.Comprimento, ref.Comprimento) {
			t.Errorf("início no ponto %d: %d checkpoints e comprimento %v, want %d e %v", r, len(p.Checkpoints), p.Comprimento, len(ref.Checkpoints), ref.Comprimento)
		}
		if p.Centro[0] != q[0] {
			t.Errorf("início no ponto %d: Centro[0] = %v, want %v (largada)", r, p.Centro[0], q[0])
		}
	}
}

// Sentido horário: a esquerda de quem anda é a parede de fora, e os
// checkpoints continuam indo da esquerda para a direita de quem anda.
func TestOvalHorario(t *testing.T) {
	rev := make([]Vec2, len(ovalPontos))
	for i, q := range ovalPontos {
		rev[len(rev)-1-i] = q
	}
	p, err := NovaPista("oval horario", 90, rev)
	if err != nil {
		t.Fatal(err)
	}
	ac, ae, ad := area(p.Centro), area(p.ParedeEsq), area(p.ParedeDir)
	if !(ac < 0 && ae < ac && ac < ad && ad < 0) {
		t.Errorf("áreas: esq=%v centro=%v dir=%v; no sentido horário want esq < centro < dir < 0", ae, ac, ad)
	}
	n := len(p.Centro)
	for i := range p.Centro {
		tan := p.Centro[(i+1)%n].Sub(p.Centro[(i-1+n)%n])
		if tan.Cross(p.ParedeEsq[i].Sub(p.Centro[i])) <= 0 || tan.Cross(p.ParedeDir[i].Sub(p.Centro[i])) >= 0 {
			t.Errorf("ponto %d: ParedeEsq não está à esquerda ou ParedeDir não está à direita de quem anda", i)
		}
	}
}

// O editor pode gravar o ponto de fechamento repetido, pontos duplicados, pontos
// no meio de uma reta e coordenadas negativas: nada disso é erro.
func TestPistaAceitaDesenhosComuns(t *testing.T) {
	casos := []struct {
		nome   string
		pontos []Vec2
	}{
		{"último igual ao primeiro", append(append([]Vec2{}, ovalPontos...), ovalPontos[0])},
		{"ponto duplicado no meio", []Vec2{{200, 300}, {600, 300}, {600, 300}, {700, 400}, {600, 500}, {200, 500}, {100, 400}}},
		{"ponto no meio da reta", []Vec2{{200, 300}, {400, 300}, {600, 300}, {700, 400}, {600, 500}, {200, 500}, {100, 400}}},
		{"coordenadas negativas", []Vec2{{-200, -300}, {200, -300}, {300, -200}, {200, -100}, {-200, -100}, {-300, -200}}},
		{"S com quinas de 90°", []Vec2{{0, 0}, {800, 0}, {800, 200}, {200, 200}, {200, 400}, {800, 400}, {800, 700}, {0, 700}}},
	}
	for _, tc := range casos {
		p, err := NovaPista(tc.nome, 60, tc.pontos)
		if err != nil {
			t.Errorf("%s: %v", tc.nome, err)
			continue
		}
		if _, _, ok := cruzaEntre(p.ParedeEsq, p.ParedeDir); ok {
			t.Errorf("%s: as paredes se cruzam", tc.nome)
		}
		if i, j, ok := cruzaPropria(semRepetidos(p.ParedeEsq)); ok {
			t.Errorf("%s: parede esquerda cruza a si mesma (%d, %d)", tc.nome, i, j)
		}
		if i, j, ok := cruzaPropria(semRepetidos(p.ParedeDir)); ok {
			t.Errorf("%s: parede direita cruza a si mesma (%d, %d)", tc.nome, i, j)
		}
	}
}

// Curvas suaves aleatórias (com semente fixa), nos dois sentidos e com larguras
// de 40 a 200, são pistas válidas: paredes simples que não invadem a pista e
// checkpoints tocados em ordem.
func TestPistasSuavesAleatorias(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for k := 0; k < 25; k++ {
		R := 250 + rng.Float64()*300
		a2, a3 := rng.Float64()*0.25*R, rng.Float64()*0.2*R
		ph2, ph3 := rng.Float64()*2*math.Pi, rng.Float64()*2*math.Pi
		m := 24 + rng.Intn(60)
		horario := k%2 == 1
		pontos := make([]Vec2, m)
		for i := range pontos {
			a := 2 * math.Pi * float64(i) / float64(m)
			r := R + a2*math.Sin(2*a+ph2) + a3*math.Sin(3*a+ph3)
			if horario {
				a = -a
			}
			pontos[i] = Vec2{800 + r*math.Cos(a), 500 + r*math.Sin(a)}
		}
		w := 40 + rng.Float64()*160
		p, err := NovaPista("aleatoria", w, pontos)
		if err != nil {
			t.Errorf("curva %d (R=%.0f, largura=%.0f, %d pontos): %v", k, R, w, m, err)
			continue
		}
		if _, _, ok := cruzaEntre(p.ParedeEsq, p.ParedeDir); ok {
			t.Errorf("curva %d: as paredes se cruzam", k)
		}
		if _, _, ok := cruzaPropria(semRepetidos(p.ParedeEsq)); ok {
			t.Errorf("curva %d: parede esquerda cruza a si mesma", k)
		}
		if _, _, ok := cruzaPropria(semRepetidos(p.ParedeDir)); ok {
			t.Errorf("curva %d: parede direita cruza a si mesma", k)
		}
		for nome, wall := range map[string][]Vec2{"esq": p.ParedeEsq, "dir": p.ParedeDir} {
			if d := distParedeCentro(wall, p.Centro); d < w/2-1e-6 {
				t.Errorf("curva %d: parede %s a %.2f px da linha central, meia largura %.2f", k, nome, d, w/2)
			}
		}
		seq := sequenciaDeCheckpoints(p.Centro, p.Checkpoints)
		ok := len(seq) == len(p.Checkpoints)
		for j := 0; ok && j < len(seq); j++ {
			ok = seq[(j+1)%len(seq)] == (seq[j]+1)%len(seq)
		}
		if !ok {
			t.Errorf("curva %d: checkpoints tocados fora de ordem", k)
		}
	}
}

// Circuito grande (5000×3000 px) carrega sem erro e sem estourar nada.
func TestPistaGrande(t *testing.T) {
	pontos := make([]Vec2, 400)
	for i := range pontos {
		a := 2 * math.Pi * float64(i) / float64(len(pontos))
		pontos[i] = Vec2{5000 * math.Cos(a), 3000 * math.Sin(a)}
	}
	p, err := NovaPista("grande", 90, pontos)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Checkpoints) != len(p.Centro) || len(p.Centro) < 1000 {
		t.Errorf("centro=%d checkpoints=%d", len(p.Centro), len(p.Checkpoints))
	}
}

// Arquivos com formas estranhas de JSON dão erro (sem pânico) ou são lidos como o contrato manda.
func TestLerPistaFormasDeJSON(t *testing.T) {
	quad := `[[0,0],[800,0],[800,800],[0,800]]`
	erros := []struct {
		nome string
		json string
		want error
	}{
		{"vazio", ``, ErrPistaInvalida},
		{"centro nulo", `{"versao":1,"nome":"x","largura":90,"centro":null}`, ErrPistaCurta},
		{"centro vazio", `{"versao":1,"nome":"x","largura":90,"centro":[]}`, ErrPistaCurta},
		{"sem centro", `{"versao":1,"nome":"x","largura":90}`, ErrPistaCurta},
		{"sem largura", `{"versao":1,"nome":"x","centro":` + quad + `}`, ErrPistaInvalida},
		{"largura em texto", `{"versao":1,"nome":"x","largura":"90","centro":` + quad + `}`, ErrPistaInvalida},
		{"coordenada em texto", `{"versao":1,"nome":"x","largura":90,"centro":[["0","0"],[800,0],[800,800],[0,800]]}`, ErrPistaInvalida},
		{"ponto nulo", `{"versao":1,"nome":"x","largura":90,"centro":[null,[800,0],[800,800],[0,800]]}`, ErrPistaInvalida},
		{"sem nome", `{"versao":1,"largura":90,"centro":` + quad + `}`, ErrPistaInvalida},
		{"versao 0", `{"versao":0,"nome":"x","largura":90,"centro":` + quad + `}`, ErrPistaInvalida},
		{"raiz é lista", `[1,2,3]`, ErrPistaInvalida},
	}
	for _, tc := range erros {
		if _, err := LerPista(strings.NewReader(tc.json)); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.nome, err, tc.want)
		}
	}
	// Campo a mais é ignorado (o arquivo pode ganhar campos em versões futuras do editor).
	p, err := LerPista(strings.NewReader(`{"versao":1,"nome":"x","largura":90,"extra":true,"centro":` + quad + `}`))
	if err != nil || p == nil || p.Nome != "x" || len(p.Pontos) != 4 {
		t.Errorf("campo extra: err=%v pista=%v", err, p)
	}
}

// Largura NaN ou infinita (vinda de um editor com defeito) não passa nem causa pânico.
func TestLarguraNaoFinitaDaErro(t *testing.T) {
	for _, l := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, -90} {
		if _, err := NovaPista("x", l, ovalPontos); !errors.Is(err, ErrPistaInvalida) {
			t.Errorf("largura %v: err = %v, want ErrPistaInvalida", l, err)
		}
	}
	nan := append([]Vec2{{math.NaN(), 0}}, ovalPontos...)
	if _, err := NovaPista("x", 90, nan); !errors.Is(err, ErrPistaInvalida) {
		t.Errorf("ponto NaN: err = %v, want ErrPistaInvalida", err)
	}
}
