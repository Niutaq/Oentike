#!/usr/bin/env python3
import os
import sys
import subprocess
import tempfile
import cv2
import numpy as np
import argparse

def vectorize_mask(mask, output_svg_path):
    """Convert a binary NumPy mask to SVG with Potrace."""
    # cv2.imwrite requires an 8-bit mask.
    # Potrace traces black pixels by default, so invert the mask first.
    with tempfile.NamedTemporaryFile(suffix=".bmp", delete=False) as temp_bmp:
        temp_bmp_path = temp_bmp.name

    try:
        inverted_mask = cv2.bitwise_not(mask)
        cv2.imwrite(temp_bmp_path, inverted_mask)

        # Potrace options:
        # -s: SVG output
        # -o: output file
        # --group: group related paths in the SVG
        cmd = ["potrace", temp_bmp_path, "-s", "--group", "-o", output_svg_path]
        subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print(f"Saved: {output_svg_path}")
    except subprocess.CalledProcessError as e:
        print(f"Potrace failed while generating {output_svg_path}: {e.stderr.decode('utf-8', errors='ignore')}")
    except FileNotFoundError:
        print("Error: 'potrace' is not installed or is not available on PATH.")
        sys.exit(1)
    finally:
        if os.path.exists(temp_bmp_path):
            os.remove(temp_bmp_path)

def main():
    parser = argparse.ArgumentParser(description="Segment a mushroom raster image and vectorize its layers into structured SVG files.")
    parser.add_argument("input_image", help="Input image path (for example, mushroom.png)")
    parser.add_argument("--output-dir", default=".", help="Output directory for SVG files")
    parser.add_argument("--prefix", help="Output file prefix (for example, 'boletus-edulis'); defaults to the input filename without its extension")
    parser.add_argument("--clusters", type=int, default=4, help="Number of color clusters (default: 4 for background, cap, stem, and hymenophore)")
    args = parser.parse_args()

    input_path = args.input_image
    if not os.path.exists(input_path):
        print(f"Error: input file not found: {input_path}")
        sys.exit(1)

    prefix = args.prefix
    if not prefix:
        prefix = os.path.splitext(os.path.basename(input_path))[0]

    os.makedirs(args.output_dir, exist_ok=True)

    # 1. Load the image.
    img = cv2.imread(input_path)
    if img is None:
        print("Error: OpenCV could not read the image. Check its file format.")
        sys.exit(1)

    print(f"Processing {input_path} ({img.shape[1]}x{img.shape[0]})")

    # Blur before clustering to reduce noise.
    img_blurred = cv2.GaussianBlur(img, (5, 5), 0)

    # 2. Use k-means to extract the main color regions.
    Z = img_blurred.reshape((-1, 3))
    Z = np.float32(Z)

    k = args.clusters
    criteria = (cv2.TERM_CRITERIA_EPS + cv2.TERM_CRITERIA_MAX_ITER, 10, 1.0)
    _, label, center = cv2.kmeans(Z, k, None, criteria, 10, cv2.KMEANS_RANDOM_CENTERS)

    label = label.flatten()

    # Conventional layer names ordered by cluster brightness.
    # A production tool could replace this with interactive mask selection or spatial analysis.
    layer_names = ["bg", "hymenophore", "stem", "cap"]

    # Sort color centers from darkest to lightest.
    brightness = np.sum(center, axis=1)
    sorted_indices = np.argsort(brightness)

    for i, cluster_idx in enumerate(sorted_indices):
        # Treat clusters beyond the known layers as detail layers.
        layer_name = layer_names[i] if i < len(layer_names) else f"layer_{i}"

        # Ignore the background; this prototype assumes it is the darkest cluster.
        if layer_name == "bg":
            continue

        # 3. Create and clean the mask.
        mask = np.zeros(label.shape, dtype=np.uint8)
        mask[label == cluster_idx] = 255
        mask = mask.reshape((img.shape[0], img.shape[1]))

        # Remove small artifacts and smooth mask edges.
        kernel = cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (7,7))
        mask = cv2.morphologyEx(mask, cv2.MORPH_OPEN, kernel)
        mask = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, kernel)

        # 4. Vectorize the mask.
        out_svg = os.path.join(args.output_dir, f"{prefix}.{layer_name}.svg")
        vectorize_mask(mask, out_svg)

    print("Vectorization complete.")

if __name__ == "__main__":
    main()
