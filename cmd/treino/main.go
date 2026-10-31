// Comando treino evolui as redes da IA numa pista, sem janela (D5), e grava o
// melhor cérebro (contrato 1.4).
//
//	go run ./cmd/treino -pista pistas/oval.json -seed 1
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

func main() {
	log.SetFlags(0)
	pista := flag.String("pista", "pistas/oval.json", "arquivo da pista")
	geracoes := flag.Int("geracoes", 300, "quantas gerações avaliar")
	seed := flag.Uint64("seed", 1, "seed do sorteio (mesma seed, mesmo resultado)")
	saida := flag.String("saida", "", "arquivo do cérebro (padrão: cerebros/<arquivo da pista sem extensão>.json)")
	camadas := flag.String("camadas", juntar(ia.CamadasPadrao),
		fmt.Sprintf("neurônios por camada, entrada %d e saída %d (ex.: %d,16,8,%d)", ia.NEntrada, ia.NSaida, ia.NEntrada, ia.NSaida))
	populacao := flag.Int("populacao", ia.ParamsPadrao().Tamanho, "indivíduos por geração")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "uso: treino [-pista arq] [-geracoes n] [-seed n] [-saida arq] [-camadas %s] [-populacao n]\n", juntar(ia.CamadasPadrao))
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := rodar(*pista, *geracoes, *seed, *saida, *camadas, *populacao); err != nil {
		log.Fatal("treino: ", err)
	}
}

func rodar(arqPista string, geracoes int, seed uint64, saida, camadas string, populacao int) error {
	if geracoes < 1 {
		return fmt.Errorf("-geracoes %d: precisa de ao menos 1", geracoes)
	}
	forma, err := separar(camadas)
	if err != nil {
		return err
	}
	p, err := sim.CarregarPista(arqPista)
	if err != nil {
		return err
	}
	t, err := ia.NovoTreino(p)
	if err != nil {
		return err
	}
	params := ia.ParamsPadrao()
	params.Camadas = forma
	params.Seed = seed
	params.Tamanho = populacao
	pop, err := ia.NovaPopulacao(params)
	if err != nil {
		return err
	}
	if saida == "" {
		if saida, err = saidaPadrao(arqPista); err != nil {
			return err
		}
	}

	fmt.Printf("pista %s (%.0f px por volta), rede %v, %d indivíduos, seed %d, %d largadas de até %.0f s cada (fim após %d ticks sem avançar)\n",
		p.Nome, p.Comprimento, forma, populacao, seed, ia.NLargadas, ia.TempoMaxTreino, ia.TicksParado)
	var melhor *ia.Rede
	// primeiraVolta: 1ª geração em que algum indivíduo completou a volta de
	// alguma largada; todasLargadas: 1ª em que o melhor a completou de todas.
	primeiraVolta, todasLargadas := -1, -1
	inicio := time.Now()
	for g := 0; ; g++ {
		ini := time.Now()
		res, err := t.AvaliarPopulacao(pop, 0)
		if err != nil {
			return err
		}
		r, nota := pop.Melhor()
		b := res[pop.Ranking()[0]]
		fmt.Printf("geração %3d  melhor %9.1f  distância média %7.0f  voltas (mín) %2d  com volta %d/%d  1ª volta (pior) %5s  batidas %4d  paradas %d  (%.1fs)\n",
			pop.Geracao, nota, distanciaMedia(b), b.VoltasMin(), b.ComVolta(), ia.NLargadas, tempoVolta(b), batidas(b), paradas(b),
			time.Since(ini).Seconds())
		if primeiraVolta < 0 && algumaVolta(res) {
			primeiraVolta = pop.Geracao
		}
		if todasLargadas < 0 && b.VoltasMin() >= 1 {
			todasLargadas = pop.Geracao
		}
		if melhor == nil || nota > melhor.Pontuacao {
			melhor = r.Copia()
		}
		if g == geracoes-1 {
			break
		}
		// Proxima deixa os filhos com nota 0: a avaliação do começo do laço
		// vem antes do próximo Proxima.
		if err := pop.Proxima(); err != nil {
			return err
		}
	}

	melhor.Pista = p.Nome
	if err := melhor.Salvar(saida); err != nil {
		return err
	}
	if primeiraVolta >= 0 {
		fmt.Printf("1ª volta completa na geração %d\n", primeiraVolta)
	} else {
		fmt.Println("nenhum indivíduo completou uma volta")
	}
	if todasLargadas >= 0 {
		fmt.Printf("1ª volta completa das %d largadas na geração %d\n", ia.NLargadas, todasLargadas)
	} else {
		fmt.Printf("nenhum indivíduo completou a volta das %d largadas\n", ia.NLargadas)
	}
	fmt.Printf("melhor: geração %d, pontuação %.1f -> %s (%.1fs no total)\n",
		melhor.Geracao, melhor.Pontuacao, saida, time.Since(inicio).Seconds())
	return nil
}

// tempoVolta é o tempo da 1ª volta na largada mais lenta; "-" se alguma
// largada não completou a volta.
func tempoVolta(a ia.Avaliacao) string {
	if a.VoltasMin() < 1 {
		return "-"
	}
	pior := 0.0
	for _, r := range a.Largadas {
		pior = max(pior, r.TempoVolta)
	}
	return fmt.Sprintf("%.1fs", pior)
}

func distanciaMedia(a ia.Avaliacao) float64 {
	s := 0.0
	for _, r := range a.Largadas {
		s += r.Distancia
	}
	return s / ia.NLargadas
}

func batidas(a ia.Avaliacao) int {
	n := 0
	for _, r := range a.Largadas {
		n += r.Batidas
	}
	return n
}

// paradas conta as largadas encerradas por ficar TicksParado ticks sem avançar.
func paradas(a ia.Avaliacao) int {
	n := 0
	for _, r := range a.Largadas {
		if r.Parou {
			n++
		}
	}
	return n
}

// algumaVolta diz se algum indivíduo completou a volta de alguma largada.
func algumaVolta(res []ia.Avaliacao) bool {
	for _, a := range res {
		if a.ComVolta() > 0 {
			return true
		}
	}
	return false
}

// saidaPadrao devolve cerebros/<arquivo da pista sem extensão>.json (contrato
// 1.4), o nome que o jogo procura. Só o nome do arquivo entra: o cérebro
// nunca vai para fora de cerebros/.
func saidaPadrao(arqPista string) (string, error) {
	base := filepath.Base(arqPista)
	nome := strings.TrimSuffix(base, filepath.Ext(base))
	if nome == "" || nome == "." || nome == ".." || strings.ContainsAny(nome, `/\`) {
		return "", fmt.Errorf("-pista %q: nome de arquivo não serve para o cérebro; use -saida", arqPista)
	}
	return filepath.Join("cerebros", nome+".json"), nil
}

func juntar(c []int) string {
	s := make([]string, len(c))
	for i, n := range c {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ",")
}

// separar lê "14,12,2"; a forma é validada pela população (entrada 14, saída 2).
func separar(s string) ([]int, error) {
	partes := strings.Split(s, ",")
	c := make([]int, len(partes))
	for i, p := range partes {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("-camadas %q: %v", s, err)
		}
		c[i] = n
	}
	return c, nil
}
