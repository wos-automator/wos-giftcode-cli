# Legacy API (archived)

This folder holds the client code and documentation for the previous,
CAPTCHA-based gift-code API. The server no longer offers that API, so the
requests made by this code fail. Nothing here is built, tested or released; it
is kept for reference only.

| Path | Content |
|------|---------|
| `BACKEND_PROTOCOL.md` | Description of the previous protocol (`/player`, `/captcha`, `/gift_code` with `captcha_code`, `/gift_code_config`) |
| `js/wos-client.js` | Node.js redemption client for the previous protocol |
| `solver/` | Offline CAPTCHA solver used by that client |
| `setup_solver.sh` | Installer for the solver's Python environment |

The current client and protocol description are in the repository root.
