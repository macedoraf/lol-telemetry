# 002-cli-menu-judge-env — Especificação Funcional

## ID
`002-cli-menu-judge-env`

## Objetivo do Produto
Tornar o SDK de League of Legends "bring your own key" para o LLM Judge (via OpenRouter) por meio de variáveis de ambiente e evoluir o executável `lol-cli.exe` para exibir um menu interativo de Features, permitindo ao usuário escolher entre explorar as rotas do SDK ou acompanhar dicas táticas geradas pelo Judge a cada 5 minutos de jogo.

## Contexto
Hoje o CLI abre diretamente no TUI de debugger das rotas da API da Riot e o Judge já está implementado pela feature `001-periodic-judge-hook`. A chave da OpenRouter é lida de `OPENROUTER_API_KEY`, mas o modelo é fixo. O jogador/testador quer poder configurar provedor e modelo sem alterar código, e quer uma CLI com menu para validar separadamente as rotas do SDK e as dicas do Judge, recebendo as hints diretamente no terminal.

## User Stories

- **EU COMO** usuário do SDK  
  **GOSTARIA** de configurar o LLM Judge apenas com variáveis de ambiente  
  **PARA** não precisar alterar código ou recompilar o binário.

- **EU COMO** usuário do `lol-cli.exe`  
  **GOSTARIA** de abrir um menu interativo de Features  
  **PARA** escolher entre validar as rotas do SDK ou acessar as dicas do jogo.

- **EU COMO** jogador/testador  
  **GOSTARIA** de receber dicas/hints do Judge a cada 5 minutos diretamente no terminal  
  **PARA** acompanhar sugestões sem sair da CLI.

- **EU COMO** usuário  
  **GOSTARIA** de visualizar as configurações de ambiente do Judge na tela de dicas  
  **PARA** confirmar que minha chave e modelo estão corretamente carregados.

## Descrição Funcional

### Configuração do Judge por variáveis de ambiente
- `OPENROUTER_API_KEY` (obrigatória): chave da API da OpenRouter.
- `OPENROUTER_MODEL` (opcional): modelo a ser usado. Padrão: `openai/gpt-4o-mini`.
- O Judge só é inicializado quando `OPENROUTER_API_KEY` está presente e não vazia.
- A chave nunca é logada, armazenada ou exposta em arquivos; apenas o prefixo é mostrado na tela de configurações (últimos 4 caracteres) para conferência.

### Menu interativo na CLI
- Ao iniciar o `.exe`, o usuário vê uma tela de menu com pelo menos duas opções:
  - `[Rotas do SDK]` — leva ao TUI de debugger já existente, integrado com as rotas da API da Riot.
  - `[Dicas do Jogo]` — leva a uma tela que mostra as dicas do Judge e as configurações de ambiente.
- Navegação com ↑/↓ ou j/k; Enter para selecionar; Esc/q/Ctrl+C para voltar ou sair.
- A partir de qualquer tela filha, o usuário pode retornar ao menu principal.

### Dicas no terminal
- A feature reutiliza o hook periódico de 5 minutos já implementado (`001-periodic-judge-hook`).
- As dicas são geradas em tempo real pelo LLM Judge a cada marco de 5 minutos do relógio do jogo.
- A dica mais recente é exibida na tela `[Dicas do Jogo]` e também continua disponível no status bar do debugger.
- Se o Judge não estiver configurado (chave ausente), a tela informa que as dicas estão desativadas e indica as variáveis de ambiente necessárias.

## System Prompt do Judge (reutilizado)
> Você é um assistente tático de League of Legends. Sua função é analisar o estado atual da partida e responder com **uma única frase curta e acionável** (máximo 140 caracteres). Foque em macro: objetivos, recalls, power spikes, rotação ou avisos de risco. Não dê explicações longas, não use listas numeradas e não repita dados óbvios. Seja direto, como um coach no ouvido do jogador.

## Critérios de Aceitação

1. **Dado** que `OPENROUTER_API_KEY` está configurada  
   **Quando** o CLI inicia  
   **Então** o Judge é inicializado e as dicas podem ser geradas.

2. **Dado** que `OPENROUTER_MODEL` não está configurada  
   **Quando** o Judge é inicializado  
   **Então** o modelo padrão `openai/gpt-4o-mini` é usado.

3. **Dado** que `OPENROUTER_MODEL` está configurada com `anthropic/claude-3.5-sonnet`  
   **Quando** o Judge envia a requisição  
   **Então** o modelo informado é utilizado.

4. **Dado** que o usuário abre o `.exe`  
   **Quando** a tela inicial é renderizada  
   **Então** o menu de Features exibe `[Rotas do SDK]` e `[Dicas do Jogo]`.

5. **Dado** que o usuário seleciona `[Rotas do SDK]`  
   **Quando** confirma com Enter  
   **Então** o TUI de debugger com as rotas da API da Riot é exibido.

6. **Dado** que o usuário seleciona `[Dicas do Jogo]`  
   **Quando** confirma com Enter  
   **Então** a tela exibe a última dica disponível e as configurações de ambiente do Judge.

7. **Dado** que o hook periódico de 5 minutos dispara  
   **Quando** o Judge retorna uma dica  
   **Então** ela aparece na tela `[Dicas do Jogo]` e no status bar do debugger.

8. **Dado** que `OPENROUTER_API_KEY` não está configurada  
   **Quando** o usuário abre `[Dicas do Jogo]`  
   **Então** uma mensagem informa que as dicas estão desativadas por falta de configuração.

9. **Dado** que as configurações de ambiente são exibidas  
   **Quando** a chave está presente  
   **Então** apenas os últimos 4 caracteres são mostrados, protegendo o restante.

10. **Dado** que o usuário está em qualquer tela filha  
    **Quando** pressiona Esc ou q  
    **Então** ele retorna ao menu principal ou sai do programa, conforme o contexto.

---

_Assinatura de consumo: Produteiro + Techlead — 1 ciclo, ~3.000 tokens estimados._
