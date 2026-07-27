# Development Workflow Rules (Strict State Machine)

The development MUST follow this sequential order. Do NOT proceed to execution without fulfilling prior phases:

0. **Phase 0: Branch Setup** -> Create a standardized Git branch before any work begins.
1. **Phase 1: Technical Specification** -> Defined in `.opencode/rules/architecture.md`.
2. **Phase 2: Functional Specification** -> Defined in `docs/specs.md`.
3. **Phase 3: Required Documentation** -> API Contracts & Mocks in `docs/api-contract.md` and `testdata/mocks/`.
4. **Phase 4: General Test Scenarios** -> Manual/Visual CLI mock flag tests in `docs/testing.md`.
5. **Phase 5: Automation Test Scenarios** -> Integration tests (`httptest`) in `docs/testing.md`.
6. **Phase 6: Unit Test Scenarios** -> TDD math logic tests in `docs/testing.md`.
7. **Phase 7: Execution** -> Write production code to pass tests.
8. **Phase 8: Git Finalization** -> Version, merge, and release via CI/CD.

---

## Phase 0: Branch Setup (Mandatory Pre-Step)

Ao iniciar qualquer task (feature ou fix), o PRIMEIRO comando deve ser criar a branch:

```bash
git checkout -b feature/<id>-<slug>   # para features
git checkout -b fix/<id>-<slug>       # para bugfixes
```

### Naming Convention
- `<id>` = ID da task (ex: `003`, `004`) conforme `docs/features/`
- `<slug>` = descrição curta em kebab-case (ex: `judge-tip-language`, `runtime-trigger-config`)

### Exemplos
```
feature/003-judge-tip-language
feature/004-runtime-trigger-config
fix/005-null-pointer-processor
```

### Constraints
- **NUNCA** iniciar trabalho sem criar a branch primeiro.
- **NUNCA** usar nomes genéricos (`feature/test`, `fix/wip`).
- A branch DEVE seguir o padrão para que o CI/CD funcione corretamente.

---

## Phase 8: Git Finalization Rules

### Branch Naming Convention
- Features: `feature/<nome-da-feature>`
- Fixes: `fix/<nome-do-bugfix>`

### Workflow Steps
1. **Push branch** (`feature/*` ou `fix/*`) → CI roda testes e build → PR automático para `main` é criado.
2. **PR para `main`** → Pipeline gera artefatos Windows assinados (`lol-cli.exe`, `lol-daemon.exe`).
3. **PR aprovado e mergeado** → Release automática:
   - Versão bumped via **SemVer**:
     - `feature/*` → **MINOR** bump (ex: `v0.3.0` → `v0.4.0`)
     - `fix/*` → **PATCH** bump (ex: `v0.3.0` → `v0.3.1`)
   - Tag `vX.Y.Z` criada e pushada.
   - GitHub Release criada com **release notes** automáticas.
   - Artefatos Windows assinados e anexados à Release (.zip).

### Code Signing (Setup One-Time)
- Gerar certificado: `bash scripts/gen-cert.sh`
- Adicionar secrets no GitHub:
  - `CODE_SIGN_PFX` = base64 do `.pfx`
  - `CODE_SIGN_PASS` = senha do `.pfx`
- Pipeline assina automaticamente se os secrets existirem.

### Constraints
- **NUNCA** fazer push direto para `main`. Todas as mudanças entram via PR.
- **NUNCA** criar tags manualmente. O CI/CD gerencia versionamento.
- PRs devem ter testes passando antes de merge.