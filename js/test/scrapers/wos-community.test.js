import { test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { scrapeWosCommunity } from '../../scrapers/wos-community.js';

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

test('scrapeWosCommunity splits codes into active and expired by data-status', async () => {
  const html = `
    <article class="woscGiftTile" data-status="active" aria-label="Gift code JPsilverweek26, active">
      <div class="woscGiftTile__codeRow">
        <code class="woscEmbed__code" translate="no">JPsilverweek26</code>
        <button data-code="JPsilverweek26">Copy</button>
      </div>
    </article>
    <article class="woscGiftTile" data-status="expired" aria-label="Gift code OldCode25, expired">
      <div class="woscGiftTile__codeRow">
        <code class="woscEmbed__code" translate="no">OldCode25</code>
        <button data-code="OldCode25">Copy</button>
      </div>
    </article>
  `;
  stubFetch({ ok: true, text: async () => html });

  const { active, expired } = await scrapeWosCommunity({});

  assert.deepEqual(active, ['JPsilverweek26']);
  assert.deepEqual(expired, ['OldCode25']);
});

test('scrapeWosCommunity returns empty lists when no codes are present', async () => {
  stubFetch({ ok: true, text: async () => '<html><body>no codes</body></html>' });

  const { active, expired } = await scrapeWosCommunity({});

  assert.deepEqual(active, []);
  assert.deepEqual(expired, []);
});

test('scrapeWosCommunity throws on a non-ok HTTP response', async () => {
  stubFetch({ ok: false, status: 503 });

  await assert.rejects(() => scrapeWosCommunity({}), /HTTP error! status: 503/);
});
