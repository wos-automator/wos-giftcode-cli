# WOS Gift-Code - Backend Protocol (Reverse Engineered)

Base URL: `https://wos-giftcode-api.centurygame.com/api`

All requests:
- **Method:** `POST`
- **Content-Type:** `application/x-www-form-urlencoded`
- **Required headers (WAF):** `Origin: https://wos-giftcode.centurygame.com`,
  `Referer: https://wos-giftcode.centurygame.com/`, and a regular browser
  `User-Agent`.
  (Without these headers, the server responds with
  `403 Unauthorized request`.)

## Signature (`sign`)

Each payload is signed before sending (`appendSign` in `src/utils/request.js`):

1. Sort all parameters **alphabetically by key**.
2. Join into a string: `key1=value1&key2=value2&...`
   - object values: `JSON.stringify(value)`
   - all others: `encodeURIComponent(value)`
3. Append fixed **salt**: `tB87#kPtkxqOS2`
4. Compute `sign = md5(string + salt)` as a hex string.
5. Add `sign` as an extra field in the payload.

## Endpoints

| Purpose | Path | Payload (before `sign`) |
|---------|------|--------------------------|
| Identify player | `/player` | `{ fid, time }` |
| Fetch captcha | `/captcha` | `{ fid, time, init }` |
| Redeem gift code | `/gift_code` | `{ fid, cdk, captcha_code, time }` |
| Gift-code config | `/gift_code_config` | `{ fid, time }` |

- `fid` = player ID
- `cdk` = gift code
- `captcha_code` = captcha text solved by user
- `time` = `Date.now()` (13-digit milliseconds)
- `init` = captcha init flag (0/1)

### Flow
1. `POST /player` with `{fid, time}` -> returns player data
   (`nickname`, `kid`, `stove_lv`, `avatar_image`, ...). Confirms the ID exists.
2. `POST /captcha` with `{fid, time, init}` -> returns a Base64 captcha image in
   `data.img` that must be solved.
3. `POST /gift_code` with `{fid, cdk, captcha_code, time}` -> redeems the code.

### Response Format
```json
{ "code": 0, "data": { ... }, "msg": "success", "err_code": "" }
```
Error details are provided in `err_code` (for example: wrong captcha, invalid or
expired code, already redeemed). `code: 1` indicates an error.

### Use of `/gift_code_config`
Gift-code config (`/gift_code_config`) returns the banners of the original wos code redemption page, which can be used to display the same banners in a custom client. The banners are organized by language code, and each language has a `mob_banner` and `pc_banner` URL.

#### shortened sample response:
```json
{
  "code": 0,
  "data": {
    "arab": {
      "mob_banner": "https://gom-s3-user-avatar.s3.us-west-2.amazonaws.com/attachment_prod/images/gift_code_banner/2026-04/6eb58e46efa4b460b3d6ce6189a69b15.png",
      "pc_banner": "https://gom-s3-user-avatar.s3.us-west-2.amazonaws.com/attachment_prod/images/gift_code_banner/2026-04/6eb58e46efa4b460b3d6ce6189a69b15.png"
    },
    "de": { ... },
    "el": { ... },
    "en": { ... },
    "es": { ... },
    "fr": { ... },
    "id": { ... },
    "it": { ... },
    "ja": { ... },
    "ko": { ... },
    "nl": { ... },
    "no": { ... },
    "pl": { ... },
    "pt": { ... },
    "ru": { ... },
    "sv": { ... },
    "th": { ... },
    "tl": { ... },
    "tr": { ... },
    "vi": { ... },
    "zh_cn": { ... },
    "zh_tw": { ... }
  },
  "err_code": 0,
  "msg": "SUCCESS"
}

```