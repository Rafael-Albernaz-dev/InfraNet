import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function validateSceneCoverage(scenes, referenceDir) {
  if (!scenes.length) throw Error('Lista de cenas vazia');
  if (
    new Set(scenes.map((s) => s.id)).size !== scenes.length ||
    new Set(scenes.map((s) => s.reference)).size !== scenes.length
  ) {
    throw Error('Cenas ou referências duplicadas');
  }
  for (const scene of scenes) {
    for (const suffix of ['claro', 'escuro']) {
      const target = path.join(referenceDir, `${scene.reference}-${suffix}.png`);
      if (!fs.existsSync(target)) throw Error(`Referência ausente: ${target}`);
    }
  }
}

export function checkLayoutGuard() {
  function check(dir) {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const file = path.join(dir, item.name);
      if (item.isDirectory()) check(file);
      else if (
        file.endsWith('.css') &&
        file !== 'frontend/src/shared/ui/tokens.css' &&
        /#[\da-f]{3,8}\b/i.test(fs.readFileSync(file, 'utf8'))
      ) {
        throw Error(`Cor fora dos tokens: ${file}`);
      }
    }
  }
  check('frontend/src');
  const kit = fs.readFileSync('frontend/src/shared/ui/kit.css', 'utf8');
  for (const state of [
    'field-error',
    'inline-error',
    'operation-error',
    'empty-state',
  ]) {
    if (!new RegExp(`\\.${state}\\s*\\{`).test(kit)) {
      throw Error(`Estado sem regra no kit: ${state}`);
    }
  }
  console.log('layout guard: PASS');
}

if (process.argv[1] === fileURLToPath(import.meta.url)) checkLayoutGuard();
