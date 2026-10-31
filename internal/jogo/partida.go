package jogo

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// Opcoes são as escolhas do menu para montar uma partida.
type Opcoes struct {
	CaminhoPista string
	DirCerebros  string // pasta com <nome da pista>.json; "" = DirCerebros
	Modo         sim.Modo
	Voltas       int // 1 a 10 (ignorado na Eliminação)
	Carros       int // 2 a 8, contando o jogador
}

// Humano é o índice do jogador em Corrida.Carros (D19).
const Humano = 0

// ControladorHumano é o sim.Controlador do jogador: Corrida.Step chama Decidir
// uma vez por tick e ele devolve o Controle das teclas guardadas por Definir
// (ControleDeTeclas, contrato 1.1).
type ControladorHumano struct{ teclas Teclas }

// Definir guarda as teclas lidas neste tick.
func (h *ControladorHumano) Definir(t Teclas) { h.teclas = t }

// Decidir implementa sim.Controlador.
func (h *ControladorHumano) Decidir(sim.Leitura) sim.Controle { return ControleDeTeclas(h.teclas) }

// Partida é uma corrida com o humano no carro 0 e a IA nos outros. A regra do
// jogo (física, laser, voltas, vencedor) está em sim.Corrida; aqui só se monta.
type Partida struct {
	Corrida *sim.Corrida
	Humano  *ControladorHumano
	Aviso   string // cérebro ausente ou inválido, para mostrar na tela
	Pista   string // nome da pista
}

// NovaPartida carrega a pista (recusa a que não cabe em 1280x720, D17) e o
// cérebro cerebros/<pista>.json, e monta a corrida com sim.NovaCorrida (humano
// = 0, D19). Sem cérebro, ou com cérebro inválido, não falha: os carros da IA
// ficam parados e Aviso diz o porquê (o jogador joga sozinho).
func NovaPartida(o Opcoes) (*Partida, error) {
	if o.CaminhoPista == "" {
		return nil, errors.New("nenhuma pista escolhida")
	}
	if o.Carros < CarrosMin || o.Carros > CarrosMax {
		return nil, fmt.Errorf("carros: %d fora de %d a %d", o.Carros, CarrosMin, CarrosMax)
	}
	if o.Modo == sim.ModoVoltas && (o.Voltas < VoltasMin || o.Voltas > VoltasMax) {
		return nil, fmt.Errorf("voltas: %d fora de %d a %d", o.Voltas, VoltasMin, VoltasMax)
	}
	p, err := sim.CarregarPista(o.CaminhoPista)
	if err != nil {
		return nil, fmt.Errorf("pista %s: %w", o.CaminhoPista, err)
	}
	if !CabeNaTela(p) {
		return nil, fmt.Errorf("pista %s: %w (paredes fora de %dx%d)", o.CaminhoPista, ErrNaoCabe, LarguraTela, AlturaTela)
	}

	nome := nomeDoArquivo(o.CaminhoPista)
	dir := o.DirCerebros
	if dir == "" {
		dir = DirCerebros
	}
	var ctlIA sim.Controlador // nil = carro parado
	aviso := ""
	caminho := filepath.Join(dir, nome+".json")
	if _, err := os.Stat(caminho); err != nil {
		aviso = fmt.Sprintf("sem cérebro para a pista %q (%s): a IA fica parada, você joga sozinho", nome, caminho)
	} else if rede, err := ia.CarregarRede(caminho); err != nil {
		aviso = fmt.Sprintf("cérebro %s inválido (%v): a IA fica parada, você joga sozinho", caminho, err)
	} else {
		ctlIA = ia.NovoControlador(rede) // a rede não guarda estado: um serve a todos
	}

	h := &ControladorHumano{}
	ctls := make([]sim.Controlador, o.Carros)
	for i := range ctls {
		if i == Humano {
			ctls[i] = h
		} else {
			ctls[i] = ctlIA
		}
	}
	cfg := sim.ConfigPadrao()
	cfg.BaterMata = false
	r, err := sim.NovaCorrida(p, cfg, o.Modo, o.Voltas, ctls, Humano)
	if err != nil {
		return nil, err
	}
	return &Partida{Corrida: r, Humano: h, Aviso: aviso, Pista: nome}, nil
}

// Passo roda um tick com as teclas do jogador. Só Corrida.Step chama Decidir.
func (p *Partida) Passo(t Teclas) {
	p.Humano.Definir(t)
	p.Corrida.Step()
}

// AvisoLaser diz quando o laser da Eliminação sai (D6/D19): "" fora da
// Eliminação, com a corrida terminada ou depois que o laser saiu.
func (p *Partida) AvisoLaser() string {
	r := p.Corrida
	if r.Modo != sim.ModoEliminacao || r.Terminada {
		return ""
	}
	if _, ativo := r.Laser(); ativo {
		return "LASER! Ande mais rápido que a faixa vermelha"
	}
	falta := int(math.Ceil(sim.LaserAtraso - r.Tempo()))
	return fmt.Sprintf("Eliminação: o laser sai em %d s. Fique à frente dele!", max(falta, 1))
}

// NomeCarro é o nome de um carro na tela: "Você" ou "IA n".
func NomeCarro(i int) string {
	if i == Humano {
		return "Você"
	}
	return fmt.Sprintf("IA %d", i)
}

// TextoVencedor é a linha de resultado da tela final.
func (p *Partida) TextoVencedor() string {
	r := p.Corrida
	switch {
	case r.Vencedor < 0:
		return "Sem vencedor"
	case r.Vencedor == Humano:
		return "Você venceu!"
	}
	return "Vencedor: " + NomeCarro(r.Vencedor)
}

// Resultado devolve a classificação final: uma linha por carro, do 1º ao último.
func (p *Partida) Resultado() []string {
	r := p.Corrida
	linhas := make([]string, 0, len(r.Carros))
	for pos, i := range r.Classificacao() {
		c := r.Carros[i]
		extra := fmt.Sprintf("%d voltas, %.0f px", c.Progresso.Voltas, c.Progresso.Distancia)
		if r.Modo == sim.ModoEliminacao && !c.Vivo {
			extra += ", eliminado"
		}
		linhas = append(linhas, fmt.Sprintf("%d. %s (%s)", pos+1, NomeCarro(i), extra))
	}
	return linhas
}
