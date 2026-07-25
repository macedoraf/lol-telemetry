# Development Workflow Rules (Strict State Machine)

The development MUST follow this sequential order. Do NOT proceed to execution without fulfilling prior phases:

1. **Phase 1: Technical Specification** -> Defined in `.opencode/rules/architecture.md`.
2. **Phase 2: Functional Specification** -> Defined in `docs/specs.md`.
3. **Phase 3: Required Documentation** -> API Contracts & Mocks in `docs/api-contract.md` and `testdata/mocks/`.
4. **Phase 4: General Test Scenarios** -> Manual/Visual CLI mock flag tests in `docs/testing.md`.
5. **Phase 5: Automation Test Scenarios** -> Integration tests (`httptest`) in `docs/testing.md`.
6. **Phase 6: Unit Test Scenarios** -> TDD math logic tests in `docs/testing.md`.
7. **Phase 7: Execution** -> Write production code to pass tests.