import { test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { scrapePocketGamer } from '../../scrapers/pocket-gamer.js';

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

test('scrapePocketGamer extracts and dedupes codes from the working-codes section', async () => {
  const html = `
    <h2>Working Whiteout Survival codes</h2>
    <ul>
      <li>ABC123</li>
      <li>DEF456&nbsp;</li>
      <li><strong>GHI789</strong></li>
      <li>bad code!</li>
      <li>ab</li>
      <li>ABC123</li>
    </ul>
    <h2>Expired Whiteout Survival codes</h2>
    <ul><li>OLD999</li></ul>
  `;
  stubFetch({ ok: true, text: async () => html });

  const codes = await scrapePocketGamer({});

  assert.deepEqual(codes, ['ABC123', 'DEF456', 'GHI789']);
});

test('scrapePocketGamer throws when the working-codes section is missing', async () => {
  stubFetch({ ok: true, text: async () => '<html><body>nothing here</body></html>' });

  await assert.rejects(
    () => scrapePocketGamer({}),
    /Could not find "Working Whiteout Survival codes" section/
  );
});

test('scrapePocketGamer throws on a non-ok HTTP response', async () => {
  stubFetch({ ok: false, status: 404 });

  await assert.rejects(() => scrapePocketGamer({}), /HTTP error! status: 404/);
});
