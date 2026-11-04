package jogo

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// Painel da rede do líder (D32): a tela de treino mostra, no canto de baixo à
// esquerda, a rede do carro que está na frente (o amarelo) ao vivo. Cada nó é
// colorido pela ativação do tick atual e cada ligação pelo sinal do peso, com a
// espessura pelo |peso|. O modelo (posições e valores) é montado aqui sem
// janela, para ser testado; o desenho só o percorre.

// AreaPainelRede é o retângulo do painel (px de tela), no canto de baixo à
// esquerda; a linha de aviso do treino fica logo abaixo.
var AreaPainelRede = image.Rect(8, AlturaTela-330, 448, AlturaTela-30)

// Margens internas do painel: topo (título), base e colunas das pontas (onde
// ficam os rótulos das entradas e das saídas).
const (
	painelTopo       = 26
	painelBase       = 10
	painelColEntrada = 112 // x do nó de entrada, contado do lado esquerdo do painel
	painelColSaida   = 96  // x do nó de saída, contado do lado direito do painel
)

// rotulosEntrada são os nomes das NEntrada entradas, na ordem de ia.Entrada
// (contrato 1.3, D20): 7 sensores, vx, vy, |v| e as duas direções da pista.
var rotulosEntrada = [ia.NEntrada]string{
	"sensor 0", "sensor 1", "sensor 2", "sensor 3", "sensor 4", "sensor 5", "sensor 6",
	"vx", "vy", "|v|", "pista x", "pista y", "pista+5 x", "pista+5 y",
}

// rotulosSaida são os nomes das 2 saídas (D9): s0 é o sinal e a intensidade da
// aceleração, s1 escolhe o eixo.
var rotulosSaida = [ia.NSaida]string{"s0 acel", "s1 eixo"}

// NoRede é um neurônio (ou entrada) do painel: posição na tela, valor da
// ativação (entre -1 e 1; as entradas já vêm normalizadas) e rótulo ("" nos
// ocultos).
type NoRede struct {
	X, Y   float64
	Valor  float64
	Rotulo string
}

// LigacaoRede liga o nó De da camada Camada ao nó Para da camada Camada+1, com
// o peso W[Para][De] daquela camada (contrato 1.4: linhas = neurônios de saída).
type LigacaoRede struct {
	Camada, De, Para int
	Peso             float64
}

// ModeloRede é a rede do líder pronta para desenhar: Nos[k] são os nós da
// camada k (entrada primeiro, saída por último); PesoMax é o maior |peso| da
// rede, a escala da espessura das ligações. Guarda cópias: não muda quando a
// rede evolui.
type ModeloRede struct {
	Individuo int // índice do líder em TreinoVisual.Individuos
	Nos       [][]NoRede
	Ligacoes  []LigacaoRede
	PesoMax   float64
}

// MontarModeloRede monta o modelo da rede r para a leitura l dentro de area:
// roda Rede.Ativacoes sobre ia.Entrada(l) (a mesma conta do controlador; a
// última camada é a saída de Forward) e espalha as camadas em colunas
// igualmente espaçadas, cada uma com os nós igualmente espaçados na vertical.
func MontarModeloRede(r *ia.Rede, l sim.Leitura, area image.Rectangle) (*ModeloRede, error) {
	if r == nil {
		return nil, fmt.Errorf("painel da rede: sem rede")
	}
	ativ, err := r.Ativacoes(ia.Entrada(l))
	if err != nil {
		return nil, err
	}
	m := &ModeloRede{Nos: make([][]NoRede, len(ativ))}
	x0 := float64(area.Min.X + painelColEntrada)
	x1 := float64(area.Max.X - painelColSaida)
	y0 := float64(area.Min.Y + painelTopo)
	y1 := float64(area.Max.Y - painelBase)
	ultima := len(ativ) - 1
	for k, valores := range ativ {
		x := x0
		if ultima > 0 {
			x = x0 + (x1-x0)*float64(k)/float64(ultima)
		}
		nos := make([]NoRede, len(valores))
		for i, v := range valores {
			y := (y0 + y1) / 2
			if n := len(valores); n > 1 {
				y = y0 + (y1-y0)*float64(i)/float64(n-1)
			}
			nos[i] = NoRede{X: x, Y: y, Valor: v, Rotulo: rotuloNo(k, ultima, i, len(valores))}
		}
		m.Nos[k] = nos
	}
	for k, w := range r.Pesos {
		for p := 0; p < w.L; p++ {
			for d := 0; d < w.C; d++ {
				peso := w.At(p, d)
				m.Ligacoes = append(m.Ligacoes, LigacaoRede{Camada: k, De: d, Para: p, Peso: peso})
				m.PesoMax = math.Max(m.PesoMax, math.Abs(peso))
			}
		}
	}
	return m, nil
}

// rotuloNo é o rótulo do nó i da camada k (a última é ultima, com n nós): as
// entradas e as saídas têm nome, os ocultos não.
func rotuloNo(k, ultima, i, n int) string {
	switch {
	case k == 0 && n == ia.NEntrada:
		return rotulosEntrada[i]
	case k == 0:
		return fmt.Sprintf("e%d", i)
	case k == ultima && n == ia.NSaida:
		return rotulosSaida[i]
	case k == ultima:
		return fmt.Sprintf("s%d", i)
	}
	return ""
}

// AlternarPainelRede liga ou desliga o painel da rede (tecla N).
func (v *TreinoVisual) AlternarPainelRede() { v.PainelRedeVisivel = !v.PainelRedeVisivel }

// ModeloRede é o modelo da rede do líder no último AtualizarModeloRede; nil
// antes da primeira atualização.
func (v *TreinoVisual) ModeloRede() *ModeloRede { return v.modeloRede }

// AtualizarModeloRede remonta o modelo para o líder (Lider) com a Leitura
// atual do carro dele. A rede troca sozinha quando muda o líder. Sem ninguém
// dirigindo (todos parados) mantém o modelo anterior: a última rede.
func (v *TreinoVisual) AtualizarModeloRede() {
	if v.Rodando() == 0 {
		return
	}
	i := v.Lider()
	if i < 0 {
		return
	}
	ind := &v.Individuos[i]
	if ind.ctl == nil || ind.ctl.Rede == nil {
		return
	}
	l := sim.Ler(&ind.Carro, v.Treino.Pista, v.Treino.Cfg)
	m, err := MontarModeloRede(ind.ctl.Rede, l, AreaPainelRede)
	if err != nil {
		return // rede malformada: fica a anterior
	}
	m.Individuo = i
	v.modeloRede = m
}

// desenharRede desenha o painel da rede do líder (se ligado e já montado).
func (t *TreinoTela) desenharRede(tela *ebiten.Image) {
	m := t.V.ModeloRede()
	if !t.V.PainelRedeVisivel || m == nil {
		return
	}
	a := AreaPainelRede
	vector.FillRect(tela, float32(a.Min.X), float32(a.Min.Y), float32(a.Dx()), float32(a.Dy()), corPainel, false)
	texto(tela, fmt.Sprintf("rede do lider (carro %d)   N: esconde", m.Individuo), a.Min.X+8, a.Min.Y+4)
	for _, lg := range m.Ligacoes {
		de, para := m.Nos[lg.Camada][lg.De], m.Nos[lg.Camada+1][lg.Para]
		cor, esp := corLigacao(lg.Peso, m.PesoMax)
		vector.StrokeLine(tela, float32(de.X), float32(de.Y), float32(para.X), float32(para.Y), esp, cor, true)
	}
	ultima := len(m.Nos) - 1
	borda := color.RGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
	for k, nos := range m.Nos {
		for _, n := range nos {
			vector.FillCircle(tela, float32(n.X), float32(n.Y), 5, corAtivacao(n.Valor), true)
			vector.StrokeCircle(tela, float32(n.X), float32(n.Y), 5, 1, borda, true)
			switch {
			case k == 0 && n.Rotulo != "":
				texto(tela, fmt.Sprintf("%-9s%+.2f", n.Rotulo, n.Valor), a.Min.X+6, int(n.Y)-7)
			case k == ultima && n.Rotulo != "":
				texto(tela, fmt.Sprintf("%s %+.2f", n.Rotulo, n.Valor), int(n.X)+10, int(n.Y)-7)
			}
		}
	}
}

// corAtivacao colore o valor v (-1 a 1): cinza perto de 0, laranja para
// positivo, azul para negativo; mais forte quanto maior |v|.
func corAtivacao(v float64) color.RGBA {
	f := math.Min(math.Abs(v), 1)
	if math.IsNaN(v) {
		f = 0
	}
	base := color.RGBA{R: 0x50, G: 0x55, B: 0x5c, A: 0xff}
	alvo := color.RGBA{R: 0xff, G: 0x9a, B: 0x2e, A: 0xff}
	if v < 0 {
		alvo = color.RGBA{R: 0x3d, G: 0x9b, B: 0xff, A: 0xff}
	}
	mix := func(a, b uint8) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*f + 0.5) }
	return color.RGBA{R: mix(base.R, alvo.R), G: mix(base.G, alvo.G), B: mix(base.B, alvo.B), A: 0xff}
}

// corLigacao devolve a cor (verde para peso positivo, vermelha para negativo)
// e a espessura de uma ligação de peso w numa rede cujo maior |peso| é maxPeso:
// a espessura cresce com |w|/maxPeso, de 0,5 a 3 px, e as fracas ficam mais
// transparentes.
func corLigacao(w, maxPeso float64) (color.RGBA, float32) {
	f := 0.0
	if maxPeso > 0 && !math.IsNaN(w) {
		f = math.Min(math.Abs(w)/maxPeso, 1)
	}
	cor := color.RGBA{R: 0x4c, G: 0xd1, B: 0x64}
	if w < 0 {
		cor = color.RGBA{R: 0xe5, G: 0x45, B: 0x45}
	}
	cor.A = uint8(40 + 180*f)
	return cor, float32(0.5 + 2.5*f)
}
