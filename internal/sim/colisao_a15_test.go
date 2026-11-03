package sim

import "testing"

// pistaDeParedes monta uma Pista só com paredes (o que a colisão usa).
func pistaDeParedes(esq, dir []Vec2) *Pista {
	return &Pista{Nome: "a15", ParedeEsq: esq, ParedeDir: dir}
}

// cfgSemAtrito deixa a velocidade do tick igual à de entrada, para conferir a conta do D28.
func cfgSemAtrito() Config {
	cfg := ConfigPadrao()
	cfg.Atrito = 0
	return cfg
}

func quadro(x0, y0, x1, y1 float64) []Vec2 {
	return []Vec2{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

func bate(t *testing.T, p *Pista, pos, vel Vec2) *Carro {
	t.Helper()
	c := carroEm(pos, 0)
	c.Vel = vel
	c.Avancar(Controle{}, p, cfgSemAtrito())
	if !c.Batido {
		t.Fatalf("de %v com Vel %v: não bateu (Pos = %v)", pos, vel, c.Pos)
	}
	if c.Pos != pos {
		t.Errorf("Pos = %v, quer a do começo do tick %v", c.Pos, pos)
	}
	if !c.Vivo {
		t.Error("morreu com BaterMata = false")
	}
	return c
}

func TestBaterDeslizaNaParedeHorizontal(t *testing.T) {
	// Parede de cima em y = 10; o carro sobe e anda para a direita.
	p := pistaDeParedes(quadro(-1000, -1000, 1000, 10), nil)
	c := bate(t, p, Vec2{0, 9.9}, Vec2{300, 100})
	if !perto(c.Vel.X, 150) || !perto(c.Vel.Y, 0) {
		t.Errorf("Vel = %v, quer (150, 0)", c.Vel)
	}
}

func TestBaterDeslizaNaParedeVertical45(t *testing.T) {
	// Parede da direita em x = 10; Vel (200,200): só a componente paralela (y), vezes 0,5.
	p := pistaDeParedes(quadro(-1000, -1000, 10, 1000), nil)
	c := bate(t, p, Vec2{9.9, 0}, Vec2{200, 200})
	if !perto(c.Vel.X, 0) || !perto(c.Vel.Y, 100) {
		t.Errorf("Vel = %v, quer (0, 100)", c.Vel)
	}
}

func TestBaterDeFrenteParaDeslizamentoNulo(t *testing.T) {
	p := pistaDeParedes(quadro(-1000, -1000, 1000, 10), nil)
	c := bate(t, p, Vec2{0, 9.9}, Vec2{0, 300})
	if !perto(c.Vel.X, 0) || !perto(c.Vel.Y, 0) {
		t.Errorf("Vel = %v, quer 0 (contra a parede, sem componente paralela)", c.Vel)
	}
}

func TestBaterEmTrechoDeComprimentoZeroParaOCarro(t *testing.T) {
	ponto := Vec2{10, 0}
	p := pistaDeParedes([]Vec2{ponto, ponto, ponto}, nil)
	c := bate(t, p, Vec2{9.9, 0}, Vec2{300, 0})
	if c.Vel != (Vec2{}) {
		t.Errorf("Vel = %v, quer 0 (trecho de comprimento zero)", c.Vel)
	}
}

func TestBaterUsaOTrechoAtingidoPrimeiro(t *testing.T) {
	// Duas paredes no caminho: x = 10 (vertical, na parede esquerda) e uma horizontal (na direita).
	vertical := []Vec2{{10, -100}, {10, 100}, {10, -100}}
	horizontal := []Vec2{{-100, 10.5}, {100, 10.5}, {-100, 10.5}}
	p := pistaDeParedes(vertical, horizontal)
	de, ate := Vec2{9.9, 10.0}, Vec2{10.1, 11.0}
	tr, ok := trechoAtingido(p, de, ate)
	if !ok {
		t.Fatal("não achou trecho")
	}
	// A vertical é tocada em t = 0,5 (x = 10); a horizontal em t = 0,5 (y = 10,5): empate
	// vai para a primeira parede da lista, a vertical.
	if tr.B.Sub(tr.A).X != 0 {
		t.Errorf("trecho = %v, quer o vertical (empate fica com o primeiro)", tr)
	}
	// Sem empate: a horizontal em y = 10.2 é tocada antes (t = 0,2) da vertical (t = 0,5).
	horizontal = []Vec2{{-100, 10.2}, {100, 10.2}, {-100, 10.2}}
	p = pistaDeParedes(vertical, horizontal)
	tr, ok = trechoAtingido(p, de, ate)
	if !ok || tr.B.Sub(tr.A).Y != 0 {
		t.Errorf("trecho = %v (ok %v), quer o horizontal, tocado primeiro", tr, ok)
	}
}

func TestBaterMataAindaDesliza(t *testing.T) {
	cfg := cfgSemAtrito()
	cfg.BaterMata = true
	p := pistaDeParedes(quadro(-1000, -1000, 1000, 10), nil)
	c := carroEm(Vec2{0, 9.9}, 0)
	c.Vel = Vec2{300, 100}
	c.Avancar(Controle{}, p, cfg)
	if c.Vivo || !c.Batido || c.Pos != (Vec2{0, 9.9}) {
		t.Errorf("Vivo = %v, Batido = %v, Pos = %v", c.Vivo, c.Batido, c.Pos)
	}
}

func TestSemBaterNaoMexeNaVel(t *testing.T) {
	p := pistaDeParedes(quadro(-1000, -1000, 1000, 1000), nil)
	c := carroEm(Vec2{0, 0}, 0)
	c.Vel = Vec2{300, 100}
	c.Avancar(Controle{}, p, cfgSemAtrito())
	if c.Batido || c.Vel != (Vec2{300, 100}) {
		t.Errorf("Batido = %v, Vel = %v", c.Batido, c.Vel)
	}
}
