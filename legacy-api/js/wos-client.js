#!/usr/bin/env node
// WOS Gift-Code Redemption Client (reverse engineered)
// Usage: node wos-client.js [-u userid] [giftcode] [-f codes.json]
import crypto from 'crypto';
import fs from 'fs';
import os from 'os';
import path from 'path';
import readline from 'readline';
import { exec, execSync } from 'child_process';
import { fileURLToPath } from 'url';

const SALT = 'tB87#kPtkxqOS2';
const BASE = 'https://wos-giftcode-api.centurygame.com/api';
const HEADERS = {
  'Content-Type': 'application/x-www-form-urlencoded',
  'Origin': 'https://wos-giftcode.centurygame.com',
  'Referer': 'https://wos-giftcode.centurygame.com/',
  'User-Agent':
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ' +
    '(KHTML, like Gecko) Chrome/126.0 Safari/537.36',
};

// Replicates appendSign() from src/utils/request.js
function appendSign(data) {
  const str = Object.keys(data)
    .sort()
    .reduce((acc, k) => {
      const v =
        typeof data[k] === 'object'
          ? JSON.stringify(data[k])
          : encodeURIComponent(data[k]);
      return (acc ? acc + '&' : '') + k + '=' + v;
    }, '');
  const sign = crypto.createHash('md5').update(str + SALT).digest('hex');
  return { sign, ...data };
}

async function api(path, data) {
  const res = await fetch(BASE + path, {
    method: 'POST',
    headers: HEADERS,
    body: new URLSearchParams(appendSign(data)).toString(),
  });
  return res.json();
}

// --- Endpoints ---
const getPlayer = (fid) => api('/player', { fid, time: Date.now() });
const getCaptcha = (fid) => api('/captcha', { fid, time: Date.now(), init: 0 });
const redeem = (fid, cdk, captcha_code) =>
  api('/gift_code', { fid, cdk, captcha_code, time: Date.now() });

// Open a file with the OS default application (browser for images/html).
function openInBrowser(filePath) {
  const cmd =
    process.platform === 'darwin'
      ? `open "${filePath}"`
      : process.platform === 'win32'
      ? `start "" "${filePath}"`
      : `xdg-open "${filePath}"`;
  exec(cmd, (err) => {
    if (err) console.error('Could not open file automatically:', err.message);
  });
}

// Prompt the user on the command line and resolve with the entered text.
function prompt(question) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

// --- State Cache Management ---
const stateFilePath = path.join(os.homedir(), '.wos-redeemed-codes.json');

function loadState() {
  try {
    if (fs.existsSync(stateFilePath)) {
      const data = fs.readFileSync(stateFilePath, 'utf8');
      return JSON.parse(data);
    }
  } catch (e) {
    // Ignore read errors, return empty
  }
  return {};
}

function saveState(state) {
  try {
    fs.writeFileSync(stateFilePath, JSON.stringify(state, null, 2), 'utf8');
  } catch (e) {
    console.error('Error saving state cache:', e.message);
  }
}

function isRedeemed(state, fid, code) {
  const codes = state[fid] || [];
  return codes.some(c => c.toLowerCase() === code.toLowerCase());
}

function markRedeemed(state, fid, code) {
  if (!state[fid]) {
    state[fid] = [];
  }
  if (!isRedeemed(state, fid, code)) {
    state[fid].push(code);
  }
}

// --- Input parsing helpers ---
function parseArgs() {
  const args = process.argv.slice(2);
  let fid = null;
  let cdk = null;
  let codesFile = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '-u') {
      if (i + 1 < args.length) {
        fid = args[i + 1];
        i++;
      }
    } else if (args[i] === '-f') {
      if (i + 1 < args.length) {
        codesFile = args[i + 1];
        i++;
      }
    } else {
      if (!fid && !args[i].startsWith('-')) {
        fid = args[i];
      } else if (!cdk && !args[i].startsWith('-')) {
        cdk = args[i];
      }
    }
  }

  return { fid, cdk, codesFile };
}

function readCodes(codesFile) {
  return new Promise((resolve, reject) => {
    if (codesFile === '-') {
      let data = '';
      process.stdin.on('data', chunk => {
        data += chunk;
      });
      process.stdin.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch (e) {
          reject(new Error(`Failed to parse JSON from stdin: ${e.message}`));
        }
      });
      process.stdin.on('error', (err) => {
        reject(err);
      });
    } else {
      try {
        const data = fs.readFileSync(codesFile, 'utf8');
        resolve(JSON.parse(data));
      } catch (e) {
        reject(new Error(`Failed to read file ${codesFile}: ${e.message}`));
      }
    }
  });
}

(async () => {
  const DEFAULT_FID = '123456789';
  let { fid, cdk, codesFile } = parseArgs();

  // If no explicit cdk or codesFile, but stdin is piped, default codesFile to '-'
  const isPipe = !process.stdin.isTTY;
  if (!cdk && !codesFile && isPipe) {
    codesFile = '-';
  }

  // Resolve fid: args > env > local .env > global ~/.wos-cli-env > default
  let envFid = process.env.DEFAULT_FID;
  if (!envFid) {
    try {
      const rootEnv = fs.readFileSync(new URL('../.env', import.meta.url), 'utf8');
      const match = rootEnv.match(/DEFAULT_FID\s*=\s*['"]?([a-zA-Z0-9_]+)['"]?/);
      if (match) {
        envFid = match[1];
      }
    } catch (e) {
      // Ignore
    }
  }
  if (!envFid) {
    try {
      const globalEnvPath = path.join(os.homedir(), '.wos-cli-env');
      if (fs.existsSync(globalEnvPath)) {
        const globalEnv = fs.readFileSync(globalEnvPath, 'utf8');
        const match = globalEnv.match(/DEFAULT_FID\s*=\s*['"]?([a-zA-Z0-9_]+)['"]?/);
        if (match) {
          envFid = match[1];
        }
      }
    } catch (e) {
      // Ignore
    }
  }
  fid = fid || envFid || DEFAULT_FID;

  // Determine if running in interactive mode (single code prompt or CLI arg)
  const interactive = !codesFile && process.stdin.isTTY;

  let codesList = [];
  if (codesFile) {
    try {
      console.log(`Loading codes list from ${codesFile === '-' ? 'stdin' : codesFile}...`);
      const parsed = await readCodes(codesFile);
      if (Array.isArray(parsed)) {
        codesList = parsed.map(item => {
          if (typeof item === 'string') return item;
          if (item && typeof item === 'object' && item.code) return item.code;
          return null;
        }).filter(Boolean);
      } else if (parsed && typeof parsed === 'object' && parsed.code) {
        codesList = [parsed.code];
      }
    } catch (e) {
      console.error(e.message);
      process.exit(1);
    }
  } else if (cdk) {
    codesList = [cdk];
  } else {
    // If no code list or CLI arg, prompt for a single code in interactive mode
    if (interactive) {
      const promptedCdk = await prompt('\nEnter gift code: ');
      if (promptedCdk) {
        codesList = [promptedCdk];
      }
    } else {
      console.error('Error: No gift code specified and not running in interactive mode.');
      process.exit(1);
    }
  }

  if (codesList.length === 0) {
    console.log('No codes to process.');
    process.exit(0);
  }

  const state = loadState();

  let codeIndex = 0;
  for (const currentCdk of codesList) {
    console.log('\n========================================');
    console.log(`Processing gift code: ${currentCdk}`);

    if (isRedeemed(state, fid, currentCdk)) {
      console.log(`Code already redeemed or marked as processed for player ${fid} (cached). Skipping.`);
      continue;
    }

    if (codeIndex > 0) {
      console.log('Waiting 10 seconds to avoid API rate limits...');
      await new Promise(resolve => setTimeout(resolve, 10000));
    }
    codeIndex++;

    console.log(`1) Identify player ${fid}...`);
    const player = await getPlayer(fid);
    console.log(JSON.stringify(player, null, 2));
    
    if (!player || player.code !== 0) {
      console.error(`Failed to identify player ${fid}. Skipping code.`);
      continue;
    }

    let attempts = 0;
    const maxAttempts = 4;
    const delayMs = 15000;

    while (attempts < maxAttempts) {
      attempts++;

      console.log(`\n2) Fetching captcha (attempt ${attempts}/${maxAttempts})...`);
      const cap = await getCaptcha(fid);

      if (!cap || !cap.data || !cap.data.img) {
        const msg = (cap && cap.msg) ? String(cap.msg).toLowerCase() : "";
        if (msg.includes("frequent")) {
          if (attempts < maxAttempts) {
            console.log(`CAPTCHA fetch frequent limit hit. Waiting 15 seconds before trying again (attempt ${attempts + 1}/${maxAttempts})...`);
            await new Promise(resolve => setTimeout(resolve, delayMs));
            await getPlayer(fid);
            continue;
          }
        }
        console.log("Failed to fetch captcha from server.", cap);
        break;
      }

      const raw = cap.data.img;
      const base64 = raw.replace(/^data:image\/\w+;base64,/, '');
      const extMatch = raw.match(/^data:image\/(\w+);base64,/);
      const ext = extMatch ? extMatch[1] : 'png';
      const imgData = Buffer.from(base64, 'base64');

      const imgPath = path.join(os.tmpdir(), `wos-captcha.${ext}`);
      fs.writeFileSync(imgPath, imgData);

      // Try to solve automatically
      let solvedCode = "";
      let solverBroken = false;
      try {
        const solverPath = fileURLToPath(new URL('../solver/solver.py', import.meta.url));
        let pythonCmd = process.platform === 'win32' ? 'python' : 'python3';
      
        const venvPath = process.platform === 'win32'
          ? new URL('../solver/venv/Scripts/python.exe', import.meta.url)
          : new URL('../solver/venv/bin/python', import.meta.url);
        const venvPython = fileURLToPath(venvPath);
        if (fs.existsSync(venvPython)) {
          pythonCmd = `"${venvPython}"`;
        }

        const output = execSync(`${pythonCmd} "${solverPath}" "${imgPath}"`, { stdio: ['pipe', 'pipe', 'ignore'] });
        const solved = output.toString().trim();
        if (solved && !solved.includes("ERROR")) {
          if (solved.length === 4) {
            solvedCode = solved;
          } else {
            console.log(`Solver returned invalid guess length (${solved.length}): ${solved}`);
          }
        } else {
          solverBroken = true;
        }
      } catch (e) {
        solverBroken = true;
      }

      let captchaCode = "";
      if (!solvedCode) {
        if (solverBroken) {
          console.log("Automatic solver is not available or failed to solve.");
          if (!interactive) {
            console.log("Running in non-interactive mode. Skipping this code.");
            break;
          }
          openInBrowser(imgPath);
          captchaCode = await prompt('\n4) Enter the captcha code shown: ');
        } else {
          if (attempts < maxAttempts) {
            console.log(`Bypassing invalid solver guess. Retrying with a new captcha (attempt ${attempts + 1}/${maxAttempts})...`);
            await new Promise(resolve => setTimeout(resolve, 2000));
            continue;
          } else {
            console.log("Auto-solve failed 4 times (invalid guesses).");
            if (!interactive) {
              console.log("Running in non-interactive mode. Skipping this code.");
              break;
            }
            openInBrowser(imgPath);
            captchaCode = await prompt('\n4) Enter the captcha code shown: ');
          }
        }
      } else {
        console.log(`Automatically solved CAPTCHA: ${solvedCode}`);
        captchaCode = solvedCode;
      }

      console.log('Redeeming gift code...');
      const res = await redeem(fid, currentCdk, captchaCode);
      console.log(JSON.stringify(res, null, 2));

      const isSuccess = (res && res.code === 0);
      const msg = (res && res.msg) ? res.msg.toLowerCase() : "";
      const isCaptchaError = msg.includes("captcha");

      if (isSuccess) {
        markRedeemed(state, fid, currentCdk);
        saveState(state);
        break;
      }

      if (isCaptchaError) {
        if (attempts < maxAttempts) {
          console.log(`CAPTCHA incorrect. Waiting 15 seconds before trying again (attempt ${attempts + 1}/${maxAttempts})...`);
          await new Promise(resolve => setTimeout(resolve, delayMs));
          await getPlayer(fid);
        } else {
          console.log("\nAuto-solve failed 4 times.");
          if (!interactive) {
            console.log("Running in non-interactive mode. Skipping this code.");
            break;
          }
          openInBrowser(imgPath);
          captchaCode = await prompt('\n4) Enter the captcha code shown: ');
          console.log('\nRedeeming gift code...');
          const finalRes = await redeem(fid, currentCdk, captchaCode);
          console.log(JSON.stringify(finalRes, null, 2));
          
          if (finalRes && (finalRes.code === 0 || !String(finalRes.msg).toLowerCase().includes("captcha"))) {
            markRedeemed(state, fid, currentCdk);
          }
          saveState(state);
          break;
        }
      } else {
        // Other terminal error (CDK code error, claimed, expired) - stop retrying and cache code
        console.log(`Terminal error received: ${res.msg || 'Unknown error'}. Caching code to prevent retries.`);
        markRedeemed(state, fid, currentCdk);
        saveState(state);
        break;
      }
    }
  }
  
  console.log('\nAll done!');
})();
