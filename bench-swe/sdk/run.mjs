// A thin bridge: SDK-native events remain unchanged for auditing and analysis.
import { readFileSync } from 'node:fs';
const req = JSON.parse(readFileSync(0, 'utf8'));
const emit = (event) => process.stdout.write(`${JSON.stringify(event)}\n`);
try {
  if (req.provider === 'claude') {
    const { query } = await import('@anthropic-ai/claude-agent-sdk');
    for await (const event of query({
      prompt: req.prompt,
      options: {
        cwd: req.cwd,
        pathToClaudeCodeExecutable: req.executable,
        ...(req.model ? { model: req.model } : {}),
        effort: 'high',
        systemPrompt: { type: 'preset', preset: 'claude_code' },
        stderr: (data) => process.stderr.write(data),
        mcpServers: req.servers,
        strictMcpConfig: true,
        settingSources: [],
        persistSession: false,
        permissionMode: 'acceptEdits',
        ...(req.readOnly ? { tools: [] } : {}),
        allowedTools: ['Bash', 'Read', 'Edit', 'Write', 'Glob', 'Grep', 'mcp__plugin_lumen_lumen__*'],
        plugins: req.pluginDir ? [{ type: 'local', path: req.pluginDir }] : [],
      },
    })) emit(event);
  } else if (req.provider === 'codex') {
    const { Codex } = await import('@openai/codex-sdk');
    const codex = new Codex({ codexPathOverride: req.executable, configOverrides: req.codexOverrides });
    const thread = codex.startThread({
      ...(req.model ? { model: req.model } : {}),
      workingDirectory: req.cwd,
      sandboxMode: req.readOnly ? 'read-only' : 'workspace-write',
      skipGitRepoCheck: !!req.readOnly,
      approvalPolicy: 'never',
      modelReasoningEffort: 'high',
      networkAccessEnabled: !req.readOnly,
      webSearchMode: 'disabled',
    });
    const { events } = await thread.runStreamed(req.prompt);
    for await (const event of events) emit(event);
  } else throw new Error(`Unknown provider: ${req.provider}`);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
}
