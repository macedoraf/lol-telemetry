# OpenCode Skill: Feature Workflow Automation

## Purpose
Guide the OpenCode agent through implementing a specific feature following the strict SDLC State Machine, ensuring Human-in-the-Loop (HITL) checkpoints for documentation review, maintaining synchronization with the tracking state, and updating the README.

## Execution Steps
1. **Ask for specification:** Create `docs/features/specs.md`and target feature status (`PENDING`).
2. **Review the specifications and make suggestions:** Improove the specifications, make suggestions for themes.
3. **Human-in-the-Loop Documentation Review (HITL):** 
   - Present the implementation plan and technical specifications for the current feature to the user.
   - **Pause and wait for user approval** before writing tests or code. Do not proceed until explicit human confirmation or feedback is provided.
4. **Done documentation:**: Finish the documentation for that feature and ask for /execute-feature {feature-name}