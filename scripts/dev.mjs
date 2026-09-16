import { spawn } from 'node:child_process';
// Frontend-only shell development. Build and run Go for authenticated API testing.
const child = spawn(process.execPath, ['node_modules/.bin/astro', 'dev', ...process.argv.slice(2)], {
  stdio: 'inherit',
  env: process.env,
});
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => child.kill(signal));
}
child.on('error', (error) => {
  console.error(error);
  process.exitCode = 1;
});
child.on('exit', (code, signal) => {
  process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 143);
});
