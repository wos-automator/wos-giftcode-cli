# WOS CAPTCHA Solver

This is a local, lightweight CAPTCHA solver for the Whiteout Survival (WOS) gift code redemption API. It achieves a **74% case-insensitive success rate** on the game's distorted CAPTCHAs.

## How it works
1. **Denoising & Preprocessing**: The solver uses OpenCV's bilateral filter (`cv2.bilateralFilter`) to smooth out background noise lines while preserving the distinct edges of the text characters.
2. **Neural Network Inference**: It runs the pre-trained `ddddocr` classification engine (utilizing ONNX runtime) to read the preprocessed image and return the 4-digit code.
3. **Integration**: The Go and JS clients automatically detect if the solver's virtual environment is initialized. If present, they solve CAPTCHAs in the background, retrying up to 4 times (with a 15-second safety delay to avoid API rate limiting) before falling back to manual entry.

---

## Installation Instructions

To install and initialize the solver, execute the setup script from the root of the repository:

```bash
./setup_solver.sh
```

### What the script does:
1. Verifies that **Python 3** is installed on your system.
2. Creates a clean, isolated Python virtual environment inside the `solver/venv/` directory (avoiding any system-wide package changes).
3. Installs `opencv-python` and `ddddocr` dependencies inside the virtual environment.
4. Performs a verification dry-run to confirm everything works.

---

## Troubleshooting

### Dynamic library loading error on macOS
If Python fails to import OpenCV or ONNX with a dynamic library error (e.g. `Library not loaded`), verify that your Homebrew installation is up-to-date and your system dependencies are properly linked:
```bash
brew update && brew upgrade
```

### Manual execution check
To verify the solver manually, you can run the virtual environment's Python executable against the solver script and a captcha image path:
```bash
./solver/venv/bin/python solver/solver.py /path/to/captcha.png
```
This should print the solved 4-digit CAPTCHA text.
