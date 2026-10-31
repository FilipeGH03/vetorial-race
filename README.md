# Corrida Vetorial

Jogo de corrida de carros controlados por vetores (Álgebra Linear), com adversários
guiados por rede neural treinada por neuroevolução. Escrito em Go com Ebitengine v2.

## Instalar o Go

Precisa do Go 1.27 ou mais novo. Não precisa de compilador C no Windows.

- Windows: `winget install GoLang.Go` (ou baixe em https://go.dev/dl/). Depois **abra um
  terminal novo** para o `go` entrar no PATH (instalação padrão em `C:\Program Files\Go\bin`).
- Confira com `go version`.

## Rodar

```
go run ./cmd/jogo
```

Abre uma janela 1280x720 a 60 TPS com a pista `pistas/oval.json` e o carro na largada.
Dirija com WASD ou as setas (duas teclas juntas dão a diagonal); bater na parede para o carro.
Outra pista: `go run ./cmd/jogo -pista pistas/outra.json`.
Na primeira vez o `go run` baixa as dependências sozinho.

## Editor de pista

```
go run ./cmd/jogo -editar pistas/minha.json
```

Arraste o mouse com o botão esquerdo para desenhar a linha central (o circuito fecha sozinho e o
sentido é o do traço). Roda do mouse ou setas cima/baixo mudam a largura (40 a 200 px). `S` salva no
arquivo, `L` recarrega, `C` ou `Esc` limpam. Se o arquivo já existe, ele é aberto para editar.
Pista inválida (por exemplo um oito) mostra o erro e marca o cruzamento; a pista só cabe em 1280x720.
Depois: `go run ./cmd/jogo -pista pistas/minha.json`.

## Treino da IA

```
go run ./cmd/treino -pista pistas/oval.json -seed 1
```

Roda a neuroevolução sem janela: 50 redes por geração, 300 gerações (`-geracoes n` muda). Cada rede
dirige sozinha 6 vezes, uma de cada posição da largada de 6 carros, por até 60 s cada, e a nota é a
média das 6. Bater na parede zera a velocidade e devolve o carro à última posição válida (como na
corrida), custa 5 pontos, e a volta é encerrada se passar 1 s (60 ticks) sem avançar. A nota de cada
largada é a distância percorrida + 1000 por volta + 20 por segundo poupado na 1ª volta − 5 por
batida. A rede vê 14 entradas: 7 sensores em direções fixas do mundo, a velocidade (x, y e módulo) e
a direção da pista à frente. Imprime a melhor nota de cada geração, a geração da 1ª volta completa
(e a da 1ª vez que o melhor completou a volta das 6 largadas) e grava a melhor rede em
`cerebros/<arquivo da pista sem extensão>.json` (ou em `-saida arq`). A mesma seed dá sempre o mesmo
resultado. Outras opções: `-camadas 14,16,2` (entrada 14 e saída 2) e `-populacao n`.

## Verificar

```
go build ./...
go vet ./...
go test ./...
```

## Estrutura

| pasta | conteúdo |
|---|---|
| `cmd/jogo` | executável do jogo |
| `cmd/treino` | executável do treino da rede (sem janela) |
| `internal/sim` | simulação: física, pista, colisão (sem Ebitengine) |
| `internal/jogo` | tela, entrada e desenho |
| `internal/ia` | rede neural e treino |
| `assets` | imagens (PNG embutido com `go:embed`) |
| `pistas` | pistas em JSON |
| `cerebros` | redes treinadas em JSON |

Convenções: coordenadas da simulação são matemáticas (y para cima); o desenho converte
com `telaY = altura - mundoY`. Passo fixo de 60 ticks por segundo.

## Sprite do carro

`assets/CarroAzul.svg` é o original. O Ebitengine não carrega SVG, então ele foi convertido
uma vez para `assets/carro_azul.png` (128 px de comprimento, fundo transparente, frente para +x)
com o `sharp` do Node (`resize(128)` sobre o SVG com largura/altura explícitas 960x476). Só
é preciso refazer se o SVG mudar.
