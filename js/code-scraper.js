#!/usr/bin/env node
import fs from 'fs';
import { argv } from 'process';

const POCKET_GAMER_URL = 'https://www.pocketgamer.com/whiteout-survival/codes/';
const WOS_REWARDS_URL = 'https://wosrewards.com';

// Simple command-line argument parser for -o <file>
function parseArgs() {
  const args = argv.slice(2);
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

async function scrapePocketGamer(headers) {
  const res = await fetch(POCKET_GAMER_URL, { headers });
  if (!res.ok) {
    throw new Error(`HTTP error! status: ${res.status}`);
  }
  
  const html = await res.text();
  const workingIndex = html.indexOf('Working Whiteout Survival codes');
  if (workingIndex === -1) {
    throw new Error('Could not find "Working Whiteout Survival codes" section on the page.');
  }

  const expiredIndex = html.indexOf('Expired', workingIndex);
  const endSliceIndex = expiredIndex !== -1 ? expiredIndex : html.length;
  const section = html.substring(workingIndex, endSliceIndex);

  const liRegex = /<li>(.*?)<\/li>/gi;
  const codes = [];
  let match;

  while ((match = liRegex.exec(section)) !== null) {
    let code = match[1];
    // Clean up HTML entities and tags
    code = code.replace(/&nbsp;/g, ' ');
    code = code.replace(/<[^>]*>/g, '');
    code = code.trim();
    
    // Whiteout Survival codes do not contain spaces.
    const cleanedCode = code.replace(/\s+/g, '');
    
    // Verify alphanumeric format
    if (cleanedCode && /^[a-zA-Z0-9/]+$/.test(cleanedCode) && cleanedCode.length >= 3 && cleanedCode.length <= 30) {
      codes.push(cleanedCode);
    }
  }

  return [...new Set(codes)];
}

async function scrapeWosRewards(headers) {
  const res = await fetch(WOS_REWARDS_URL, { headers });
  if (!res.ok) {
    throw new Error(`HTTP error! status: ${res.status}`);
  }
  
  const html = await res.text();
  const regex = /absolute top-2 right-2 bg-(?:green|red)-\d+[\s\S]*?(ACTIVE|EXPIRED)[\s\S]*?<h3[^>]*>([a-zA-Z0-9/]+)<\/h3>/gi;
  
  let match;
  const active = [];
  const expired = [];

  while ((match = regex.exec(html)) !== null) {
    const status = match[1].toUpperCase();
    const code = match[2].trim();
    
    if (status === 'ACTIVE') {
      active.push(code);
    } else if (status === 'EXPIRED') {
      expired.push(code);
    }
  }

  return { active, expired };
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

  // Combine and deduplicate active codes
  const activeSet = new Set([...pgCodes, ...wosRewardsActive]);

  // Remove codes marked expired by wosrewards.com (case-insensitive check)
  const expiredSet = new Set(wosRewardsExpired.map(c => c.toLowerCase()));
  const finalCodes = [];

  for (const code of activeSet) {
    if (!expiredSet.has(code.toLowerCase())) {
      finalCodes.push(code);
    }
  }

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

main();
