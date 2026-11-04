package jogo

import (
	"testing"

	"vetorial-race/internal/sim"
)

// passarLargada roda os ticks das luzes (D31) sem teclas, até a corrida poder andar.
func passarLargada(p *Partida) {
	for p.Largando() {
		p.Passo(Teclas{})
	}
}

func novaPartidaLargada(t *testing.T, modo sim.Modo) *Partida {
	t.Helper()
	p, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: cerebroOval, Modo: modo, Voltas: 3, Carros: 6})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Até o tick 360 ninguém se move nem o tempo conta, mesmo com a tecla de acelerar;
// no tick seguinte a corrida anda (D31).
func TestLargadaCongelaAteApagar(t *testing.T) {
	p := novaPartidaLargada(t, sim.ModoVoltas)
	r := p.Corrida
	ini := make([]sim.Carro, len(r.Carros))
	copy(ini, r.Carros)
	for i := 0; i < TicksLargada; i++ {
		if !p.Largando() {
			t.Fatalf("passo %d: já deveria ter largado", i)
		}
		p.Passo(Teclas{Cima: true, Direita: true})
		if r.Tick != 0 || r.Tempo() != 0 {
			t.Fatalf("passo %d: Tick=%d Tempo=%v, deveria ser 0", i, r.Tick, r.Tempo())
		}
		for k := range r.Carros {
			if r.Carros[k].Pos != ini[k].Pos || r.Carros[k].Vel != ini[k].Vel || r.Carros[k].Progresso != ini[k].Progresso {
				t.Fatalf("passo %d: carro %d se mexeu durante as luzes", i, k)
			}
		}
	}
	if p.Largando() || p.LuzesAcesas() != 0 {
		t.Fatalf("após %d ticks as luzes deveriam ter apagado", TicksLargada)
	}
	p.Passo(Teclas{Cima: true, Direita: true})
	if r.Tick != 1 {
		t.Fatalf("no tick seguinte a corrida deveria andar: Tick=%d", r.Tick)
	}
	if r.Carros[Humano].Vel == (sim.Vec2{}) {
		t.Error("o humano deveria acelerar depois da largada")
	}
}

// LuzesAcesas: 0 no 1º segundo, +1 a cada segundo até 5, 0 quando apagam.
func TestLuzesAcesasSequencia(t *testing.T) {
	p := novaPartidaLargada(t, sim.ModoVoltas)
	if p.LuzesAcesas() != 0 {
		t.Errorf("antes do 1º passo: %d", p.LuzesAcesas())
	}
	anterior := 0
	vistos := map[int]bool{}
	for i := 1; i <= TicksLargada; i++ {
		p.Passo(Teclas{})
		n := p.LuzesAcesas()
		want := min(i/TicksPorLuz, NumLuzes)
		if i >= TicksLargada {
			want = 0
		}
		if n != want {
			t.Fatalf("passo %d: %d luzes, esperava %d", i, n, want)
		}
		if i < TicksLargada && (n < anterior || n > anterior+1) {
			t.Fatalf("passo %d: %d luzes depois de %d (deve subir uma de cada vez)", i, n, anterior)
		}
		anterior = n
		vistos[n] = true
	}
	for k := 0; k <= NumLuzes; k++ {
		if !vistos[k] {
			t.Errorf("nunca mostrou %d luzes", k)
		}
	}
	// A 5ª luz acende no tick 300 e fica até o 360.
	q := novaPartidaLargada(t, sim.ModoVoltas)
	for i := 0; i < 5*TicksPorLuz; i++ {
		q.Passo(Teclas{})
	}
	if q.LuzesAcesas() != NumLuzes {
		t.Errorf("tick 300: %d luzes", q.LuzesAcesas())
	}
}

// O laser da Eliminação só começa a contar depois das luzes.
func TestLargadaLaserSoDepoisDasLuzes(t *testing.T) {
	p := novaPartidaLargada(t, sim.ModoEliminacao)
	for i := 0; i < TicksLargada; i++ {
		p.Passo(Teclas{})
	}
	if _, ativo := p.Corrida.Laser(); ativo || p.Corrida.Tempo() != 0 {
		t.Fatal("laser/tempo não podem contar durante as luzes")
	}
	for p.Corrida.Tick < 180 {
		p.Passo(Teclas{})
	}
	if _, ativo := p.Corrida.Laser(); !ativo {
		t.Error("o laser deveria sair 3 s depois de apagar as luzes")
	}
}

// Jogar de novo (R) monta outra Partida e as luzes recomeçam; Esc durante as luzes
// volta ao menu (Voltar descarta a partida).
func TestLargadaRecomecaAoJogarDeNovo(t *testing.T) {
	m := NovoMenu([]string{ovalJSON}, "")
	m.DirCerebros = t.TempDir()
	m.Modo = sim.ModoEliminacao
	m.Carros = 3
	a := NovoAppSemJanela(m)
	if err := a.Iniciar(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60*120 && a.Tela == TelaPartida; i++ {
		a.PassoPartida(Teclas{})
	}
	if a.Tela != TelaFim {
		t.Fatalf("deveria terminar (tela %d)", a.Tela)
	}
	if err := a.iniciarCom(a.Opcoes); err != nil {
		t.Fatal(err)
	}
	p := a.Partida
	if !p.Largando() || p.LuzesAcesas() != 0 || p.Corrida.Tick != 0 {
		t.Fatalf("luzes deveriam recomeçar: largando=%v luzes=%d tick=%d", p.Largando(), p.LuzesAcesas(), p.Corrida.Tick)
	}
	a.PassoPartida(Teclas{Cima: true})
	if p.Corrida.Tick != 0 {
		t.Error("a corrida não anda durante as luzes")
	}
	a.Voltar()
	if a.Tela != TelaMenu || a.Partida != nil {
		t.Error("Esc/Voltar durante as luzes deveria ir ao menu")
	}
}
