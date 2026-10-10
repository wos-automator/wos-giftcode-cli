import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parseArgs, combineCodes } from '../code-scraper.js';

test('parseArgs returns null outFile when -o is not given', () => {
  assert.deepEqual(parseArgs([]), { outFile: null });
});

test('parseArgs picks up the file path following -o', () => {
  assert.deepEqual(parseArgs(['-o', 'codes.json']), { outFile: 'codes.json' });
});

test('parseArgs ignores unrelated arguments', () => {
  assert.deepEqual(parseArgs(['--verbose', '-o', 'out.json', 'extra']), { outFile: 'out.json' });
});

test('combineCodes merges active codes from all sources and dedupes', () => {
  const result = combineCodes({
    pgCodes: ['CODE1', 'CODE2'],
    wosRewardsActive: ['CODE2', 'CODE3'],
    wosCommunityActive: ['CODE4']
  });

  assert.deepEqual(new Set(result), new Set(['CODE1', 'CODE2', 'CODE3', 'CODE4']));
  assert.equal(result.length, 4);
});

test('combineCodes drops codes marked expired by any source, case-insensitively', () => {
  const result = combineCodes({
    pgCodes: ['ActiveCode', 'expiredcode'],
    wosRewardsActive: ['AnotherActive'],
    wosRewardsExpired: ['EXPIREDCODE'],
    wosCommunityActive: ['StillActive', 'GoneToo'],
    wosCommunityExpired: ['gonetoo']
  });

  assert.deepEqual(
    new Set(result),
    new Set(['ActiveCode', 'AnotherActive', 'StillActive'])
  );
});

test('combineCodes handles missing source fields', () => {
  assert.deepEqual(combineCodes({}), []);
});
