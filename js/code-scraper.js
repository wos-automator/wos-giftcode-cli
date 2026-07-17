#!/usr/bin/env node
import fs from 'fs';
import { argv } from 'process';

const POCKET_GAMER_URL = 'https://www.pocketgamer.com/whiteout-survival/codes/';

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

async function main() {
  const { outFile } = parseArgs();

  const headers = {
    'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36'
  };

  try {
    const res = await fetch(POCKET_GAMER_URL, { headers });
    if (!res.ok) {
      throw new Error(`HTTP error! status: ${res.status}`);
    }
    
    const html = await res.text();
    const workingIndex = html.indexOf('Working Whiteout Survival codes');
    if (workingIndex === -1) {
      console.error('Error: Could not find "Working Whiteout Survival codes" section on the page.');
      process.exit(1);
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

    // Deduplicate
    const uniqueCodes = [...new Set(codes)];

    if (uniqueCodes.length === 0) {
      console.error('Error: No active gift codes found.');
      process.exit(1);
    }

    // Map to required JSON array format: {code: "cdk", valid-to: null}
    const codeObjects = uniqueCodes.map(code => ({
      code,
      'valid-to': null // Expiration time is not exposed on the website, so default to null
    }));

    const jsonOutput = JSON.stringify(codeObjects, null, 2);

    if (outFile) {
      fs.writeFileSync(outFile, jsonOutput, 'utf8');
    } else {
      console.log(jsonOutput);
    }
  } catch (error) {
    console.error(`Error executing scraper: ${error.message}`);
    process.exit(1);
  }
}

main();
