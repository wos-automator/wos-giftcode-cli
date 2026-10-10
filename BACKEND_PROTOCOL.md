# WOS Gift-Code - Backend Protocol (Reverse Engineered)

Base URL: `https://wos-giftcode-api.centurygame.com/api`

This document describes the protocol as the Go client (`go/main.go`) speaks it.
Redeeming a code is a single request that names the player ID and the
player's state (`kid`).

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

There is no session to set up and no preceding request.

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
