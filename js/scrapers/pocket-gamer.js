const POCKET_GAMER_URL = 'https://www.pocketgamer.com/whiteout-survival/codes/';

export async function scrapePocketGamer(headers) {
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
