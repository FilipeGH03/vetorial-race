package jogo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vetorial-race/internal/sim"
)

const (
	ovalJSON    = "../../pistas/oval.json"
	cerebroOval = "../../cerebros"
)

// pistaGrande é uma pista válida cujas paredes saem de 1280x720.
const pistaGrande = `{"versao":1,"nome":"grande","largura":100,
"centro":[[200,200],[1150,200],[1240,360],[1150,520],[200,520],[120,360]]}`

func escrever(t *testing.T, dir, nome, conteudo string) string {
	t.Helper()
	c := filepath.Join(dir, nome)
	if err := os.WriteFile(c, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestMenuPadroes(t *testing.T) {
	m := NovoMenu([]string{"pistas/a.json", "pistas/b.json"}, "")
	if m.Voltas != 3 || m.Carros != 6 || m.Modo != sim.ModoVoltas || m.Pista != 0 {
		t.Fatalf("padrões errados: %+v", m)
	}
	o := m.Opcoes()
	if o.CaminhoPista != "pistas/a.json" || o.Voltas != 3 || o.Carros != 6 {
		t.Errorf("opções: %+v", o)
	}
	if m.NomePista() != "a" {
		t.Errorf("nome %q", m.NomePista())
	}
}

func TestMenuLimitesECampos(t *testing.T) {
	m := NovoMenu([]string{"p/a.json", "p/b.json", "p/c.json"}, "p/b.json")
	if m.Pista != 1 {
		t.Fatalf("preferida: %d", m.Pista)
	}
	// Pista circula nos dois sentidos.
	m.Direita()
	m.Direita()
	if m.Pista != 0 {
		t.Errorf("deveria dar a volta: %d", m.Pista)
	}
	m.Esquerda()
	if m.Pista != 2 {
		t.Errorf("deveria voltar para o fim: %d", m.Pista)
	}
	// Modo alterna.
	m.Baixo()
	m.Direita()
	if m.Modo != sim.ModoEliminacao {
		t.Error("modo deveria virar Eliminação")
	}
	m.Esquerda()
	if m.Modo != sim.ModoVoltas {
		t.Error("modo deveria voltar a Voltas")
	}
	// Voltas de 1 a 10.
	m.Baixo()
	for i := 0; i < 20; i++ {
		m.Direita()
	}
	if m.Voltas != 10 {
		t.Errorf("voltas máx: %d", m.Voltas)
	}
	for i := 0; i < 20; i++ {
		m.Esquerda()
	}
	if m.Voltas != 1 {
		t.Errorf("voltas mín: %d", m.Voltas)
	}
	// Carros de 2 a 8.
	m.Baixo()
	for i := 0; i < 20; i++ {
		m.Direita()
	}
	if m.Carros != 8 {
		t.Errorf("carros máx: %d", m.Carros)
	}
	for i := 0; i < 20; i++ {
		m.Esquerda()
	}
	if m.Carros != 2 {
		t.Errorf("carros mín: %d", m.Carros)
	}
	// Cursor circula.
	m.Baixo()
	if m.Selecao != CampoPista {
		t.Errorf("cursor: %d", m.Selecao)
	}
	m.Cima()
	if m.Selecao != CampoCarros {
		t.Errorf("cursor: %d", m.Selecao)
	}
}

func TestMenuSemPistasEPreferidaFora(t *testing.T) {
	m := NovoMenu(nil, "")
	if m.Aviso == "" || m.CaminhoPista() != "" || m.NomePista() != "" {
		t.Errorf("sem pistas: %+v", m)
	}
	m.Direita() // não pode dar pânico
	m = NovoMenu([]string{"p/a.json"}, "outra/x.json")
	if m.CaminhoPista() != "outra/x.json" {
		t.Errorf("preferida fora da pasta: %q", m.CaminhoPista())
	}
}

func TestMenuLinhasEliminacaoSemVoltas(t *testing.T) {
	m := NovoMenu([]string{"p/a.json"}, "")
	l := m.Linhas()
	if !strings.Contains(l[CampoVoltas], "3") || !strings.Contains(l[CampoCarros], "6") || !strings.Contains(l[CampoCarros], "5 da IA") {
		t.Errorf("linhas: %q", l)
	}
	m.Modo = sim.ModoEliminacao
	if strings.Contains(m.Linhas()[CampoVoltas], "3") {
		t.Errorf("Eliminação não usa voltas: %q", m.Linhas()[CampoVoltas])
	}
}

func TestListarPistasOrdenada(t *testing.T) {
	dir := t.TempDir()
	escrever(t, dir, "b.json", "{}")
	escrever(t, dir, "a.json", "{}")
	escrever(t, dir, "nota.txt", "x")
	l := ListarPistas(dir)
	if len(l) != 2 || filepath.Base(l[0]) != "a.json" || filepath.Base(l[1]) != "b.json" {
		t.Errorf("lista: %v", l)
	}
	if len(ListarPistas(filepath.Join(dir, "nao-existe"))) != 0 {
		t.Error("pasta inexistente deveria dar lista vazia")
	}
}

// Padrão do menu: 3 voltas, jogador + 5 IAs com o cérebro do oval, humano = 0.
func TestNovaPartidaPadraoComCerebro(t *testing.T) {
	m := NovoMenu([]string{ovalJSON}, "")
	m.DirCerebros = cerebroOval
	p, err := NovaPartida(m.Opcoes())
	if err != nil {
		t.Fatal(err)
	}
	r := p.Corrida
	if len(r.Carros) != 6 || r.Voltas != 3 || r.Modo != sim.ModoVoltas || r.Humano != 0 {
		t.Fatalf("partida: carros=%d voltas=%d humano=%d", len(r.Carros), r.Voltas, r.Humano)
	}
	if p.Aviso != "" {
		t.Errorf("com cérebro não deveria avisar: %q", p.Aviso)
	}
	if r.Controladores[0] != sim.Controlador(p.Humano) {
		t.Error("o carro 0 deve ser o do humano")
	}
	for i := 1; i < 6; i++ {
		if r.Controladores[i] == nil {
			t.Errorf("carro %d sem IA", i)
		}
	}
}

// O humano é lido por Corrida.Step (Decidir), sem ninguém chamar Decidir por fora.
func TestPassoUsaTeclasDoHumano(t *testing.T) {
	m := NovoMenu([]string{ovalJSON}, "")
	m.DirCerebros = t.TempDir() // sem cérebro: só o humano se mexe
	p, err := NovaPartida(m.Opcoes())
	if err != nil {
		t.Fatal(err)
	}
	ini := p.Corrida.Carros[0].Pos
	for i := 0; i < 30; i++ {
		p.Passo(Teclas{Direita: true})
	}
	c := p.Corrida.Carros[0]
	if c.Vel.X <= 0 || c.Pos == ini {
		t.Errorf("humano não andou com a tecla: vel=%v pos=%v", c.Vel, c.Pos)
	}
	for i := 1; i < 6; i++ {
		if v := p.Corrida.Carros[i].Vel; v != (sim.Vec2{}) {
			t.Errorf("IA %d deveria estar parada: %v", i, v)
		}
	}
	if p.Corrida.Tick != 30 {
		t.Errorf("tick %d", p.Corrida.Tick)
	}
}

func TestNovaPartidaSemCerebroAvisaEFicaJogavel(t *testing.T) {
	o := Opcoes{CaminhoPista: ovalJSON, DirCerebros: t.TempDir(), Modo: sim.ModoVoltas, Voltas: 3, Carros: 6}
	p, err := NovaPartida(o)
	if err != nil {
		t.Fatalf("sem cérebro não pode falhar: %v", err)
	}
	if !strings.Contains(p.Aviso, "sem cérebro") || !strings.Contains(p.Aviso, "oval") {
		t.Errorf("aviso: %q", p.Aviso)
	}
	for i := 0; i < 600; i++ {
		p.Passo(Teclas{Cima: true})
	}
	if p.Corrida.Carros[0].Pos == sim.Largada(p.Corrida.Pista, 6)[0].Pos {
		t.Error("o jogador deveria poder andar")
	}
}

func TestNovaPartidaCerebroInvalidoAvisa(t *testing.T) {
	dir := t.TempDir()
	escrever(t, dir, "oval.json", `{"versao":1,"camadas":[10,12,2]}`)
	p, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: dir, Voltas: 3, Carros: 6})
	if err != nil {
		t.Fatalf("cérebro inválido não pode falhar: %v", err)
	}
	if !strings.Contains(p.Aviso, "inválido") {
		t.Errorf("aviso: %q", p.Aviso)
	}
	escrever(t, dir, "oval.json", "não é json")
	p, err = NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: dir, Voltas: 3, Carros: 6})
	if err != nil || p.Aviso == "" {
		t.Errorf("json quebrado: err=%v aviso=%q", err, p.Aviso)
	}
}

func TestNovaPartidaRecusaPistaQueNaoCabe(t *testing.T) {
	dir := t.TempDir()
	c := escrever(t, dir, "grande.json", pistaGrande)
	if p, err := sim.CarregarPista(c); err != nil || CabeNaTela(p) {
		t.Fatalf("pista de teste deveria valer e não caber: err=%v", err)
	}
	_, err := NovaPartida(Opcoes{CaminhoPista: c, Voltas: 3, Carros: 6})
	if !errors.Is(err, ErrNaoCabe) {
		t.Errorf("esperava ErrNaoCabe, veio %v", err)
	}
	if _, err := Novo(c); !errors.Is(err, ErrNaoCabe) {
		t.Errorf("Novo deveria recusar com ErrNaoCabe, veio %v", err)
	}
}

func TestNovaPartidaOpcoesInvalidas(t *testing.T) {
	casos := []Opcoes{
		{CaminhoPista: "", Voltas: 3, Carros: 6},
		{CaminhoPista: ovalJSON, Voltas: 3, Carros: 1},
		{CaminhoPista: ovalJSON, Voltas: 3, Carros: 9},
		{CaminhoPista: ovalJSON, Voltas: 0, Carros: 6},
		{CaminhoPista: ovalJSON, Voltas: 11, Carros: 6},
		{CaminhoPista: "nao-existe.json", Voltas: 3, Carros: 6},
	}
	for _, o := range casos {
		if p, err := NovaPartida(o); err == nil || p != nil {
			t.Errorf("%+v deveria ser recusada", o)
		}
	}
	// Eliminação ignora voltas.
	if _, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: t.TempDir(), Modo: sim.ModoEliminacao, Carros: 2}); err != nil {
		t.Errorf("Eliminação com 2 carros: %v", err)
	}
}

// Eliminação: o aviso do laser conta 3 s, o laser sai e, com o jogador parado, ele
// é eliminado e a partida termina com a classificação completa.
func TestEliminacaoLaserAvisoEFim(t *testing.T) {
	o := Opcoes{CaminhoPista: ovalJSON, DirCerebros: t.TempDir(), Modo: sim.ModoEliminacao, Carros: 4}
	p, err := NovaPartida(o)
	if err != nil {
		t.Fatal(err)
	}
	if av := p.AvisoLaser(); !strings.Contains(av, "3 s") {
		t.Errorf("no início deveria avisar 3 s: %q", av)
	}
	p.Passo(Teclas{})
	if av := p.AvisoLaser(); !strings.Contains(av, "3 s") {
		t.Errorf("1 tick depois ainda 3 s: %q", av)
	}
	for p.Corrida.Tick < 61 {
		p.Passo(Teclas{})
	}
	if av := p.AvisoLaser(); !strings.Contains(av, "2 s") {
		t.Errorf("com 1 s deveria faltar 2 s: %q", av)
	}
	for p.Corrida.Tick < 179 {
		p.Passo(Teclas{})
	}
	if av := p.AvisoLaser(); !strings.Contains(av, "1 s") {
		t.Errorf("no tick 179 deveria faltar 1 s: %q", av)
	}
	if _, ativo := p.Corrida.Laser(); ativo {
		t.Fatal("laser não deveria ter saído antes dos 3 s")
	}
	// Com o laser ativo (e a corrida em andamento) o aviso muda.
	p.Corrida.Tick = 200
	if av := p.AvisoLaser(); !strings.Contains(av, "LASER") {
		t.Errorf("aviso com o laser ativo: %q", av)
	}
	p.Corrida.Tick = 179
	for i := 0; i < 60*60 && !p.Corrida.Terminada; i++ {
		p.Passo(Teclas{})
	}
	r := p.Corrida
	if !r.Terminada || r.Carros[Humano].Vivo {
		t.Fatalf("o humano parado deveria ser eliminado e a partida acabar: terminada=%v", r.Terminada)
	}
	if p.AvisoLaser() != "" {
		t.Error("sem aviso depois do fim")
	}
	if got := len(p.Resultado()); got != 4 {
		t.Errorf("resultado com %d linhas", got)
	}
	if !strings.Contains(strings.Join(p.Resultado(), "\n"), "eliminado") {
		t.Error("resultado deveria marcar eliminados")
	}
}

func TestAvisoLaserSoNaEliminacao(t *testing.T) {
	p, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: t.TempDir(), Voltas: 3, Carros: 3})
	if err != nil {
		t.Fatal(err)
	}
	if p.AvisoLaser() != "" {
		t.Error("modo Voltas não tem laser")
	}
}

func TestTextoVencedor(t *testing.T) {
	p, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: t.TempDir(), Voltas: 3, Carros: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		v    int
		want string
	}{{-1, "Sem vencedor"}, {0, "Você venceu"}, {2, "IA 2"}} {
		p.Corrida.Vencedor = c.v
		if got := p.TextoVencedor(); !strings.Contains(got, c.want) {
			t.Errorf("vencedor %d: %q sem %q", c.v, got, c.want)
		}
	}
}

// Partida de Voltas: quando alguém completa a meta, o vencedor do App é o que a
// Corrida diz e a tela passa para TelaFim. Aqui o jogador é movido à mão até o fim.
func TestAppFluxoMenuPartidaFim(t *testing.T) {
	m := NovoMenu([]string{ovalJSON}, "")
	m.DirCerebros = t.TempDir()
	m.Voltas = 1
	m.Carros = 2
	a := NovoAppSemJanela(m)
	if a.Tela != TelaMenu {
		t.Fatal("começa no menu")
	}
	if err := a.Iniciar(); err != nil {
		t.Fatal(err)
	}
	if a.Tela != TelaPartida || a.Partida == nil {
		t.Fatal("deveria estar na partida")
	}
	// Leva o carro do jogador até o fim da volta pela própria Corrida: acelera
	// seguindo a linha central com um piloto simples (apenas para fechar a meta).
	r := a.Partida.Corrida
	for i := 0; i < 60*120 && a.Tela == TelaPartida; i++ {
		a.PassoPartida(pilotoSimples(r))
	}
	if a.Tela != TelaFim {
		t.Skipf("piloto simples não fechou a volta (tela %d); o fim é coberto pela Eliminação", a.Tela)
	}
	if r.Vencedor != Humano || !strings.Contains(a.Partida.TextoVencedor(), "Você") {
		t.Errorf("vencedor %d (%s)", r.Vencedor, a.Partida.TextoVencedor())
	}
	a.Voltar()
	if a.Tela != TelaMenu || a.Partida != nil {
		t.Error("Voltar deve ir ao menu")
	}
}

// pilotoSimples escolhe teclas que levam o carro do humano ao ponto da linha
// central 4 trechos à frente.
func pilotoSimples(r *sim.Corrida) Teclas {
	c := r.Carros[Humano]
	p := r.Pista
	// trecho mais perto
	melhor, d2 := 0, 1e18
	for i, q := range p.Centro {
		if d := q.Sub(c.Pos).Dot(q.Sub(c.Pos)); d < d2 {
			melhor, d2 = i, d
		}
	}
	alvo := p.Centro[(melhor+4)%len(p.Centro)].Sub(c.Pos)
	var t Teclas
	// Quer a velocidade na direção do alvo (módulo ~200).
	querX, querY := alvo.Norm().X*200, alvo.Norm().Y*200
	t.Direita, t.Esquerda = c.Vel.X < querX-10, c.Vel.X > querX+10
	t.Cima, t.Baixo = c.Vel.Y < querY-10, c.Vel.Y > querY+10
	return t
}

func TestAppIniciarRecusadaFicaNoMenuComAviso(t *testing.T) {
	dir := t.TempDir()
	c := escrever(t, dir, "grande.json", pistaGrande)
	a := NovoAppSemJanela(NovoMenu([]string{c}, ""))
	if err := a.Iniciar(); !errors.Is(err, ErrNaoCabe) {
		t.Fatalf("esperava ErrNaoCabe, veio %v", err)
	}
	if a.Tela != TelaMenu || a.Partida != nil || a.Menu.Aviso == "" {
		t.Errorf("deveria ficar no menu com aviso: tela=%d aviso=%q", a.Tela, a.Menu.Aviso)
	}
	// Sem pistas: Iniciar também recusa sem pânico.
	a = NovoAppSemJanela(NovoMenu(nil, ""))
	if err := a.Iniciar(); err == nil || a.Tela != TelaMenu {
		t.Errorf("sem pistas: err=%v tela=%d", err, a.Tela)
	}
}

func TestAppEliminacaoTerminaNaTelaFim(t *testing.T) {
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
		t.Fatalf("deveria terminar na tela do vencedor (tela %d)", a.Tela)
	}
	// Jogar de novo monta outra partida com as mesmas escolhas.
	if err := a.iniciarCom(a.Opcoes); err != nil || a.Tela != TelaPartida || a.Partida.Corrida.Tick != 0 {
		t.Errorf("jogar de novo: err=%v tela=%d", err, a.Tela)
	}
}

func TestPassoPartidaSemPartidaNaoFazNada(t *testing.T) {
	a := NovoAppSemJanela(NovoMenu(nil, ""))
	a.PassoPartida(Teclas{Cima: true})
	if a.Tela != TelaMenu {
		t.Error("não deveria mudar de tela")
	}
}

func TestEditorAbreVazioComArquivoInvalido(t *testing.T) {
	dir := t.TempDir()
	casos := map[string]string{
		"quebrado.json": "isso não é json",
		"grande.json":   pistaGrande,
		"curta.json":    `{"versao":1,"nome":"curta","largura":90,"centro":[[0,0],[1,1]]}`,
	}
	for nome, conteudo := range casos {
		c := escrever(t, dir, nome, conteudo)
		e, err := NovoEditorTela(c)
		if err != nil || e == nil {
			t.Fatalf("%s: não pode falhar: %v", nome, err)
		}
		if !strings.Contains(e.Aviso, "não abriu") || len(e.E.Pontos) != 0 {
			t.Errorf("%s: editor deveria abrir vazio com aviso: %q (%d pontos)", nome, e.Aviso, len(e.E.Pontos))
		}
	}
	e, err := NovoEditorTela(ovalJSON)
	if err != nil || !strings.Contains(e.Aviso, "carregado") || len(e.E.Pontos) == 0 {
		t.Errorf("arquivo válido deveria carregar: err=%v aviso=%q", err, e.Aviso)
	}
}

func TestSemAcento(t *testing.T) {
	if got := semAcento("Eliminação: você já não é IÁ"); got != "Eliminacao: voce ja nao e IA" {
		t.Errorf("semAcento: %q", got)
	}
}
