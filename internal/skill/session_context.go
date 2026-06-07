package skill

// Built-in skill: session-context — detects when a new conversation touches on a
// topic that already has an active or saved session, and asks whether the user
// would prefer to resume that session instead of starting fresh.
//
// The skill body is inlined for the model to follow as a playbook, but the
// controller also runs the check automatically on every first turn (see
// controller.checkSessionContext), so the user gets prompted even when the
// model doesn't invoke the skill.
const builtinSessionContextBody = `This skill is INLINED — you run in the parent loop. The user started a new session with a question that may overlap an existing conversation. Your job is to check for matching sessions and, when found, ask whether to resume one instead of starting fresh.

How to operate:
1. First, confirm this is a new session: check if History() is empty of user messages (only system prompt exists).
2. List saved sessions from the session directory: find *.jsonl files under the session dir (check the config for the active session directory — typically ~/.reasonix/sessions/).
3. Read each session's first user message to compare with the current question.
4. If one or more sessions match by topic (significant keyword overlap or same topic_id):
   - Present options to the user via the ask tool:
     - "Start fresh in this new session" (the default)
     - "Switch to [session preview]" for each close match
5. If no match: proceed normally without prompting.
6. If the user chooses to switch: return the path and session details so the parent can load it.

Rules:
- Be precise, not noisy — only prompt on a genuine topic match.
- Ignore sessions older than 7 days unless the user asks about an old topic.
- Always include the session preview, turn count, and last-activity time in options.
- If the current message is a slash command (/clear, /new, /resume, etc.), skip the check entirely.

Your final answer: Present the matching sessions or confirm no match was found.`
