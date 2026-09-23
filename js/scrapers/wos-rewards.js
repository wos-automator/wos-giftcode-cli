const WOS_REWARDS_URL = 'https://wosrewards.com';

export async function scrapeWosRewards(headers) {
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
