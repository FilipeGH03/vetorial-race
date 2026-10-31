package sim

import (
	"math"
	"testing"
)

// carroEm cria um carro vivo em pos, virado para o ângulo a (rad).
func carroEm(pos Vec2, a float64) *Carro {
	c := NovoCarro(1, pos)
	c.DefinirAngulo(a)
	return &c
}

// pistaRetangulo é um retângulo de 3000 x 1000 com retas longas, onde o raio da
// frente, no meio da reta de baixo, não alcança parede nenhuma.
func pistaRetangulo(t *testing.T) *Pista {
	t.Helper()
	p, err := NovaPista("retangulo", 90, []Vec2{{0, 0}, {3000, 0}, {3000, 1000}, {0, 1000}})
	if err != nil {
		t.Fatalf("NovaPista: %v", err)
	}
	return p
}

// distParede é a menor distância do ponto q às duas paredes da pista.
func distParede(p *Pista, q Vec2) float64 {
	d := math.Inf(1)
	for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for i := range w {
			d = math.Min(d, distPontoSeg(q, w[i], w[(i+1)%len(w)]))
		}
	}
	return d
}

// angRad é o ângulo do sensor k em radianos: k·360/7 graus a partir de +x (D20).
func angRad(k int) float64 { return 2 * math.Pi * float64(k) / NumSensores }

func TestSensoresFixosNoMundoNaRetaDoOval(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	// Reta de baixo do oval: de (200,300) a (600,300); paredes em y = 345 e y = 255.
	c := carroEm(Vec2{400, 300}, 0)
	l := Ler(c, p, cfg)
	meia := p.Largura / 2
	want := meia / cfg.AlcanceSensor // 45/300 = 0,15
	// k = 2 (~103°) e k = 5 (~257°) apontam quase para cima e para baixo.
	for _, k := range []int{2, 5} {
		if math.Abs(l.Sensores[k]-want)/want > 0.03 {
			t.Errorf("sensor %d (%.1f°) = %v, quer ≈ %v", k, angulosSensores[k], l.Sensores[k], want)
		}
	}
	// Os que cruzam a reta (k = 1..6) batem a meia/|sen θ| do centro.
	for k := 1; k < NumSensores; k++ {
		w := meia / math.Abs(math.Sin(angRad(k))) / cfg.AlcanceSensor
		if math.Abs(l.Sensores[k]-w) > 1e-9 {
			t.Errorf("sensor %d (%.1f°) = %v, quer %v", k, angulosSensores[k], l.Sensores[k], w)
		}
	}
	for k, s := range l.Sensores {
		if !(s >= 0 && s <= 1) {
			t.Errorf("sensor %d = %v fora de [0,1]", k, s)
		}
	}
	// k = 0 (+x) segue a reta e bate na parede de fora da curva, antes do alcance.
	if !(l.Sensores[0] > want && l.Sensores[0] < 1) {
		t.Errorf("sensor 0 (+x) = %v, quer entre %v e 1", l.Sensores[0], want)
	}
}

// Girar o ângulo do carro não muda sensor nenhum nem DirPista (D20).
func TestLeituraNaoMudaComOAnguloDoCarro(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	ref := Ler(carroEm(Vec2{400, 320}, 0), p, cfg)
	for _, a := range []float64{0.3, math.Pi / 2, math.Pi, -2.2, 2 * math.Pi, 10 * math.Pi} {
		l := Ler(carroEm(Vec2{400, 320}, a), p, cfg)
		if l.Sensores != ref.Sensores {
			t.Errorf("ângulo %v: sensores %v, quer %v", a, l.Sensores, ref.Sensores)
		}
		if l.DirPista != ref.DirPista {
			t.Errorf("ângulo %v: DirPista %v, quer %v", a, l.DirPista, ref.DirPista)
		}
		if l.Angulo != a {
			t.Errorf("Angulo = %v, quer %v", l.Angulo, a)
		}
	}
	// Os raios (para a tela de treino) também não giram.
	r0 := RaiosSensores(carroEm(Vec2{400, 320}, 0), p, cfg)
	r1 := RaiosSensores(carroEm(Vec2{400, 320}, 1.234), p, cfg)
	if r0 != r1 {
		t.Errorf("RaiosSensores muda com o ângulo do carro: %v e %v", r0, r1)
	}
}

func TestSensorSemParedeDaUm(t *testing.T) {
	p := pistaRetangulo(t)
	cfg := ConfigPadrao()
	c := carroEm(Vec2{1000, 0}, 0) // meio da reta de baixo
	l := Ler(c, p, cfg)
	if l.Sensores[0] != 1 {
		t.Errorf("sensor 0 (+x) sem parede = %v, quer 1", l.Sensores[0])
	}
	// Os outros cruzam a reta e batem a 45/|sen θ| do centro, bem antes do alcance.
	for k := 1; k < NumSensores; k++ {
		if l.Sensores[k] >= 1 {
			t.Errorf("sensor %d (%.1f°) = %v, quer < 1", k, angulosSensores[k], l.Sensores[k])
		}
	}

	// Longe de tudo, todos os raios dão 1.
	longe := carroEm(Vec2{1500, 500}, 0.3)
	for i, s := range Ler(longe, p, cfg).Sensores {
		if s != 1 {
			t.Errorf("carro longe das paredes: sensor %d = %v, quer 1", i, s)
		}
	}
	// Sem pista também.
	for i, s := range Ler(c, nil, cfg).Sensores {
		if s != 1 {
			t.Errorf("pista nil: sensor %d = %v, quer 1", i, s)
		}
	}
}

func TestLeituraUsaAnguloDoCarro(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	// Andando para +x, mas com o ângulo guardado para +y: Angulo segue c.Angulo()
	// e Vel/Modulo seguem o carro; os sensores não dependem do ângulo (D20).
	c := carroEm(Vec2{400, 300}, math.Pi/2)
	c.Vel = Vec2{30, -40}
	l := Ler(c, p, cfg)
	if l.Angulo != c.Angulo() || l.Angulo != math.Pi/2 {
		t.Errorf("Angulo = %v, quer c.Angulo() = %v", l.Angulo, c.Angulo())
	}
	if l.Vel != c.Vel || l.Modulo != 50 {
		t.Errorf("Vel = %v, Modulo = %v; quer %v e 50", l.Vel, l.Modulo, c.Vel)
	}
	if ref := Ler(carroEm(Vec2{400, 300}, 0), p, cfg); l.Sensores != ref.Sensores {
		t.Errorf("sensores %v mudaram com o ângulo; quer %v", l.Sensores, ref.Sensores)
	}
}

// DirPista na reta de baixo do oval é (1,0) nos dois vetores; numa quina os dois
// diferem (D20, D23). O trecho vem da posição do carro, não do Progresso.
func TestDirPistaRetaEQuinaDoOval(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	c := carroEm(Vec2{400, 300}, 0)
	l := Ler(c, p, cfg)
	for j, d := range l.DirPista {
		if math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
			t.Errorf("reta: DirPista[%d] = %v, quer (1,0)", j, d)
		}
	}
	// Perto de (600,300), onde a pista vira para (700,400): o trecho mais perto
	// ainda é reto e 5 trechos adiante já é a diagonal.
	c = carroEm(Vec2{540, 300}, 0)
	l = Ler(c, p, cfg)
	if l.DirPista[0].Sub(l.DirPista[1]).Len() < 0.1 {
		t.Errorf("quina: DirPista = %v, quer vetores diferentes", l.DirPista)
	}
	if d := l.DirPista[0]; math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
		t.Errorf("quina: DirPista[0] = %v, quer (1,0)", d)
	}
}

// DirPista segue a fórmula do contrato (D23) em qualquer pista: j é o trecho da
// linha central mais perto do carro, e os vetores são sempre unitários. Não
// depende de Progresso, qualquer que seja o Proximo (inclusive fora de [0, m)).
func TestDirPistaFormulaEUnitarios(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		m := len(p.Centro)
		for _, j := range []int{0, 1, 2, m / 2, m - 7, m - 6, m - 2, m - 1} {
			// O meio do trecho j está a 0 px dele, e de nenhum outro.
			meio := p.Centro[j].Add(p.Centro[(j+1)%m]).Scale(0.5)
			want := [2]Vec2{
				p.Centro[(j+1)%m].Sub(p.Centro[j]).Norm(),
				p.Centro[(j+6)%m].Sub(p.Centro[(j+5)%m]).Norm(),
			}
			for _, prox := range []int{0, 3, m / 2, m + 3, -1} {
				c := carroEm(meio, 0)
				c.Progresso.Proximo = prox
				l := Ler(c, p, cfg)
				for k, d := range l.DirPista {
					if math.Abs(d.Len()-1) > 1e-9 {
						t.Errorf("%s, trecho %d, Proximo %d: |DirPista[%d]| = %v, quer 1", p.Nome, j, prox, k, d.Len())
					}
					if d.Sub(want[k]).Len() > 1e-12 {
						t.Errorf("%s, trecho %d, Proximo %d: DirPista[%d] = %v, quer %v", p.Nome, j, prox, k, d, want[k])
					}
				}
			}
		}
	}
	// Sem pista: vetores nulos, sem pânico.
	if l := Ler(carroEm(Vec2{}, 0), nil, cfg); l.DirPista != ([2]Vec2{}) {
		t.Errorf("pista nil: DirPista = %v, quer nulos", l.DirPista)
	}
}

// ovalAntigo é o oval de exemplo com o 1º ponto no começo da reta de baixo
// (antes da D23), para testar a largada em quina.
func ovalAntigo(t *testing.T) *Pista {
	t.Helper()
	p, err := NovaPista("oval-antigo", 90, []Vec2{{200, 300}, {600, 300}, {700, 400}, {600, 500}, {200, 500}, {100, 400}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Um carro na diagonal antes da largada do oval antigo recebe a direção da
// diagonal, não (1,0) como pelo Progresso.Proximo = 0 (D23), e o Progresso não
// muda o resultado.
func TestDirPistaNaDiagonalAntesDaLargada(t *testing.T) {
	p := ovalAntigo(t)
	cfg := ConfigPadrao()
	diag := Vec2{1, -1}.Norm() // (100,400) -> (200,300)
	c := carroEm(Vec2{188, 375}, 0)
	ref := Ler(c, p, cfg).DirPista
	if d := ref[0]; d.Sub(diag).Len() > 1e-6 {
		t.Errorf("DirPista[0] = %v, quer paralelo à diagonal %v", d, diag)
	}
	for _, prox := range []int{0, 1, 7, len(p.Centro) - 1, -3, 1000} {
		c.Progresso = Progresso{Proximo: prox, Voltas: prox % 3, Distancia: float64(prox) * 17}
		if got := Ler(c, p, cfg).DirPista; got != ref {
			t.Errorf("Progresso %+v: DirPista = %v, quer %v (independe de Progresso)", c.Progresso, got, ref)
		}
	}
}

// AvancoNaLinha: > 0 andando no sentido da pista (antes e depois da largada),
// < 0 de ré, ≈ 0 parado, e a soma de uma volta é ≈ Comprimento (D23).
func TestAvancoNaLinha(t *testing.T) {
	for _, p := range []*Pista{carregarOval(t), ovalAntigo(t)} {
		// Parado.
		q := Vec2{188, 375}
		if a := AvancoNaLinha(p, q, q); a != 0 {
			t.Errorf("%s: parado deu %v, quer 0", p.Nome, a)
		}
		// Sobre a linha central: cada passo de ~5 px no sentido da pista dá > 0
		// (inclusive ao cruzar a largada e a volta do índice) e de ré dá < 0; a
		// soma de uma volta é o comprimento.
		var soma, somaRe float64
		passos := int(p.Comprimento / 5)
		ponto := func(i int) Vec2 { return pontoDaLinha(p, float64(i)*p.Comprimento/float64(passos)) }
		for i := 0; i < passos; i++ {
			a := AvancoNaLinha(p, ponto(i), ponto(i+1))
			b := AvancoNaLinha(p, ponto(i+1), ponto(i))
			if a <= 0 || b >= 0 {
				t.Fatalf("%s: passo %d: avanço %v, de ré %v; quer > 0 e < 0", p.Nome, i, a, b)
			}
			soma += a
			somaRe += b
		}
		if math.Abs(soma-p.Comprimento) > 1 || math.Abs(somaRe+p.Comprimento) > 1 {
			t.Errorf("%s: soma de uma volta = %v e de ré = %v, quer ±%v", p.Nome, soma, somaRe, p.Comprimento)
		}
	}
	// Fora da linha central: andar 10 px para +x na diagonal (-x, +y) do oval
	// antigo, que aponta para (1,-1): avança 10/√2 px.
	p := ovalAntigo(t)
	if a := AvancoNaLinha(p, Vec2{150, 350}, Vec2{160, 350}); math.Abs(a-10/math.Sqrt2) > 1e-6 {
		t.Errorf("diagonal: avanço = %v, quer %v", a, 10/math.Sqrt2)
	}
	if AvancoNaLinha(nil, Vec2{}, Vec2{1, 1}) != 0 {
		t.Error("pista nil: quer 0")
	}
}

// pontoDaLinha é o ponto da linha central a s px de Centro[0] (0 ≤ s ≤ Comprimento).
func pontoDaLinha(p *Pista, s float64) Vec2 {
	n := len(p.Centro)
	j := min(int(s/p.Passo), n-1)
	f := (s - float64(j)*p.Passo) / p.Passo
	a, b := p.Centro[j], p.Centro[(j+1)%n]
	return a.Add(b.Sub(a).Scale(f))
}

func TestRaiosSensores(t *testing.T) {
	cfg := ConfigPadrao()
	for _, p := range pistasDeForma(t) {
		for i := 0; i < len(p.Centro); i += 3 {
			n := len(p.Centro)
			dir := p.Centro[(i+1)%n].Sub(p.Centro[i])
			c := carroEm(p.Centro[i], dir.Angle())
			l := Ler(c, p, cfg)
			raios := RaiosSensores(c, p, cfg)
			for k, r := range raios {
				s := l.Sensores[k]
				if !(s >= 0 && s <= 1) {
					t.Fatalf("%s, ponto %d: sensor %d = %v fora de [0,1]", p.Nome, i, k, s)
				}
				if r.A != c.Pos {
					t.Fatalf("%s, ponto %d: raio %d sai de %v, quer %v", p.Nome, i, k, r.A, c.Pos)
				}
				if math.Abs(r.B.Sub(r.A).Len()-s*cfg.AlcanceSensor) > 1e-6 {
					t.Errorf("%s, ponto %d: raio %d mede %v, sensor diz %v", p.Nome, i, k, r.B.Sub(r.A).Len(), s*cfg.AlcanceSensor)
				}
				wantAng := angRad(k)
				if d := r.B.Sub(r.A); math.Abs(math.Remainder(d.Angle()-wantAng, 2*math.Pi)) > 1e-9 {
					t.Errorf("%s, ponto %d: raio %d no ângulo %v, quer %v", p.Nome, i, k, d.Angle(), wantAng)
				}
				if s < 1 && distParede(p, r.B) > 1e-6 {
					t.Errorf("%s, ponto %d: raio %d termina a %v px da parede", p.Nome, i, k, distParede(p, r.B))
				}
				// Da linha central, a parede está a pelo menos meia largura.
				if s*cfg.AlcanceSensor < p.Largura/2-1e-6 {
					t.Errorf("%s, ponto %d: sensor %d vê parede a %v px, meia largura %v", p.Nome, i, k, s*cfg.AlcanceSensor, p.Largura/2)
				}
			}
		}
	}
}

// TestSensorEmPontoRepetido mira um raio exatamente num ponto que a parede
// repete (trecho de comprimento zero, onde o laço da quina foi cortado): o carro
// fica a 50 px do ponto, na direção oposta à de cada sensor k.
func TestSensorEmPontoRepetido(t *testing.T) {
	cfg := ConfigPadrao()
	const dist = 50.0
	achou := 0
	for _, p := range pistasDeForma(t) {
		for _, w := range [][]Vec2{p.ParedeEsq, p.ParedeDir} {
			n := len(w)
			for i := range w {
				if w[i] != w[(i+1)%n] {
					continue
				}
				achou++
				for k := 0; k < NumSensores; k++ {
					c := carroEm(w[i].Sub(direcaoSensor(k).Scale(dist)), 0) // o sensor k aponta para o ponto
					got := Ler(c, p, cfg).Sensores[k] * cfg.AlcanceSensor
					if math.IsNaN(got) || got > dist+1e-6 || got < distParede(p, c.Pos)-1e-6 {
						t.Errorf("%s, ponto %d, sensor %d: %v px, quer entre %v e %v", p.Nome, i, k, got, distParede(p, c.Pos), dist)
					}
				}
			}
		}
	}
	if achou == 0 {
		t.Fatal("nenhuma pista de teste tem parede com ponto repetido; o teste não exercita o caso")
	}
}

func TestSensoresComAlcanceZero(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	cfg.AlcanceSensor = 0
	c := carroEm(Vec2{400, 300}, 0)
	for i, s := range Ler(c, p, cfg).Sensores {
		if s != 1 {
			t.Errorf("alcance 0: sensor %d = %v, quer 1", i, s)
		}
	}
	for i, r := range RaiosSensores(c, p, cfg) {
		if r.A != r.B {
			t.Errorf("alcance 0: raio %d = %v, quer de comprimento zero", i, r)
		}
	}
}
