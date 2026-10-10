#!/usr/bin/env python3
import sys
import os

# Suppress ddddocr or ONNX runtime warnings if any
os.environ['TF_CPP_MIN_LOG_LEVEL'] = '3'

try:
    import cv2
    import ddddocr
except ImportError:
    print("ERROR: missing python dependencies (run pip install -r requirements.txt)", file=sys.stderr)
    sys.exit(1)

def solve(img_path):
    if not os.path.exists(img_path):
        print(f"ERROR: file {img_path} not found", file=sys.stderr)
        sys.exit(2)
        
    img = cv2.imread(img_path)
    if img is None:
        print("ERROR: failed to read image", file=sys.stderr)
        sys.exit(3)
        
    # Bilateral filter (Mode 3, which achieved 74% case-insensitive accuracy)
    bilateral = cv2.bilateralFilter(img, 9, 75, 75)
    _, buf = cv2.imencode(".png", bilateral)
    img_bytes = buf.tobytes()
    
    ocr = ddddocr.DdddOcr(show_ad=False)
    pred = ocr.classification(img_bytes)
    return pred.strip().replace(" ", "").replace("\n", "")

def main():
    if len(sys.argv) < 2:
        print("Usage: python solver.py <image_path>", file=sys.stderr)
        sys.exit(4)
        
    code = solve(sys.argv[1])
    print(code)

if __name__ == '__main__':
    main()
