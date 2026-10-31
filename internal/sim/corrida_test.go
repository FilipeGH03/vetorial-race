package sim

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

// seguidor é um controlador falso: persegue o ponto da linha central 2 pontos à
// frente do mais perto do carro com velocidade alvo vAlvo. Com paraEm > 0, a
// partir desse tick ele freia até parar. Lê a posição direto do carro (o
// controlador de verdade só recebe a Leitura); c é ligado depois de NovaCorrida.
type seguidor struct {
	p      *Pista
	c      *Carro
	vAlvo  float64
	paraEm int
	ticks  int
}

func (s *seguidor) Decidir(Leitura) Controle {
	s.ticks++
	var alvo Vec2
	if s.paraEm == 0 || s.ticks < s.paraEm {
		n := len(s.p.Centro)
		melhor, dist := 0, math.Inf(1)
		for i, q := range s.p.Centro {
			v := q.Sub(s.c.Pos)
			if d := v.Dot(v); d < dist {
				melhor, dist = i, d
			}
		}
		alvo = s.p.Centro[(melhor+2)%n].Sub(s.c.Pos).Norm().Scale(s.vAlvo)
	}
	erro := alvo.Sub(s.c.Vel)
	var ctl Controle
	for _, e := range []struct {
		eixo Eixo
		v    float64
	}{{EixoX, erro.X}, {EixoY, erro.Y}} {
		if math.Abs(e.v) > 10 {
			sinal := int8(1)
			if e.v < 0 {
				sinal = -1
			}
			ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e.eixo, Sinal: sinal})
		}
	}
	return ctl
}

// parado é um controlador que nunca age.
type parado struct{}

func (parado) Decidir(Leitura) Controle { return Controle{} }

// reta acelera sempre para +x.
type reta struct{}

func (reta) Decidir(Leitura) Controle {
	return Controle{Acoes: []Acao{{Ativa: true, Eixo: EixoX, Sinal: 1}}}
}

// novaCorridaComSeguidores monta a corrida e liga cada seguidor ao seu carro.
func novaCorridaComSeguidores(t testing.TB, p *Pista, modo Modo, voltas int, ctls ...Controlador) *Corrida {
	t.Helper()
	r, err := NovaCorrida(p, ConfigPadrao(), modo, voltas, ctls, -1)
	if err != nil {
		t.Fatalf("NovaCorrida: %v", err)
	}
	for i, c := range ctls {
		if s, ok := c.(*seguidor); ok {
			s.c = &r.Carros[i]
		}
	}
	return r
}

func TestCorridaVoltasVenceOPrimeiro(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoVoltas, 2,
		&seguidor{p: p, vAlvo: 120},
		&seguidor{p: p, vAlvo: 200},
		&seguidor{p: p, vAlvo: 160},
	)
	if len(r.Carros) != 3 || r.Vencedor != -1 || r.Terminada {
		t.Fatalf("estado inicial errado: %d carros, vencedor %d, terminada %v", len(r.Carros), r.Vencedor, r.Terminada)
	}
	for tick := 0; tick < 60*120 && !r.Terminada; tick++ {
		r.Step()
	}
	if !r.Terminada {
		t.Fatalf("corrida não terminou em 120 s: %+v", r.Carros)
	}
	if r.Vencedor != 1 {
		t.Fatalf("vencedor = %d, quer o carro mais rápido (1); progresso %+v", r.Vencedor, r.Carros)
	}
	if v := r.Carros[1].Progresso.Voltas; v != 2 {
		t.Errorf("vencedor tem %d voltas, quer 2", v)
	}
	for _, i := range []int{0, 2} {
		if r.Carros[i].Progresso.Voltas >= 2 {
			t.Errorf("carro %d também completou a meta: %+v", i, r.Carros[i].Progresso)
		}
	}
	if got, quer := r.Classificacao(), []int{1, 2, 0}; !slices.Equal(got, quer) {
		t.Errorf("Classificacao = %v, quer %v", got, quer)
	}
	if got := r.Chegada(); len(got) != 1 || got[0] != 1 {
		t.Errorf("Chegada = %v, quer [1]", got)
	}
	// Terminada: Step não faz mais nada (carros congelados).
	tick, carros := r.Tick, slices.Clone(r.Carros)
	for i := 0; i < 30; i++ {
		r.Step()
	}
	if r.Tick != tick || !slices.Equal(r.Carros, carros) {
		t.Errorf("Step depois de Terminada mexeu na corrida")
	}
	t.Logf("2 voltas em %.1f s", r.Tempo())
}

func TestCorridaVoltasUmaVoltaNasLargadasDeTodosOsCarros(t *testing.T) {
	p := carregarOval(t)
	for _, n := range []int{2, 5, 8} {
		ctls := make([]Controlador, n)
		for i := range ctls {
			ctls[i] = &seguidor{p: p, vAlvo: 100 + 15*float64(i)}
		}
		r := novaCorridaComSeguidores(t, p, ModoVoltas, 1, ctls...)
		for tick := 0; tick < 60*60 && !r.Terminada; tick++ {
			r.Step()
		}
		if !r.Terminada || r.Vencedor != n-1 {
			t.Errorf("n=%d: terminada=%v vencedor=%d, quer %d", n, r.Terminada, r.Vencedor, n-1)
		}
	}
}

func TestCorridaVoltasSemVencedorSeTodosMorrem(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	cfg.BaterMata = true
	r, err := NovaCorrida(p, cfg, ModoVoltas, 3, []Controlador{reta{}, reta{}}, -1)
	if err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 60*30 && !r.Terminada; tick++ {
		r.Step()
	}
	if !r.Terminada || r.Vencedor != -1 || len(r.Chegada()) != 0 {
		t.Errorf("terminada=%v vencedor=%d chegada=%v, quer fim sem vencedor", r.Terminada, r.Vencedor, r.Chegada())
	}
	for i, c := range r.Carros {
		if c.Vivo || !c.Batido {
			t.Errorf("carro %d: Vivo=%v Batido=%v, quer morto por parede", i, c.Vivo, c.Batido)
		}
	}
}

func TestCorridaEliminacaoLaserEUltimoVivo(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0,
		&seguidor{p: p, vAlvo: 250},              // 0: foge do laser por mais tempo
		parado{},                                 // 1: nunca sai da largada
		&seguidor{p: p, vAlvo: 60},               // 2: mais lento que o laser (120 px/s)
		&seguidor{p: p, vAlvo: 200, paraEm: 150}, // 3: anda e para antes do laser sair
	)
	vivoAntes := make([]bool, len(r.Carros))
	mortePorTick := make([]int, len(r.Carros))
	for i := range mortePorTick {
		mortePorTick[i] = -1
	}
	for tick := 0; tick < 60*120 && !r.Terminada; tick++ {
		for i := range r.Carros {
			vivoAntes[i] = r.Carros[i].Vivo
		}
		r.Step()
		d, ativo := r.Laser()
		for i := range r.Carros {
			c := &r.Carros[i]
			if vivoAntes[i] && !c.Vivo {
				mortePorTick[i] = r.Tick
				if !ativo || c.Progresso.Distancia > d {
					t.Fatalf("tick %d: carro %d morreu com Distancia %.1f e laser (%.1f, ativo=%v)", r.Tick, i, c.Progresso.Distancia, d, ativo)
				}
			}
			if c.Vivo && ativo && c.Progresso.Distancia <= d {
				t.Fatalf("tick %d: carro %d vivo com Distancia %.1f ≤ laser %.1f", r.Tick, i, c.Progresso.Distancia, d)
			}
			if !vivoAntes[i] && c.Vivo {
				t.Fatalf("tick %d: carro %d ressuscitou", r.Tick, i)
			}
		}
	}
	if !r.Terminada || r.Vencedor != 0 {
		t.Fatalf("terminada=%v vencedor=%d, quer o carro 0 vivo", r.Terminada, r.Vencedor)
	}
	if !r.Carros[0].Vivo {
		t.Errorf("o vencedor não está vivo")
	}
	// O parado (Distancia 0) morre no tick em que o laser sai (3 s = tick 180).
	if mortePorTick[1] != 180 {
		t.Errorf("carro parado morreu no tick %d, quer 180", mortePorTick[1])
	}
	// O que parou no meio da pista morre quando o laser o alcança, depois do parado.
	if mortePorTick[3] <= 180 {
		t.Errorf("carro que parou morreu no tick %d", mortePorTick[3])
	}
	if got, quer := r.Classificacao()[0], 0; got != quer {
		t.Errorf("1º da classificação = %d, quer %d", got, quer)
	}
	// Eliminados: quem morreu mais tarde vem antes.
	ord := r.Classificacao()
	for k := 1; k < len(ord)-1; k++ {
		if mortePorTick[ord[k]] < mortePorTick[ord[k+1]] {
			t.Errorf("classificação %v: carro %d (morte %d) antes de %d (morte %d)", ord, ord[k], mortePorTick[ord[k]], ord[k+1], mortePorTick[ord[k+1]])
		}
	}
	t.Logf("mortes por tick %v, vencedor %d em %.1f s", mortePorTick, r.Vencedor, r.Tempo())
}

// Vivos ficam acima de eliminados na classificação mesmo com menos progresso.
func TestCorridaEliminacaoVivosAcimaDeEliminados(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{}, parado{})
	r.Carros[0].Progresso.Distancia = 900 // mais adiante, mas eliminado
	r.Carros[0].Vivo = false
	r.morte[0] = 10
	r.Carros[1].Progresso.Distancia = 5
	r.Carros[2].Progresso.Distancia = 7
	if got, quer := r.Classificacao(), []int{2, 1, 0}; !slices.Equal(got, quer) {
		t.Errorf("Classificacao = %v, quer %v (vivos por progresso, depois o eliminado)", got, quer)
	}
}

// Dois últimos morrem no mesmo tick: vence quem estava mais adiante.
func TestCorridaEliminacaoMorteJuntaVenceOMaisAdiante(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{}, parado{})
	r.Tick = 400 // o laser já saiu e passa dos três de uma vez
	r.Carros[0].Progresso.Distancia = 30
	r.Carros[1].Progresso.Distancia = 50
	r.Carros[2].Progresso.Distancia = 10
	r.Step()
	if !r.Terminada || r.Vencedor != 1 {
		t.Fatalf("terminada=%v vencedor=%d, quer o carro 1", r.Terminada, r.Vencedor)
	}
	if got, quer := r.Classificacao(), []int{1, 0, 2}; !slices.Equal(got, quer) {
		t.Errorf("Classificacao = %v, quer %v", got, quer)
	}
}

func TestCorridaLaserD6(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0, parado{}, parado{})
	if _, ativo := r.Laser(); ativo {
		t.Fatal("laser ativo no tick 0")
	}
	r.Tick = 179
	if _, ativo := r.Laser(); ativo {
		t.Fatal("laser ativo antes de 3 s")
	}
	r.Tick = 180
	if d, ativo := r.Laser(); !ativo || d != 0 {
		t.Fatalf("aos 3 s o laser devia sair da largada: d=%v ativo=%v", d, ativo)
	}
	r.Tick = 180 + 60 // 1 s depois: 120*1 + 8*1²/2
	if d, _ := r.Laser(); math.Abs(d-124) > 1e-9 {
		t.Errorf("1 s depois d = %v, quer 124", d)
	}
	// A velocidade sobe 8 px/s a cada segundo: entre 10 s e 11 s anda 120+8*10.5.
	r.Tick = 180 + 600
	d10, _ := r.Laser()
	r.Tick = 180 + 660
	d11, _ := r.Laser()
	if math.Abs((d11-d10)-(120+8*10.5)) > 1e-9 {
		t.Errorf("anda %v px entre 10 s e 11 s, quer %v", d11-d10, 120+8*10.5)
	}
	// Posição: na reta de baixo do oval (y = 300) e a faixa atravessa a pista.
	r.Tick = 180 + 60
	pos := r.LaserPos()
	if math.Abs(pos.Y-300) > 1e-6 || math.Abs(pos.X-324) > 1.5 {
		t.Errorf("LaserPos = %+v, quer ≈ (324, 300)", pos)
	}
	f := r.LaserFaixa()
	if l := f.B.Sub(f.A).Len(); math.Abs(l-p.Largura) > 1e-6 {
		t.Errorf("faixa tem %v px, quer a largura da pista %v", l, p.Largura)
	}
	if mid := f.A.Add(f.B).Scale(0.5); mid.Sub(pos).Len() > 1.5 {
		t.Errorf("faixa centrada em %+v, quer perto de %+v", mid, pos)
	}
	// Dá a volta: depois de um comprimento o laser volta ao começo da linha.
	r.Tick = 180
	ini := r.LaserPos()
	if ini.Sub(p.Centro[0]).Len() > 1e-9 {
		t.Errorf("laser sai de %+v, quer Centro[0] = %+v", ini, p.Centro[0])
	}
	// Em Voltas não há laser.
	rv := novaCorridaComSeguidores(t, p, ModoVoltas, 1, parado{}, parado{})
	rv.Tick = 10000
	if _, ativo := rv.Laser(); ativo {
		t.Error("laser ativo em ModoVoltas")
	}
}

// A corrida usa Avancar (com colisão): quem acelera contra a parede é parado
// por ela e nunca sai da pista.
func TestCorridaUsaAvancarComColisao(t *testing.T) {
	p := carregarOval(t)
	r := novaCorridaComSeguidores(t, p, ModoVoltas, 3, reta{}, parado{})
	lo, hi := p.ParedeEsq[0], p.ParedeEsq[0]
	for _, q := range append(slices.Clone(p.ParedeEsq), p.ParedeDir...) {
		lo = Vec2{X: min(lo.X, q.X), Y: min(lo.Y, q.Y)}
		hi = Vec2{X: max(hi.X, q.X), Y: max(hi.Y, q.Y)}
	}
	bateu := false
	for tick := 0; tick < 60*20; tick++ {
		r.Step()
		c := r.Carros[0]
		bateu = bateu || c.Batido
		if c.Pos.X < lo.X || c.Pos.X > hi.X || c.Pos.Y < lo.Y || c.Pos.Y > hi.Y {
			t.Fatalf("tick %d: carro saiu da pista em %+v", r.Tick, c.Pos)
		}
		if !c.Vivo {
			t.Fatal("carro morreu na corrida com BaterMata = false")
		}
	}
	if !bateu {
		t.Error("o carro nunca bateu na parede")
	}
}

func TestCorridaDeterministica(t *testing.T) {
	p := carregarOval(t)
	rodar := func() *Corrida {
		r := novaCorridaComSeguidores(t, p, ModoEliminacao, 0,
			&seguidor{p: p, vAlvo: 180}, &seguidor{p: p, vAlvo: 140}, parado{})
		for i := 0; i < 60*30; i++ {
			r.Step()
		}
		return r
	}
	a, b := rodar(), rodar()
	if a.Tick != b.Tick || a.Vencedor != b.Vencedor || !slices.Equal(a.Carros, b.Carros) {
		t.Errorf("duas corridas iguais deram resultados diferentes")
	}
}

func TestNovaCorridaErros(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	um := []Controlador{parado{}, parado{}}
	casos := []struct {
		nome string
		p    *Pista
		cfg  Config
		modo Modo
		v    int
		ctls []Controlador
	}{
		{"pista nil", nil, cfg, ModoVoltas, 1, um},
		{"sem controladores", p, cfg, ModoVoltas, 1, nil},
		{"zero voltas", p, cfg, ModoVoltas, 0, um},
		{"modo inválido", p, cfg, Modo(9), 1, um},
		{"DT zero", p, Config{}, ModoVoltas, 1, um},
		{"pista vazia", &Pista{}, cfg, ModoVoltas, 1, um},
	}
	for _, c := range casos {
		if r, err := NovaCorrida(c.p, c.cfg, c.modo, c.v, c.ctls, -1); !errors.Is(err, ErrCorridaInvalida) || r != nil {
			t.Errorf("%s: err = %v, quer ErrCorridaInvalida", c.nome, err)
		}
	}
	// Eliminação ignora as voltas.
	if _, err := NovaCorrida(p, cfg, ModoEliminacao, 0, um, -1); err != nil {
		t.Errorf("Eliminação com voltas 0: %v", err)
	}
	// Controlador nil vale como carro parado.
	r, err := NovaCorrida(p, cfg, ModoVoltas, 1, []Controlador{nil, nil}, -1)
	if err != nil {
		t.Fatal(err)
	}
	r.Step()
}

// rodar600 roda 600 ticks de uma corrida de n carros com seguidores e devolve o
// menor tempo de 3 tentativas (descarta o ruído da máquina).
func rodar600(t *testing.T, p *Pista, n int) time.Duration {
	t.Helper()
	melhor := time.Hour
	for k := 0; k < 3; k++ {
		ctls := make([]Controlador, n)
		for i := range ctls {
			ctls[i] = &seguidor{p: p, vAlvo: 120 + 10*float64(i)}
		}
		r := novaCorridaComSeguidores(t, p, ModoVoltas, 100, ctls...)
		inicio := time.Now()
		for i := 0; i < 600; i++ {
			r.Step()
		}
		if dur := time.Since(inicio); dur < melhor {
			melhor = dur
		}
		if r.Tick != 600 {
			t.Fatalf("Tick = %d, quer 600 (a corrida terminou?)", r.Tick)
		}
	}
	return melhor
}

// 600 ticks sem janela rodam em menos de 50 ms (4 carros, cada um com os 7
// sensores lidos a cada tick). Com 8 carros o tempo só é informado.
func TestCorridaRapida(t *testing.T) {
	p := carregarOval(t)
	t.Logf("600 ticks com 8 carros em %v", rodar600(t, p, 8))
	if dur := rodar600(t, p, 4); dur > 50*time.Millisecond {
		t.Errorf("600 ticks com 4 carros levaram %v, quer < 50 ms", dur)
	} else {
		t.Logf("600 ticks com 4 carros em %v", dur)
	}
}
