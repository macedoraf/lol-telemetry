# 🛡️ Guardrails & Operational Rules (Open-Spec Aligned)
*(Regras universais de contensão e segurança operacional aplicadas a TODOS os agentes).*

## 1. Prevenção de Loops Infinitos & Custo (Infinite Loop & Cost Guardrails)
- **Limite de Tentativas (Fail-Fast):** Máximo de **3 (três) tentativas** para qualquer artefato, teste ou correção de erro. Atingido o limite, mude o status para `BLOCKED`, documente o motivo no `state.json` e encerre imediatamente a execução.
- **Proibição de Poluição de Logs no Bash:** Ao rodar comandos de compilação ou testes, execute apenas o escopo estritamente necessário (ex: testes focados). Ocultar ou direcionar saídas massivas para evitar estouro da janela de contexto.
- **Edição Incremental Obrigatória:** Proibido reescrever arquivos inteiros via sobrescrita total quando a alteração for pontual. Use ferramentas de patch/edição de blocos (`edit`).

## 2. Auditoria e Telemetria Obrigatória (Compliance & Telemetry)
- **Assinatura de Entrega:** Todo arquivo de documentação gerado em `/docs` deve encerrar com a assinatura de consumo correspondente:
  ```markdown
  ---
  *Gerado por [Nome do Agente] | Tokens estimados: ~X.XXX | Ciclos: X*