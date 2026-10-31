package jogo

import (
	"errors"
	"fmt"
	"path/filepath"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// Treino visual (R14): a mesma neuroevolução de cmd/treino (D5, D21, D22), só
// que a população inteira dirige ao mesmo tempo na tela. Cada geração tem
// ia.NLargadas rodadas; na rodada k os 50 indivíduos largam de Treino.Largada(k)
// e dirigem como em ia.Treino.RodarDe (mesmo tick, mesmo fim por tempo ou por
// TicksParado sem avançar). Quando a última rodada termina, a nota de cada
// indivíduo é a média das rodadas, igual a ia.Treino.Rodar, e nasce a geração
// seguinte. Não há regra de jogo aqui: física, colisão e progresso são de
// internal/sim, nota e evolução são de internal/ia.

// Velocidades são os multiplicadores de ticks por quadro (1x = 60 ticks/s).
var Velocidades = []int{1, 2, 4, 8, 16, 32}

// VelocidadePadrao é o índice em Velocidades com que a tela abre (4x).
const VelocidadePadrao = 2

// PontoGeracao é o resumo de uma geração já avaliada.
type PontoGeracao struct {
	Geracao int
	Melhor  float64 // maior nota da geração
	Media   float64 // média das notas da geração
}

// Individuo é uma rede dirigindo um carro numa rodada.
type Individuo struct {
	Carro sim.Carro
	Res   ia.Resultado // vale por inteiro quando Fim
	Fim   bool         // encerrado (tempo, ou TicksParado sem avançar)

	ctl    *ia.Controlador
	ultima float64 // maior Distancia até agora
	parado int     // ticks seguidos sem a Distancia aumentar
}

// TreinoVisual é o estado do treino na tela, sem janela (testável).
type TreinoVisual struct {
	Pista       *sim.Pista
	Nome        string
	Treino      *ia.Treino
	Pop         *ia.Populacao
	DirCerebros string

	Individuos []Individuo // um por rede de Pop.Redes, na rodada atual
	Largada    int         // rodada atual, 0 <= Largada < ia.NLargadas
	Tick       int         // ticks desde o começo da rodada
	Historico  []PontoGeracao

	Melhor        *ia.Rede // melhor rede de todas as gerações (cópia), nil até a 1ª geração terminar
	MelhorNota    float64
	MelhorGeracao int // geração em que Melhor nasceu

	Pausado   bool
	VelIndice int    // índice em Velocidades
	Aviso     string // último resultado do botão salvar
	Erro      error  // falha da evolução: o treino para

	avals []ia.Avaliacao // avals[i] vai sendo preenchida rodada a rodada
}

// NovoTreinoVisual carrega a pista (recusa a que não cabe em 1280x720, D17) e
// começa a geração 0 com os parâmetros da D5 e a seed dada. dirCerebros é a
// pasta onde Salvar grava ("" = DirCerebros).
func NovoTreinoVisual(caminhoPista, dirCerebros string, seed uint64) (*TreinoVisual, error) {
	p, err := sim.CarregarPista(caminhoPista)
	if err != nil {
		return nil, fmt.Errorf("pista %s: %w", caminhoPista, err)
	}
	if !CabeNaTela(p) {
		return nil, fmt.Errorf("pista %s: %w (paredes fora de %dx%d)", caminhoPista, ErrNaoCabe, LarguraTela, AlturaTela)
	}
	t, err := ia.NovoTreino(p)
	if err != nil {
		return nil, err
	}
	params := ia.ParamsPadrao()
	params.Seed = seed
	pop, err := ia.NovaPopulacao(params)
	if err != nil {
		return nil, err
	}
	if dirCerebros == "" {
		dirCerebros = DirCerebros
	}
	v := &TreinoVisual{
		Pista:       p,
		Nome:        nomeDoArquivo(caminhoPista),
		Treino:      t,
		Pop:         pop,
		DirCerebros: dirCerebros,
		VelIndice:   VelocidadePadrao,
	}
	v.novaGeracao()
	return v, nil
}

// novaGeracao zera as notas parciais e começa a rodada 0 da geração atual.
func (v *TreinoVisual) novaGeracao() {
	v.avals = make([]ia.Avaliacao, len(v.Pop.Redes))
	v.iniciarLargada(0)
}

// iniciarLargada põe todas as redes na posição k de largada.
func (v *TreinoVisual) iniciarLargada(k int) {
	v.Largada, v.Tick = k, 0
	v.Individuos = make([]Individuo, len(v.Pop.Redes))
	for i, r := range v.Pop.Redes {
		c := v.Treino.Largada(k)
		v.Individuos[i] = Individuo{Carro: c, ctl: ia.NovoControlador(r), ultima: c.Progresso.Distancia}
	}
}

// Geracao é a geração em andamento (0 é a população inicial).
func (v *TreinoVisual) Geracao() int { return v.Pop.Geracao }

// Passo roda um tick para todos os indivíduos que ainda dirigem. Nada de
// relógio de parede: o resultado só depende das redes e da pista. Ao terminar
// o último indivíduo da rodada, passa para a seguinte (ou fecha a geração).
func (v *TreinoVisual) Passo() {
	if v.Erro != nil {
		return
	}
	todos := true
	for i := range v.Individuos {
		ind := &v.Individuos[i]
		if ind.Fim {
			continue
		}
		v.passoIndividuo(ind)
		if !ind.Fim {
			todos = false
		}
	}
	v.Tick++
	if todos {
		v.fecharLargada()
	}
}

// passoIndividuo é um tick de ia.Treino.RodarDe: Ler, Decidir (uma vez) e
// Avancar; cada tick com Batido conta uma batida.
func (v *TreinoVisual) passoIndividuo(ind *Individuo) {
	t := v.Treino
	c, res := &ind.Carro, &ind.Res
	c.Avancar(ind.ctl.Decidir(sim.Ler(c, t.Pista, t.Cfg)), t.Pista, t.Cfg)
	res.Ticks++
	if c.Batido {
		res.Batidas++
	}
	if res.TempoVolta == 0 && c.Progresso.Voltas >= 1 {
		res.TempoVolta = float64(res.Ticks) * t.Cfg.DT
	}
	if c.Progresso.Distancia > ind.ultima {
		ind.ultima, ind.parado = c.Progresso.Distancia, 0
	} else {
		ind.parado++
		if ind.parado >= ia.TicksParado {
			res.Parou = true
			ind.Fim = true
		}
	}
	if res.Ticks >= t.MaxTicks || !c.Vivo {
		ind.Fim = true
	}
	res.Distancia, res.Voltas = c.Progresso.Distancia, c.Progresso.Voltas
	if ind.Fim {
		res.Pontuacao = ia.Pontuacao(res.Distancia, res.Voltas, res.TempoVolta, res.Batidas)
	}
}

// fecharLargada guarda o resultado da rodada e começa a próxima; depois da
// última, fecha a geração.
func (v *TreinoVisual) fecharLargada() {
	for i := range v.Individuos {
		v.avals[i].Largadas[v.Largada] = v.Individuos[i].Res
	}
	if v.Largada+1 < ia.NLargadas {
		v.iniciarLargada(v.Largada + 1)
		return
	}
	v.fecharGeracao()
}

// fecharGeracao calcula a nota de cada rede (média das rodadas, mesma ordem de
// soma de ia.Treino.AvaliarPopulacao), registra o resumo, guarda o melhor
// cérebro e gera a população seguinte.
func (v *TreinoVisual) fecharGeracao() {
	notas := make([]float64, len(v.avals))
	soma := 0.0
	for i := range v.avals {
		s := 0.0
		for _, r := range v.avals[i].Largadas {
			s += r.Pontuacao
		}
		v.avals[i].Pontuacao = s / ia.NLargadas
		notas[i] = v.avals[i].Pontuacao
		soma += notas[i]
	}
	if err := v.Pop.Definir(notas); err != nil {
		v.Erro = err
		return
	}
	r, nota := v.Pop.Melhor()
	v.Historico = append(v.Historico, PontoGeracao{Geracao: v.Pop.Geracao, Melhor: nota, Media: soma / float64(len(notas))})
	if v.Melhor == nil || nota > v.MelhorNota {
		v.Melhor = r.Copia()
		v.Melhor.Pista = v.Nome
		v.MelhorNota, v.MelhorGeracao = nota, r.Geracao
	}
	if err := v.Pop.Proxima(); err != nil {
		v.Erro = err
		return
	}
	v.novaGeracao()
}

// ErroSemMelhor é devolvido por Salvar antes de uma geração terminar.
var ErroSemMelhor = errors.New("nenhuma geração terminou ainda: não há melhor cérebro")

// CaminhoCerebro é o arquivo em que Salvar grava: <DirCerebros>/<pista>.json
// (contrato 1.4, o nome que o jogo procura).
func (v *TreinoVisual) CaminhoCerebro() string {
	return filepath.Join(v.DirCerebros, v.Nome+".json")
}

// Salvar grava o melhor cérebro de todas as gerações no arquivo 1.4 (Rede.Salvar
// é atômico e cria a pasta) e põe o resultado em Aviso.
func (v *TreinoVisual) Salvar() error {
	if v.Melhor == nil {
		v.Aviso = "ainda sem melhor cérebro: espere a 1ª geração terminar"
		return ErroSemMelhor
	}
	caminho := v.CaminhoCerebro()
	if err := v.Melhor.Salvar(caminho); err != nil {
		v.Aviso = "não salvou: " + err.Error()
		return err
	}
	v.Aviso = fmt.Sprintf("salvo em %s (nota %.1f, geração %d)", caminho, v.MelhorNota, v.MelhorGeracao)
	return nil
}

// Velocidade é o multiplicador de ticks por quadro.
func (v *TreinoVisual) Velocidade() int { return Velocidades[v.VelIndice] }

// MaisRapido e MaisLento mudam a velocidade dentro de Velocidades.
func (v *TreinoVisual) MaisRapido() { v.VelIndice = min(v.VelIndice+1, len(Velocidades)-1) }
func (v *TreinoVisual) MaisLento()  { v.VelIndice = max(v.VelIndice-1, 0) }

// Rodando conta os indivíduos que ainda dirigem nesta rodada.
func (v *TreinoVisual) Rodando() int {
	n := 0
	for i := range v.Individuos {
		if !v.Individuos[i].Fim {
			n++
		}
	}
	return n
}

// Lider é o índice do indivíduo mais adiante na rodada: dos que ainda dirigem,
// se houver algum, o de maior Distancia; senão o de maior Distancia de todos.
// -1 sem indivíduos.
func (v *TreinoVisual) Lider() int {
	lider := -1
	for i := range v.Individuos {
		ind := &v.Individuos[i]
		if lider < 0 {
			lider = i
			continue
		}
		l := &v.Individuos[lider]
		if ind.Fim != l.Fim {
			if !ind.Fim {
				lider = i
			}
			continue
		}
		if ind.Carro.Progresso.Distancia > l.Carro.Progresso.Distancia {
			lider = i
		}
	}
	return lider
}

// DistanciaAoVivo é a maior e a média das Distancia dos indivíduos na rodada
// atual (px).
func (v *TreinoVisual) DistanciaAoVivo() (melhor, media float64) {
	for i := range v.Individuos {
		d := v.Individuos[i].Carro.Progresso.Distancia
		melhor = max(melhor, d)
		media += d
	}
	if n := len(v.Individuos); n > 0 {
		media /= float64(n)
	}
	return melhor, media
}
