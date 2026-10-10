# WOS Gift-Code - Backend Protocol (Reverse Engineered)

Base URL: `https://wos-giftcode-api.centurygame.com/api`

This document describes the protocol as the Go client (`go/main.go`) speaks it.
Redeeming a code is a single request: the CAPTCHA step is gone, and the request
must name the player's state (`kid`) in addition to the player ID.

All requests:
- **Method:** `POST`
- **Content-Type:** `application/x-www-form-urlencoded`
- **Required headers (WAF):** `Origin: https://wos-giftcode.centurygame.com`,
  `Referer: https://wos-giftcode.centurygame.com/`, and a regular browser
  `User-Agent`.
  (Without these headers, the server responds with
  `403 Unauthorized request`.)

## Signature (`sign`)

Each payload is signed before sending (`appendSign` in `src/utils/request.js`
of the web client, mirrored by `appendSign` in `go/main.go`):

1. Sort all parameters **alphabetically by key**.
2. Join into a string: `key1=value1&key2=value2&...`
   - object values: `JSON.stringify(value)`
   - all others: `encodeURIComponent(value)`
3. Append fixed **salt**: `tB87#kPtkxqOS2`
4. Compute `sign = md5(string + salt)` as a hex string.
5. Add `sign` as an extra field in the payload.

`/gift_code` only sends string values, so the Go client implements the
`encodeURIComponent` branch only. Its encoding leaves
`A-Z a-z 0-9 - _ . ! ~ * ' ( )` untouched and percent-encodes every other byte
of the UTF-8 value in upper-case hex.

### Worked example

Payload before signing:

| Field | Value |
|-------|-------|
| `fid` | `123456789` |
| `cdk` | `GuDokYTKOR` |
| `kid` | `2402` |
| `time` | `1789316476` |

String to hash (sorted pairs, then the salt):

```text
cdk=GuDokYTKOR&fid=123456789&kid=2402&time=1789316476tB87#kPtkxqOS2
```

Result: `sign = ef2fdbd577d7da48847f69b02e95affb`

Request body (field order does not matter):

```text
sign=ef2fdbd577d7da48847f69b02e95affb&fid=123456789&cdk=GuDokYTKOR&kid=2402&time=1789316476
```

This case is covered by `TestAppendSign` in `go/main_test.go`.

## Redeem a gift code: `/gift_code`

| Field | Meaning |
|-------|---------|
| `fid` | Player ID |
| `cdk` | Gift code |
| `kid` | State number of the player (the `#1234` shown in the in-game profile) |
| `time` | Request time as a Unix timestamp in **seconds** (10 digits) |
| `sign` | Signature over the four fields above |

### Flow

1. `POST /gift_code` with `{fid, cdk, kid, time, sign}` -> redeems the code.

There is no session to set up: the client does not call `/player` or `/captcha`
first.

### Response format

```json
{ "code": 0, "data": { ... }, "msg": "SUCCESS", "err_code": 0 }
```

- `code: 0` means the code was redeemed.
- Any other `code` is an error; `msg` carries the reason (for example an
  invalid, expired or already claimed code) and `err_code` a numeric detail
  code. The individual `err_code` values are not catalogued here.

The Go client treats every answer as final. It caches the code for the player
whether the answer was a success or an error and does not retry it. Only a
request that produces no decodable JSON answer (network failure, non-JSON body)
is left uncached and retried on the next run.

### Rate limiting

The Go client waits 10 seconds between two `/gift_code` requests.

## Other endpoints (previous protocol)

The endpoints below were documented for the previous, CAPTCHA-based protocol.
The Go client no longer calls them and they have not been re-checked against
the current backend. The legacy Node.js client (`js/wos-client.js`) still uses
the first two.

| Purpose | Path | Payload (before `sign`) |
|---------|------|--------------------------|
| Identify player | `/player` | `{ fid, time }` |
| Fetch captcha | `/captcha` | `{ fid, time, init }` |
| Gift-code config | `/gift_code_config` | `{ fid, time }` |

- `time` = `Date.now()` (13-digit milliseconds) in the previous protocol
- `init` = captcha init flag (0/1)

Previous flow:
1. `POST /player` with `{fid, time}` -> returned player data
   (`nickname`, `kid`, `stove_lv`, `avatar_image`, ...) and confirmed the ID.
2. `POST /captcha` with `{fid, time, init}` -> returned a Base64 captcha image in
   `data.img` that had to be solved.
3. `POST /gift_code` with `{fid, cdk, captcha_code, time}` -> redeemed the code.

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
