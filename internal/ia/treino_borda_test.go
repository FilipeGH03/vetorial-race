package ia

import (
	"fmt"
	"path/filepath"
	"testing"

	"vetorial-race/internal/sim"
)

// O que o treino mede tem de ser o que o jogo faz: a mesma rede, rodada numa
// sim.Corrida de NLargadas carros com a Config do treino (a da corrida, D23)
// pelos mesmos ticks, só ela dirigindo o carro k (os outros parados), dá a
// mesma distância, voltas e batidas que a largada k de Treino.Rodar (D22).
func TestRodarIgualACorridaDoJogo(t *testing.T) {
	tr := treinoOval(t)
	for seed := uint64(1); seed <= 2; seed++ {
		params := ParamsPadrao()
		params.Seed = seed
		params.Tamanho = 6
		pop, err := NovaPopulacao(params)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range pop.Redes {
			a := tr.Rodar(r)
			for k, want := range a.Largadas {
				ctls := make([]sim.Controlador, NLargadas)
				ctls[k] = NovoControlador(r)
				cor, err := sim.NovaCorrida(tr.Pista, tr.Cfg, sim.ModoVoltas, 10, ctls, -1)
				if err != nil {
					t.Fatal(err)
				}
				batidas := 0
				for n := 0; n < want.Ticks && !cor.Terminada; n++ {
					cor.Step()
					if cor.Carros[k].Batido {
						batidas++
					}
				}
				c := cor.Carros[k]
				if c.Progresso.Distancia != want.Distancia || c.Progresso.Voltas != want.Voltas || cor.Tick != want.Ticks ||
					batidas != want.Batidas || !c.Vivo {
					t.Errorf("seed %d rede %d largada %d: corrida (dist %v, voltas %d, ticks %d, batidas %d, vivo %v), treino %+v",
						seed, i, k, c.Progresso.Distancia, c.Progresso.Voltas, cor.Tick, batidas, c.Vivo, want)
				}
			}
		}
	}
}

// A nota gravada no cérebro é a que o jogo reproduz ao recarregá-lo.
func TestCerebroSalvoReproduzANota(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 10
	params.Elite = 2
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	for g := 0; g < 3; g++ {
		if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
			t.Fatal(err)
		}
		if g < 2 {
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
		}
	}
	melhor, nota := pop.Melhor()
	melhor = melhor.Copia()
	melhor.Pista = "oval"
	arq := filepath.Join(t.TempDir(), "c", "oval.json")
	if err := melhor.Salvar(arq); err != nil {
		t.Fatal(err)
	}
	lido, err := CarregarRede(arq)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.Rodar(lido).Pontuacao; got != nota {
		t.Errorf("cérebro recarregado dá %v, nota gravada %v", got, nota)
	}
}

// Mesma seed, mesmas notas (reprodutibilidade do treino, em paralelo).
func TestTreinoMesmaSeedMesmoResultado(t *testing.T) {
	tr := treinoOval(t)
	rodar := func() []float64 {
		params := ParamsPadrao()
		params.Tamanho = 12
		params.Elite = 2
		pop, err := NovaPopulacao(params)
		if err != nil {
			t.Fatal(err)
		}
		var notas []float64
		for g := 0; g < 4; g++ {
			if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
				t.Fatal(err)
			}
			_, n := pop.Melhor()
			notas = append(notas, n)
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
		}
		return notas
	}
	a, b := rodar(), rodar()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("geração %d: %v x %v", i, a[i], b[i])
		}
	}
}

// População de 2 (mínimo) e Elite 0 também avaliam, mesmo com mais workers
// que indivíduos.
func TestAvaliarPopulacaoMinima(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 2
	params.Elite = 0
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tr.AvaliarPopulacao(pop, 16)
	if err != nil || len(res) != 2 {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}

// provaCorrida é o critério de prova da C3 (D22): o cérebro r, numa
// sim.Corrida de 60 s com ConfigPadrao, largando de cada uma das NLargadas
// posições de sim.Largada(p, NLargadas) (só ele dirige; os outros carros ficam
// parados), completa ≥ 1 volta e nunca passa 2 s (120 ticks) sem a Distancia
// aumentar. Devolve um texto por largada que falhou.
func provaCorrida(p *sim.Pista, r *Rede) []string {
	const ticks, maxParado = 3600, 120
	var falhas []string
	for k := range NLargadas {
		ctls := make([]sim.Controlador, NLargadas)
		ctls[k] = NovoControlador(r)
		cor, err := sim.NovaCorrida(p, sim.ConfigPadrao(), sim.ModoVoltas, 10, ctls, -1)
		if err != nil {
			return []string{err.Error()}
		}
		ultima, parado, pior := cor.Carros[k].Progresso.Distancia, 0, 0
		for cor.Tick < ticks && !cor.Terminada {
			cor.Step()
			if d := cor.Carros[k].Progresso.Distancia; d > ultima {
				ultima, parado = d, 0
				continue
			}
			parado++
			pior = max(pior, parado)
		}
		c := cor.Carros[k]
		if c.Progresso.Voltas < 1 || pior > maxParado {
			falhas = append(falhas, fmt.Sprintf("largada %d: %d voltas, distância %.0f, %d ticks seguidos sem avançar",
				k, c.Progresso.Voltas, c.Progresso.Distancia, pior))
		}
	}
	return falhas
}

// O cérebro versionado do oval (gerado por go run ./cmd/treino -pista
// pistas/oval.json -seed 1) passa na prova da C3 em todas as largadas.
// Pulado com -short.
func TestCerebroOvalCompletaVoltaDeTodasAsLargadas(t *testing.T) {
	if testing.Short() {
		t.Skip("prova da C3")
	}
	r, err := CarregarRede("../../cerebros/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range provaCorrida(pistaOval(t), r) {
		t.Error(f)
	}
}

// Critério de prova da C3 de ponta a ponta: o treino padrão (seed 1, 50
// indivíduos, 14-12-2, 300 gerações, o comando da prova) produz um cérebro
// que passa na prova em todas as largadas. Pulado com -short (leva minutos).
func TestTreinoOvalPassaNaProva(t *testing.T) {
	if testing.Short() {
		t.Skip("treino longo")
	}
	tr := treinoOval(t)
	pop, err := NovaPopulacao(ParamsPadrao())
	if err != nil {
		t.Fatal(err)
	}
	const geracoes = 300
	var melhor *Rede
	for g := range geracoes {
		if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
			t.Fatal(err)
		}
		if r, nota := pop.Melhor(); melhor == nil || nota > melhor.Pontuacao {
			melhor = r.Copia()
		}
		if g < geracoes-1 {
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("melhor: geração %d, nota %.1f", melhor.Geracao, melhor.Pontuacao)
	for _, f := range provaCorrida(tr.Pista, melhor) {
		t.Error(f)
	}
}
