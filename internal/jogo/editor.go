package jogo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vetorial-race/internal/sim"
)

// Regras do editor de pista (B4).
const (
	LarguraInicial = 90.0 // px: largura de uma pista nova
	PassoLargura   = 10.0 // px: quanto a roda do mouse (ou as setas) muda a largura

	// EspacoMinTraco é a menor distância (px) entre dois pontos guardados do
	// traço do mouse: pontos mais perto que isso do anterior são descartados.
	EspacoMinTraco = 12.0
	// PassosSuavizar é quantas vezes a linha é suavizada (média 1-2-1 entre
	// vizinhos): reamostrar não suaviza, e uma tremida de mais de ~2 px do mouse
	// seria recusada como ErrPistaCruza (testador da A2).
	PassosSuavizar = 3
)

// ErrNaoCabe indica uma pista cujas paredes saem da janela de 1280x720 (D17).
var ErrNaoCabe = errors.New("a pista não cabe na janela")

// CabeNaTela diz se todas as paredes da pista estão dentro de [0,1280]x[0,720] (D17).
func CabeNaTela(p *sim.Pista) bool {
	for _, parede := range [2][]sim.Vec2{p.ParedeEsq, p.ParedeDir} {
		for _, q := range parede {
			if q.X < 0 || q.X > LarguraTela || q.Y < 0 || q.Y > AlturaTela {
				return false
			}
		}
	}
	return true
}

// TracoParaPontos converte o traço do mouse (pontos em coordenadas de tela, na
// ordem em que foram arrastados) na linha central do arquivo 1.2: coordenadas de
// mundo (y para cima), sem pontos a menos de EspacoMinTraco do anterior, sem a
// sobra de quem passou da largada (o traço que volta perto do primeiro ponto
// termina ali) e suavizada. O circuito fecha sozinho: o último ponto liga no
// primeiro. Devolve nil se sobrarem menos de 3 pontos.
func TracoParaPontos(tela []sim.Vec2, largura float64) []sim.Vec2 {
	var p []sim.Vec2
	for _, t := range tela {
		x, y := TelaParaMundo(t.X, t.Y)
		q := sim.Vec2{X: x, Y: y}
		if len(p) == 0 || q.Sub(p[len(p)-1]).Len() >= EspacoMinTraco {
			p = append(p, q)
		}
	}
	p = cortarVolta(p, largura)
	for len(p) > 3 && p[len(p)-1].Sub(p[0]).Len() < EspacoMinTraco {
		p = p[:len(p)-1] // o último ponto quase em cima do primeiro
	}
	if len(p) < 3 {
		return nil
	}
	for k := 0; k < PassosSuavizar; k++ {
		p = suavizar(p)
	}
	return p
}

// cortarVolta corta o traço no primeiro ponto em que ele, depois de andar mais de
// 4 larguras, volta a menos de meia largura do primeiro ponto: dali em diante o
// traço só repetiria o começo da pista.
func cortarVolta(p []sim.Vec2, largura float64) []sim.Vec2 {
	var andado float64
	for j := 1; j < len(p); j++ {
		andado += p[j].Sub(p[j-1]).Len()
		if andado > 4*largura && p[j].Sub(p[0]).Len() < largura/2 {
			return p[:j]
		}
	}
	return p
}

// suavizar faz uma passada de média 1-2-1 com os vizinhos no circuito fechado p.
func suavizar(p []sim.Vec2) []sim.Vec2 {
	n := len(p)
	q := make([]sim.Vec2, n)
	for i := range p {
		q[i] = p[(i-1+n)%n].Add(p[i].Scale(2)).Add(p[(i+1)%n]).Scale(0.25)
	}
	return q
}

// primeiroCruzamento acha onde a linha central p (circuito fechado) cruza a si
// mesma: o ponto de cruzamento do primeiro par de trechos não vizinhos que se
// cortam. É só uma pista visual para o erro; quem decide se a pista vale é
// sim.NovaPista.
func primeiroCruzamento(p []sim.Vec2) (sim.Vec2, bool) {
	n := len(p)
	for i := 0; i < n; i++ {
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // vizinhos pelo fechamento
			}
			a, b := p[i], p[(i+1)%n]
			c, d := p[j], p[(j+1)%n]
			r, s := b.Sub(a), d.Sub(c)
			den := r.Cross(s)
			if den == 0 {
				continue
			}
			t := c.Sub(a).Cross(s) / den
			u := c.Sub(a).Cross(r) / den
			if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
				return a.Add(r.Scale(t)), true
			}
		}
	}
	return sim.Vec2{}, false
}

// Editor guarda o estado do editor de pista: o traço que o mouse está
// desenhando, a linha central resultante e o resultado da validação. Não depende
// de janela; EditorTela liga o mouse e o desenho.
type Editor struct {
	Nome    string
	Largura float64

	Traco      []sim.Vec2 // traço bruto do mouse em coordenadas de tela, durante o arrasto
	Arrastando bool
	Pontos     []sim.Vec2 // linha central em coordenadas de mundo (o que o arquivo grava)

	Pista      *sim.Pista // pista montada, nil se Pontos não valem
	Erro       error      // por que não vale (errors.Is com sim.ErrPista* ou ErrNaoCabe)
	Cruzamento sim.Vec2   // onde a linha se cruza, se TemCruzamento
	// TemCruzamento diz se Cruzamento vale.
	TemCruzamento bool
}

// NovoEditor começa um editor vazio para a pista de nome dado.
func NovoEditor(nome string) *Editor {
	return &Editor{Nome: nome, Largura: LarguraInicial}
}

// Limpar apaga o traço e a pista.
func (e *Editor) Limpar() {
	e.Traco, e.Arrastando, e.Pontos = nil, false, nil
	e.Pista, e.Erro, e.TemCruzamento = nil, nil, false
}

// Comecar inicia um traço novo no ponto (x, y) da tela, apagando o anterior.
func (e *Editor) Comecar(x, y float64) {
	e.Limpar()
	e.Arrastando = true
	e.Arrastar(x, y)
}

// Arrastar acrescenta o ponto (x, y) da tela ao traço. O ponto é preso a uma
// margem de meia largura da borda da janela: as paredes ficam dentro dela (D17).
func (e *Editor) Arrastar(x, y float64) {
	if !e.Arrastando {
		return
	}
	m := e.Largura / 2
	x = min(max(x, m), LarguraTela-m)
	y = min(max(y, m), AlturaTela-m)
	if n := len(e.Traco); n > 0 && (sim.Vec2{X: x, Y: y}).Sub(e.Traco[n-1]).Len() < 1 {
		return
	}
	e.Traco = append(e.Traco, sim.Vec2{X: x, Y: y})
}

// Soltar termina o traço: converte em linha central (fechando o circuito) e valida.
func (e *Editor) Soltar() {
	if !e.Arrastando {
		return
	}
	e.Arrastando = false
	e.Pontos = TracoParaPontos(e.Traco, e.Largura)
	e.Traco = nil
	e.Validar()
}

// AjustarLargura muda a largura da pista (40 a 200 px) e valida de novo.
func (e *Editor) AjustarLargura(delta float64) {
	e.Largura = min(max(e.Largura+delta, sim.LarguraPistaMin), sim.LarguraPistaMax)
	e.Validar()
}

// Validar monta a pista com sim.NovaPista e confere que cabe na janela (D17). O
// resultado fica em Pista (nil se não vale) e Erro; para um cruzamento, também
// em Cruzamento.
func (e *Editor) Validar() {
	e.Pista, e.Erro, e.TemCruzamento = nil, nil, false
	if e.Arrastando {
		return
	}
	if len(e.Pontos) == 0 {
		e.Erro = fmt.Errorf("%w: traço curto demais", sim.ErrPistaCurta)
		return
	}
	p, err := sim.NovaPista(e.Nome, e.Largura, e.Pontos)
	if err == nil && !CabeNaTela(p) {
		err = fmt.Errorf("%w (paredes fora de %dx%d): diminua a largura ou o traçado", ErrNaoCabe, LarguraTela, AlturaTela)
	}
	if err != nil {
		e.Erro = err
		if errors.Is(err, sim.ErrPistaCruza) {
			e.Cruzamento, e.TemCruzamento = primeiroCruzamento(e.Pontos)
		}
		return
	}
	e.Pista = p
}

// Salvar grava a pista em caminho (arquivo 1.2). O nome da pista é o do arquivo,
// sem a extensão. Recusa se a pista não vale.
func (e *Editor) Salvar(caminho string) error {
	nome := strings.TrimSuffix(filepath.Base(caminho), filepath.Ext(caminho))
	if strings.TrimSpace(nome) == "" {
		return fmt.Errorf("%w: nome de arquivo vazio", sim.ErrPistaInvalida)
	}
	e.Nome = nome
	e.Validar()
	if e.Erro != nil {
		return e.Erro
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return err
	}
	return sim.SalvarPista(caminho, e.Pista)
}

// Carregar abre uma pista salva para continuar editando (a linha central e a
// largura do arquivo). Se o arquivo não vale ou não cabe na janela, devolve o
// erro e deixa o editor como estava.
func (e *Editor) Carregar(caminho string) error {
	p, err := sim.CarregarPista(caminho)
	if err != nil {
		return err
	}
	if !CabeNaTela(p) {
		return fmt.Errorf("%w (%s): paredes fora de %dx%d", ErrNaoCabe, caminho, LarguraTela, AlturaTela)
	}
	e.Limpar()
	e.Nome, e.Largura = p.Nome, p.Largura
	e.Pontos = append([]sim.Vec2(nil), p.Pontos...)
	e.Validar()
	return nil
}
