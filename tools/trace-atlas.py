#!/usr/bin/env python3
"""Trace generated RGBA botanical art into the fixed Oentike palette.

Requires Python with OpenCV/numpy and the potrace executable. Layers represent
colours, not anatomical parts. Alpha is preserved as the specimen silhouette.
Usage: python3 tools/trace-atlas.py input.png output.svg
"""
import argparse
from pathlib import Path
import subprocess
import tempfile
import xml.etree.ElementTree as ET
import cv2
import numpy as np


def trace(source, destination):
    image = cv2.imread(str(source), cv2.IMREAD_UNCHANGED)
    if image is None or image.ndim != 3 or image.shape[2] != 4:
        raise ValueError('Expected an RGBA image with a transparent background')
    # Fit the opaque specimen into the same generous square margins as old plates.
    y, x = np.where(image[:, :, 3] > 127)
    if not len(x):
        raise ValueError('Empty image')
    image = image[y.min():y.max()+1, x.min():x.max()+1]
    scale = 760 / max(image.shape[:2])
    image = cv2.resize(image, None, fx=scale, fy=scale, interpolation=cv2.INTER_AREA)
    height, width = image.shape[:2]
    canvas = np.zeros((1024, 1024, 4), dtype=np.uint8)
    canvas[(1024-height)//2:(1024-height)//2+height, (1024-width)//2:(1024-width)//2+width] = image
    opaque = canvas[:, :, 3] > 127
    rgb = cv2.cvtColor(canvas[:, :, :3], cv2.COLOR_BGR2RGB).astype(np.float32)
    palette = np.array([[246, 245, 239], [228, 119, 45], [25, 37, 30]], dtype=np.float32)
    labels = np.argmin(np.sum((rgb[:, :, None, :] - palette)**2, axis=3), axis=2).astype(np.uint8)
    labels = cv2.medianBlur(labels, 3)
    ET.register_namespace('', 'http://www.w3.org/2000/svg')
    svg = ET.Element('{http://www.w3.org/2000/svg}svg', {'viewBox':'0 0 1024 1024', 'width':'1024', 'height':'1024'})
    with tempfile.TemporaryDirectory() as folder:
        for index, (name, color) in enumerate([('form-light','#F6F5EF'), ('accent','#E4772D'), ('ink-shade','#19251E')]):
            mask = np.where(opaque & (labels == index), 0, 255).astype(np.uint8)
            bmp = Path(folder)/f'{name}.bmp'
            out = Path(folder)/f'{name}.svg'
            cv2.imwrite(str(bmp), mask)
            subprocess.run(['potrace', str(bmp), '-s', '--turdsize','3','--opttolerance','0.35','-o',str(out)], check=True)
            doc = ET.parse(out).getroot()
            for group in doc.findall('{http://www.w3.org/2000/svg}g'):
                group.set('id', name)
                group.set('fill', color)
                svg.append(group)
    destination.parent.mkdir(parents=True, exist_ok=True)
    ET.indent(svg)
    ET.ElementTree(svg).write(destination, encoding='unicode')
    print(f'{destination}: {destination.stat().st_size // 1024} KiB')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    trace(args.source, args.destination)
