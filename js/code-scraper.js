#!/usr/bin/env node
import fs from 'fs';
import { argv } from 'process';
import { scrapePocketGamer } from './scrapers/pocket-gamer.js';
import { scrapeWosRewards } from './scrapers/wos-rewards.js';
import { scrapeWosCommunity } from './scrapers/wos-community.js';

// Simple command-line argument parser for -o <file>
export function parseArgs(args = argv.slice(2)) {
  let outFile = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '-o') {
      if (i + 1 < args.length) {
        outFile = args[i + 1];
        i++;
      } else {
        console.error('Error: -o option requires a file path argument.');
        process.exit(1);
      }
    }
  }

  return { outFile };
}

// Merge codes from all sources: any code marked expired by a source is
// dropped even if another source still lists it as active.
export function combineCodes({
  pgCodes = [],
  wosRewardsActive = [],
  wosRewardsExpired = [],
  wosCommunityActive = [],
  wosCommunityExpired = []
}) {
  const activeSet = new Set([...pgCodes, ...wosRewardsActive, ...wosCommunityActive]);

  const expiredSet = new Set(
    [...wosRewardsExpired, ...wosCommunityExpired].map(c => c.toLowerCase())
  );

  const finalCodes = [];
  for (const code of activeSet) {
    if (!expiredSet.has(code.toLowerCase())) {
      finalCodes.push(code);
    }
  }

  return finalCodes;
}

async function main() {
  const { outFile } = parseArgs();

  const headers = {
    'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36'
  };

  let pgCodes = [];
  try {
    pgCodes = await scrapePocketGamer(headers);
  } catch (error) {
    console.error(`Warning: Failed to scrape PocketGamer: ${error.message}`);
  }

  let wosRewardsActive = [];
  let wosRewardsExpired = [];
  try {
    const wosRewards = await scrapeWosRewards(headers);
    wosRewardsActive = wosRewards.active;
    wosRewardsExpired = wosRewards.expired;
  } catch (error) {
    console.error(`Warning: Failed to scrape WOS Rewards: ${error.message}`);
  }

  let wosCommunityActive = [];
  let wosCommunityExpired = [];
  try {
    const wosCommunity = await scrapeWosCommunity(headers);
    wosCommunityActive = wosCommunity.active;
    wosCommunityExpired = wosCommunity.expired;
  } catch (error) {
    console.error(`Warning: Failed to scrape WOS Community: ${error.message}`);
  }

  const finalCodes = combineCodes({
    pgCodes,
    wosRewardsActive,
    wosRewardsExpired,
    wosCommunityActive,
    wosCommunityExpired
  });

  if (finalCodes.length === 0) {
    console.error('Error: No active gift codes found across sources.');
    process.exit(1);
  }

  // Map to required JSON array format
  const codeObjects = finalCodes.map(code => ({
    code,
    'valid-to': null
  }));

  const jsonOutput = JSON.stringify(codeObjects, null, 2);

  if (outFile) {
    fs.writeFileSync(outFile, jsonOutput, 'utf8');
  } else {
    console.log(jsonOutput);
  }
}

const isMainModule = argv[1] && import.meta.url === `file://${argv[1]}`;
if (isMainModule) {
  main();
}
