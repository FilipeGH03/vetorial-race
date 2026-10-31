package jogo

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"vetorial-race/internal/sim"
)

// Pastas padrão, relativas ao diretório de trabalho.
const (
	DirPistas   = "pistas"
	DirCerebros = "cerebros"
)

// Padrões e limites da partida (D10): 3 voltas (1 a 10) e 6 carros contando o
// jogador (2 a 8, o mesmo limite de sim.NovaCorrida, D19).
const (
	VoltasPadrao = 3
	VoltasMin    = 1
	VoltasMax    = 10
	CarrosPadrao = 6
	CarrosMin    = 2
	CarrosMax    = 8
)

// Campo é a linha do menu em que o cursor está.
type Campo int8

const (
	CampoPista Campo = iota
	CampoModo
	CampoVoltas
	CampoCarros
	numCampos
)

// Menu é o estado da tela inicial (sem janela, para poder ser testado): a pista,
// o modo, o número de voltas e o de carros, e em qual linha está o cursor.
type Menu struct {
	Pistas      []string // caminhos das pistas (pistas/*.json), em ordem alfabética
	Pista       int      // índice em Pistas
	Modo        sim.Modo
	Voltas      int
	Carros      int // contando o jogador
	Selecao     Campo
	Aviso       string // última mensagem (pista recusada, sem pistas)
	DirCerebros string // pasta dos cérebros
}

// ListarPistas devolve os arquivos .json de dir, em ordem alfabética.
func ListarPistas(dir string) []string {
	arqs, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(arqs)
	return arqs
}

// NovoMenu cria o menu com os padrões da D10 sobre as pistas dadas. Se
// preferida for uma das pistas (mesmo caminho), ela já vem escolhida.
func NovoMenu(pistas []string, preferida string) *Menu {
	m := &Menu{
		Pistas:      pistas,
		Modo:        sim.ModoVoltas,
		Voltas:      VoltasPadrao,
		Carros:      CarrosPadrao,
		DirCerebros: DirCerebros,
	}
	if preferida != "" {
		m.Pista = -1
		for i, p := range pistas {
			if filepath.Clean(p) == filepath.Clean(preferida) {
				m.Pista = i
			}
		}
		if m.Pista < 0 { // fora da pasta de pistas: entra na lista mesmo assim
			m.Pistas = append(append([]string(nil), pistas...), preferida)
			m.Pista = len(pistas)
		}
	}
	if len(pistas) == 0 {
		m.Aviso = "nenhuma pista em " + DirPistas + "/: desenhe uma com -editar"
	}
	return m
}

// Cima e Baixo movem o cursor entre as linhas (com volta ao fim).
func (m *Menu) Cima()  { m.Selecao = (m.Selecao + numCampos - 1) % numCampos }
func (m *Menu) Baixo() { m.Selecao = (m.Selecao + 1) % numCampos }

// Esquerda e Direita mudam o valor da linha selecionada.
func (m *Menu) Esquerda() { m.mudar(-1) }
func (m *Menu) Direita()  { m.mudar(1) }

func (m *Menu) mudar(d int) {
	switch m.Selecao {
	case CampoPista:
		if n := len(m.Pistas); n > 0 {
			m.Pista = (m.Pista + d + n) % n
		}
	case CampoModo:
		if m.Modo == sim.ModoVoltas {
			m.Modo = sim.ModoEliminacao
		} else {
			m.Modo = sim.ModoVoltas
		}
	case CampoVoltas:
		m.Voltas = min(VoltasMax, max(VoltasMin, m.Voltas+d))
	case CampoCarros:
		m.Carros = min(CarrosMax, max(CarrosMin, m.Carros+d))
	}
	m.Aviso = ""
}

// CaminhoPista é a pista escolhida ("" se não há nenhuma).
func (m *Menu) CaminhoPista() string {
	if m.Pista < 0 || m.Pista >= len(m.Pistas) {
		return ""
	}
	return m.Pistas[m.Pista]
}

// NomePista é o nome da pista escolhida (o do arquivo, sem a extensão).
func (m *Menu) NomePista() string { return nomeDoArquivo(m.CaminhoPista()) }

// Opcoes devolve as escolhas do menu para NovaPartida.
func (m *Menu) Opcoes() Opcoes {
	return Opcoes{
		CaminhoPista: m.CaminhoPista(),
		DirCerebros:  m.DirCerebros,
		Modo:         m.Modo,
		Voltas:       m.Voltas,
		Carros:       m.Carros,
	}
}

// Linhas é o texto de cada linha do menu, na ordem dos Campo.
func (m *Menu) Linhas() [numCampos]string {
	pista := m.NomePista()
	if pista == "" {
		pista = "(nenhuma)"
	}
	modo, voltas := "Voltas", fmt.Sprint(m.Voltas)
	if m.Modo == sim.ModoEliminacao {
		modo, voltas = "Eliminação (laser)", "-"
	}
	return [numCampos]string{
		"Pista:   " + pista,
		"Modo:    " + modo,
		"Voltas:  " + voltas,
		fmt.Sprintf("Carros:  %d (você + %d da IA)", m.Carros, m.Carros-1),
	}
}

func nomeDoArquivo(caminho string) string {
	if caminho == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(caminho), filepath.Ext(caminho))
}
