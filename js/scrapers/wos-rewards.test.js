import { test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { scrapeWosRewards } from './wos-rewards.js';

const originalFetch = globalThis.fetch;

function stubFetch(response) {
  globalThis.fetch = async () => response;
}

beforeEach(() => {
  globalThis.fetch = undefined;
});

afterEach(() => {
  globalThis.fetch = originalFetch;
});

test('scrapeWosRewards splits codes into active and expired', async () => {
  const html = `
    <div class="card">
      <div class="absolute top-2 right-2 bg-green-500 text-white">ACTIVE</div>
      <h3 class="code-title">CODE1</h3>
    </div>
    <div class="card">
      <div class="absolute top-2 right-2 bg-red-500 text-white">EXPIRED</div>
      <h3 class="code-title">CODE2</h3>
    </div>
  `;
  stubFetch({ ok: true, text: async () => html });

  const { active, expired } = await scrapeWosRewards({});

  assert.deepEqual(active, ['CODE1']);
  assert.deepEqual(expired, ['CODE2']);
});

test('scrapeWosRewards returns empty lists when nothing matches', async () => {
  stubFetch({ ok: true, text: async () => '<html><body>no codes</body></html>' });

  const { active, expired } = await scrapeWosRewards({});

  assert.deepEqual(active, []);
  assert.deepEqual(expired, []);
});

test('scrapeWosRewards throws on a non-ok HTTP response', async () => {
  stubFetch({ ok: false, status: 500 });

  await assert.rejects(() => scrapeWosRewards({}), /HTTP error! status: 500/);
});
