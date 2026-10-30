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
