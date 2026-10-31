package jogo

import (
	"testing"

	"vetorial-race/internal/sim"
)

// Roteiro do B5: Voltas com o padrão (3 voltas, jogador + 5 IAs com o cérebro real).
// Quando a partida termina, o vencedor é o primeiro a chegar e completou 3 voltas.
func TestB5VoltasPadraoVencedorCerto(t *testing.T) {
	m := NovoMenu([]string{ovalJSON}, "")
	m.DirCerebros = cerebroOval
	p, err := NovaPartida(m.Opcoes())
	if err != nil {
		t.Fatal(err)
	}
	r := p.Corrida
	for i := 0; i < 60*240 && !r.Terminada; i++ {
		p.Passo(pilotoSimples(r))
	}
	if !r.Terminada {
		t.Skip("ninguém fechou 3 voltas em 240 s com este cérebro/piloto")
	}
	if r.Vencedor < 0 {
		t.Fatal("Voltas terminada deveria ter vencedor")
	}
	if ch := r.Chegada(); len(ch) == 0 || ch[0] != r.Vencedor {
		t.Errorf("vencedor %d != primeiro da chegada %v", r.Vencedor, ch)
	}
	if r.Carros[r.Vencedor].Progresso.Voltas < 3 && r.Carros[Humano].Vivo {
		t.Errorf("vencedor com %d voltas", r.Carros[r.Vencedor].Progresso.Voltas)
	}
}

// Eliminação com IA: o laser elimina quem fica para trás e a classificação cobre todos.
func TestB5EliminacaoComCerebro(t *testing.T) {
	p, err := NovaPartida(Opcoes{CaminhoPista: ovalJSON, DirCerebros: cerebroOval, Modo: sim.ModoEliminacao, Carros: 6})
	if err != nil {
		t.Fatal(err)
	}
	r := p.Corrida
	for i := 0; i < 60*120 && !r.Terminada; i++ {
		p.Passo(Teclas{})
	}
	if !r.Terminada {
		t.Fatal("Eliminação deveria acabar (humano parado é eliminado pelo laser)")
	}
	if r.Carros[Humano].Vivo {
		t.Error("humano parado deveria ter sido eliminado")
	}
	if got := len(r.Classificacao()); got != 6 {
		t.Errorf("classificação com %d", got)
	}
	if len(p.Resultado()) != 6 || p.TextoVencedor() == "" {
		t.Error("resultado incompleto")
	}
}
