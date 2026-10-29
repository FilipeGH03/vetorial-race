package sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// Regras do arquivo de pista (contrato 1.2).
const (
	VersaoPista      = 1
	LarguraPistaMin  = 40.0  // px
	LarguraPistaMax  = 200.0 // px
	EspacamentoPista = 20.0  // px: distância aproximada entre pontos reamostrados
	PontosPistaMin   = 8     // mínimo de pontos depois de reamostrar
	pontosArquivoMin = 3     // mínimo de pontos no arquivo para formar um circuito
)

// Erros de validação da pista. Use errors.Is para distingui-los.
var (
	ErrPistaInvalida = errors.New("pista inválida")
	ErrPistaCurta    = errors.New("pista curta demais")
	ErrPistaCruza    = errors.New("pista se cruza")
)

// Segmento é um segmento de reta de A a B.
type Segmento struct{ A, B Vec2 }

// Pista é a pista pronta para a simulação. Todas as polilinhas são circuitos
// fechados implícitos: o último ponto liga no primeiro. Coordenadas matemáticas.
type Pista struct {
	Nome    string
	Largura float64
	Pontos  []Vec2 // linha central como veio do arquivo (é o que SalvarPista grava)

	Centro    []Vec2 // linha central reamostrada, um ponto a cada Passo px; Centro[0] = Pontos[0]
	ParedeEsq []Vec2 // Centro[i] + normal esquerda * Largura/2
	ParedeDir []Vec2 // Centro[i] - normal esquerda * Largura/2

	// Checkpoints[i] vai de ParedeEsq[i] a ParedeDir[i]. O 0 é a largada/chegada.
	Checkpoints []Segmento

	Passo       float64 // distância ao longo da linha central entre pontos consecutivos de Centro
	Comprimento float64 // comprimento de uma volta pela linha central
}

// arquivoPista é o formato JSON do contrato 1.2.
type arquivoPista struct {
	Versao  int         `json:"versao"`
	Nome    string      `json:"nome"`
	Largura float64     `json:"largura"`
	Centro  [][]float64 `json:"centro"`
}

// CarregarPista lê pistas/<nome>.json, valida e monta a pista.
func CarregarPista(caminho string) (*Pista, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, fmt.Errorf("carregar pista: %w", err)
	}
	defer f.Close()
	p, err := LerPista(f)
	if err != nil {
		return nil, fmt.Errorf("carregar pista %s: %w", caminho, err)
	}
	return p, nil
}

// LerPista decodifica o JSON do contrato 1.2 e monta a pista.
func LerPista(r io.Reader) (*Pista, error) {
	var a arquivoPista
	if err := json.NewDecoder(r).Decode(&a); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrPistaInvalida, err)
	}
	if a.Versao != VersaoPista {
		return nil, fmt.Errorf("%w: versao %d, esperada %d", ErrPistaInvalida, a.Versao, VersaoPista)
	}
	pontos := make([]Vec2, len(a.Centro))
	for i, c := range a.Centro {
		if len(c) != 2 {
			return nil, fmt.Errorf("%w: centro[%d] tem %d coordenadas, esperadas 2", ErrPistaInvalida, i, len(c))
		}
		pontos[i] = Vec2{c[0], c[1]}
	}
	return NovaPista(a.Nome, a.Largura, pontos)
}

// NovaPista valida os dados e monta a pista: reamostra a linha central, gera as
// paredes e os checkpoints. Devolve erro (sem pânico) se algo não fecha.
func NovaPista(nome string, largura float64, pontos []Vec2) (*Pista, error) {
	if strings.TrimSpace(nome) == "" {
		return nil, fmt.Errorf("%w: nome vazio", ErrPistaInvalida)
	}
	if !(largura >= LarguraPistaMin && largura <= LarguraPistaMax) {
		return nil, fmt.Errorf("%w: largura %v fora de [%v, %v]", ErrPistaInvalida, largura, LarguraPistaMin, LarguraPistaMax)
	}
	if len(pontos) < pontosArquivoMin {
		return nil, fmt.Errorf("%w: centro tem %d pontos, mínimo %d", ErrPistaCurta, len(pontos), pontosArquivoMin)
	}
	for i, q := range pontos {
		if math.IsNaN(q.X) || math.IsNaN(q.Y) || math.IsInf(q.X, 0) || math.IsInf(q.Y, 0) {
			return nil, fmt.Errorf("%w: centro[%d] não é finito", ErrPistaInvalida, i)
		}
	}

	centro, passo, total := reamostrar(pontos, EspacamentoPista)
	if len(centro) < PontosPistaMin {
		return nil, fmt.Errorf("%w: %d pontos depois de reamostrar (%.0f px), mínimo %d", ErrPistaCurta, len(centro), total, PontosPistaMin)
	}

	n := len(centro)
	esq := make([]Vec2, n)
	dir := make([]Vec2, n)
	cps := make([]Segmento, n)
	meia := largura / 2
	for i := range centro {
		t := centro[(i+1)%n].Sub(centro[(i-1+n)%n])
		if t.Len() == 0 {
			return nil, fmt.Errorf("%w: a linha central volta sobre si no ponto %d", ErrPistaCruza, i)
		}
		t = t.Norm()
		normal := Vec2{-t.Y, t.X} // esquerda de quem anda no sentido da pista
		esq[i] = centro[i].Add(normal.Scale(meia))
		dir[i] = centro[i].Sub(normal.Scale(meia))
		cps[i] = Segmento{esq[i], dir[i]}
	}

	if i, j, ok := cruzaPropria(centro); ok {
		return nil, fmt.Errorf("%w: a linha central cruza a si mesma (trechos %d e %d)", ErrPistaCruza, i, j)
	}
	if i, j, ok := trechosColados(centro, passo, largura); ok {
		return nil, fmt.Errorf("%w: os pontos %d e %d da linha central estão a menos de uma largura (%v px); a pista se sobrepõe", ErrPistaCruza, i, j, largura)
	}
	if i, j, ok := cruzaEntre(esq, dir); ok {
		return nil, fmt.Errorf("%w: a parede esquerda (trecho %d) cruza a direita (trecho %d); trechos próximos demais", ErrPistaCruza, i, j)
	}

	return &Pista{
		Nome:        nome,
		Largura:     largura,
		Pontos:      append([]Vec2(nil), pontos...),
		Centro:      centro,
		ParedeEsq:   esq,
		ParedeDir:   dir,
		Checkpoints: cps,
		Passo:       passo,
		Comprimento: total,
	}, nil
}

// SalvarPista valida a pista e grava o JSON do contrato 1.2 com os Pontos
// originais. Carregar o arquivo depois devolve a mesma pista.
func SalvarPista(caminho string, p *Pista) error {
	var buf bytes.Buffer
	if err := EscreverPista(&buf, p); err != nil {
		return fmt.Errorf("salvar pista %s: %w", caminho, err)
	}
	if err := os.WriteFile(caminho, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("salvar pista: %w", err)
	}
	return nil
}

// EscreverPista valida a pista e escreve o JSON do contrato 1.2, um ponto por linha.
func EscreverPista(w io.Writer, p *Pista) error {
	if p == nil {
		return fmt.Errorf("%w: pista nula", ErrPistaInvalida)
	}
	if _, err := NovaPista(p.Nome, p.Largura, p.Pontos); err != nil {
		return err
	}
	nome, err := json.Marshal(p.Nome)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "{\n  \"versao\": %d,\n  \"nome\": %s,\n  \"largura\": %s,\n  \"centro\": [\n",
		VersaoPista, nome, numero(p.Largura))
	for i, q := range p.Pontos {
		sep := ","
		if i == len(p.Pontos)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "    [%s, %s]%s\n", numero(q.X), numero(q.Y), sep)
	}
	b.WriteString("  ]\n}\n")
	_, err = io.WriteString(w, b.String())
	return err
}

// numero formata um float64 com o mínimo de dígitos que voltam ao mesmo valor.
func numero(x float64) string { return strconv.FormatFloat(x, 'g', -1, 64) }

// reamostrar percorre o circuito fechado pontos e devolve n pontos igualmente
// espaçados ao longo dele (passo = total/n ≈ esp), começando em pontos[0].
func reamostrar(pontos []Vec2, esp float64) (res []Vec2, passo, total float64) {
	m := len(pontos)
	comp := make([]float64, m)
	for i := range pontos {
		comp[i] = pontos[(i+1)%m].Sub(pontos[i]).Len()
		total += comp[i]
	}
	n := int(math.Round(total / esp))
	if n <= 0 {
		return nil, 0, total
	}
	passo = total / float64(n)
	res = make([]Vec2, 0, n)
	seg, inicio := 0, 0.0 // trecho atual e distância acumulada no início dele
	for k := 0; k < n; k++ {
		alvo := float64(k) * passo
		for seg < m-1 && inicio+comp[seg] < alvo {
			inicio += comp[seg]
			seg++
		}
		a, b := pontos[seg], pontos[(seg+1)%m]
		f := 0.0
		if comp[seg] > 0 {
			f = math.Min(1, (alvo-inicio)/comp[seg])
		}
		res = append(res, a.Add(b.Sub(a).Scale(f)))
	}
	return res, passo, total
}

// cruzaPropria diz se dois trechos não vizinhos do circuito fechado p se tocam.
func cruzaPropria(p []Vec2) (int, int, bool) {
	n := len(p)
	for i := 0; i < n; i++ {
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // vizinhos pelo fechamento
			}
			if segmentosSeTocam(p[i], p[(i+1)%n], p[j], p[(j+1)%n]) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

// trechosColados procura dois pontos da linha central c (circuito fechado,
// pontos a cada passo px) que estão longe ao longo da pista mas perto em linha
// reta: aí as faixas de largura se sobrepõem. "Longe" é mais de meia volta de um
// retorno de raio largura/2 (π·largura/2), cujas pontas ficam exatamente a uma
// largura uma da outra; por isso curvas fechadas demais também são recusadas.
func trechosColados(c []Vec2, passo, largura float64) (int, int, bool) {
	n := len(c)
	perto := int(math.Ceil(math.Pi * largura / 2 / passo))
	lim := largura * (1 - 1e-9)
	for i := 0; i < n; i++ {
		for j := i + perto + 1; j < n; j++ {
			if n-(j-i) <= perto {
				break // perto pelo outro lado do circuito
			}
			if c[j].Sub(c[i]).Len() < lim {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

// cruzaEntre diz se algum trecho do circuito fechado a toca algum trecho do circuito fechado b.
func cruzaEntre(a, b []Vec2) (int, int, bool) {
	na, nb := len(a), len(b)
	for i := 0; i < na; i++ {
		for j := 0; j < nb; j++ {
			if segmentosSeTocam(a[i], a[(i+1)%na], b[j], b[(j+1)%nb]) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

// orientacao é o sinal de (q-p) × (r-p): >0 se r está à esquerda de p→q.
func orientacao(p, q, r Vec2) float64 { return q.Sub(p).Cross(r.Sub(p)) }

// noRetangulo diz se r (colinear com p e q) está dentro do retângulo de p a q.
func noRetangulo(p, q, r Vec2) bool {
	return math.Min(p.X, q.X) <= r.X && r.X <= math.Max(p.X, q.X) &&
		math.Min(p.Y, q.Y) <= r.Y && r.Y <= math.Max(p.Y, q.Y)
}

// segmentosSeTocam diz se os segmentos p1p2 e q1q2 têm algum ponto em comum.
func segmentosSeTocam(p1, p2, q1, q2 Vec2) bool {
	d1 := orientacao(q1, q2, p1)
	d2 := orientacao(q1, q2, p2)
	d3 := orientacao(p1, p2, q1)
	d4 := orientacao(p1, p2, q2)
	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) && ((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	return (d1 == 0 && noRetangulo(q1, q2, p1)) ||
		(d2 == 0 && noRetangulo(q1, q2, p2)) ||
		(d3 == 0 && noRetangulo(p1, p2, q1)) ||
		(d4 == 0 && noRetangulo(p1, p2, q2))
}
