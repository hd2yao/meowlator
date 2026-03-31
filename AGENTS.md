# AGENTS.md

## Commit Strategy (Required)
- Make small, reviewable commits for every code update.
- Create a commit whenever one independent rollback unit is completed.
- Never mix unrelated changes in one commit.
- Use clear commit messages with scope and intent.

## End-of-Conversation Gate (Required)
- Run a code review before ending any coding task conversation.
- Review must cover regression risk, error handling, edge cases, and tests.
- If any blocking issue is found, do not end the conversation.
- Keep fixing and re-reviewing until no blocking issues remain.

## Execution Loop
1. Implement a small change.
2. Run the fastest relevant check.
3. Commit the change.
4. Repeat.
5. Before ending, run code review.
6. If review fails, return to step 1. End only after review passes.
