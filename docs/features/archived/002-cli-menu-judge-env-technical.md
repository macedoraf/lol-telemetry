# 002-cli-menu-judge-env — Especificação Técnica

## 1. Visão Geral da Solução

A feature evolui o entrypoint do CLI (`cmd/lol-cli/main.go`) para apresentar um menu de Features antes de entrar em uma das duas visões filhas: (a) debugger de rotas do SDK (existente) e (b) painel de dicas do Judge. Ela também torna o modelo da OpenRouter configurável via variável de ambiente `OPENROUTER_MODEL`, mantendo o SDK no padrão BYOK.

## 2. Pacotes e Responsabilidades

```text
lol-telemetry/
├── cmd/lol-cli/
│   └── main.go                      # entrypoint; wiring apenas
├── internal/
│   ├── menu/                        # menu principal de Features
│   │   ├── model.go                 # Bubble Tea model do menu
│   │   └── model_test.go            # testes unitários do menu
│   ├── tips/                        # painel de dicas do Judge
│   │   ├── model.go                 # Bubble Tea model das dicas
│   │   └── model_test.go            # testes unitários das dicas
│   ├── judge/
│   │   └── openrouter/
│   │       └── client.go            # lê OPENROUTER_MODEL via env
│   └── renderer/                    # debugger existente (sem lógica de menu)
└── docs/features/
    └── 002-cli-menu-judge-env-*     # especificações
```

### Responsabilidades detalhadas

| Pacote | Responsabilidade |
|--------|------------------|
| `internal/menu` | Renderizar o menu de Features, capturar input e emitir mensagem de seleção. |
| `internal/tips` | Renderizar a última dica, status do Judge e configurações de ambiente (chave parcialmente mascarada). |
| `internal/judge/openrouter` | Inicializar `Client` lendo `OPENROUTER_API_KEY` e `OPENROUTER_MODEL` do ambiente. |
| `cmd/lol-cli/main.go` | Criar o menu, as visões filhas e o orquestrador; encaminhar mensagens de dicas entre orquestrador e a visão ativa. |

## 3. Interfaces Principais

### EnvConfig
```go
type EnvConfig struct {
    APIKey string
    Model  string
}

func LoadEnvConfig() EnvConfig
func (c EnvConfig) Enabled() bool
func (c EnvConfig) MaskedKey() string
```

### menu.Model
```go
type Model struct {
    choices  []string
    cursor   int
    selected string
}

func NewModel() Model
```

Mensagem emitida:
```go
type SelectMsg struct {
    Choice string
}
```

### tips.Model
```go
type Model struct {
    cfg      EnvConfig
    advice   string
    width    int
    height   int
}

func NewModel(cfg EnvConfig) Model
```

Recebe:
```go
type UpdateAdviceMsg struct {
    Advice string
}
```

## 4. Fluxo de Dados Completo

1. **Inicialização**: `main.go` lê `EnvConfig` e cria `menu.Model`, `renderer.Model` (debugger) e `tips.Model`.
2. **Top-level app**: `main.go` envolve as três visões em um modelo mestre (`appModel`) que gerencia qual sub-modelo está ativo.
3. **Orquestrador**: se a configuração estiver habilitada, `main.go` cria e inicia o loop de Judge já existente, com um ticker de 5 segundos.
4. **AdviceMsg**: quando o orquestrador produz uma dica, ele envia `renderer.AdviceMsg` (ou equivalente) para o programa Bubble Tea.
5. **Roteamento**: o modelo mestre recebe a dica e encaminha para o `tips.Model` e para o `renderer.Model` (debugger) simultaneamente.
6. **Navegação**: `menu.Model` emite `SelectMsg`; o modelo mestre altera `activeView` para `"routes"` ou `"tips"`.
7. **Retorno**: nas visões filhas, `Esc`/`q` volta ao menu principal.

```
[main.go] --cria--> [appModel] --ativa--> [menu.Model]
                              |            | (SelectMsg)
                              |            v
                              |--ativa--> [renderer.Model] (debugger)
                              |--ativa--> [tips.Model]
                              ^
[orchestrator] --AdviceMsg--> [appModel]
```

## 5. Integração com Código Existente

### O que reutilizar
- `renderer.Model` e `renderer.AdviceMsg` existentes para o debugger de rotas.
- `internal/hooks`, `internal/judge`, `internal/judge/payload`, `internal/orchestrator` para o loop de dicas.
- `pkg/riotclient.Client` para os dados de jogo.

### O que modificar
- **`internal/judge/openrouter/client.go`**: adicionar `OPENROUTER_MODEL` ao construtor, permitindo override via env.
- **`cmd/lol-cli/main.go`**: substituir a criação direta do `renderer.Model` por um `appModel` que contém menu, debugger e tips.
- **Novo `internal/menu`**: implementar o menu Bubble Tea.
- **Novo `internal/tips`**: implementar o painel de dicas Bubble Tea.

### O que não alterar
- `pkg/riotclient` continua 100% desacoplado de `internal/` e `cmd/`.
- A lógica de cálculo do `processor` permanece isolada.
- O loop de 5 minutos do Judge permanece no `orchestrator` e no hook periódico já existente.

## 6. Estratégia de Testes

### Unitários (in-memory, sem rede)
- `internal/menu`: navegação ↑/↓, seleção Enter e mensagem emitida.
- `internal/tips`: renderização de dica, mensagem de desativado quando sem chave, mascaramento da chave.
- `internal/judge/openrouter`: construção do client com `OPENROUTER_MODEL` padrão e customizado.

### Integração
- `httptest` para o OpenRouter validando que o modelo enviado no body corresponde a `OPENROUTER_MODEL`.
- Fluxo do `appModel` com mocks: menu -> select -> tips -> recebe dica.

### Mocks
- `mockGameDataProvider` e `mockLLMClient` existentes para o orquestrador.
- Mock de `tea.Model` não é necessário; os testes usam as mensagens Bubble Tea.

## 7. Dependências

Nenhuma dependência nova. Reutilizam-se:
- `github.com/charmbracelet/bubbletea` para o menu e as visões filhas.
- `github.com/charmbracelet/lipgloss` para estilização.
- `net/http` e `encoding/json` nativos para o client OpenRouter.

## 8. Critérios Técnicos de Aceitação

1. `OPENROUTER_MODEL` é lida do ambiente; quando ausente, o modelo padrão é `openai/gpt-4o-mini`.
2. `OPENROUTER_API_KEY` continua sendo a única variável obrigatória; quando ausente, o Judge não é inicializado.
3. O menu de Features é o primeiro modelo exibido ao iniciar o CLI.
4. A seleção `[Rotas do SDK]` ativa o debugger existente sem perder as rotas ou o status bar.
5. A seleção `[Dicas do Jogo]` ativa o painel de dicas, mostrando a última dica e a configuração de ambiente.
6. A dica do Judge é roteada para ambas as visões (debugger e tips) quando a visão ativa a suporta.
7. A chave da API é mascarada na UI: `OPENROUTER_API_KEY=sk...abcd` (últimos 4 caracteres).
8. `Esc`/`q` em qualquer visão filha retorna ao menu; no menu, sai do programa.
9. `cmd/lol-cli/main.go` continua com zero lógica de negócio; apenas wiring e roteamento de mensagens.
10. Todos os testes unitários rodam sem chamadas de rede reais.

---

_Assinatura de consumo: Techlead — 1 ciclo, ~2.500 tokens estimados._
