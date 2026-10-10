const WOS_COMMUNITY_URL = 'https://www.whiteoutsurvival-community.com/en/gift-codes.html';

export async function scrapeWosCommunity(headers) {
  const res = await fetch(WOS_COMMUNITY_URL, { headers });
  if (!res.ok) {
    throw new Error(`HTTP error! status: ${res.status}`);
  }

  const html = await res.text();
  const regex = /data-status="(active|expired)"[\s\S]*?<code class="woscEmbed__code"[^>]*>([^<]+)<\/code>/gi;

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
