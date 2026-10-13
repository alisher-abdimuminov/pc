"""Builds a crude 'photo of a phone screen showing a face' from a face photo (replay attack simulation)."""
import sys
import cv2
import numpy as np

src = cv2.imread(sys.argv[1])
h, w = src.shape[:2]
# crop around upper-middle (face region is large in these photos); keep the full image so the face is whole
scr_h = 900
scr_w = int(scr_h * w / h)
screen = cv2.resize(src, (scr_w, scr_h), interpolation=cv2.INTER_AREA).astype(np.float32)
# moire / pixel grid + backlight tint + glare
yy, xx = np.mgrid[0:scr_h, 0:scr_w]
grid = 0.07 * (np.sin(xx * 1.9) + np.sin(yy * 2.1))[..., None]
screen *= (1 + grid)
screen *= np.array([1.05, 1.0, 0.92], dtype=np.float32)  # cooler/bluer display cast (BGR)
glare = np.clip(1 - np.hypot(xx - scr_w * 0.75, yy - scr_h * 0.2) / (scr_w * 0.6), 0, 1)[..., None]
screen = screen * 0.85 + 70 * glare
screen = np.clip(screen, 0, 255).astype(np.uint8)

W, H = 1600, 1200
canvas = np.zeros((H, W, 3), np.float32)
canvas[:] = np.linspace(60, 110, W)[None, :, None] * np.array([0.9, 1.0, 1.1])  # table gradient
pad = 38  # phone bezel
phone = np.full((scr_h + 2 * pad, scr_w + 2 * pad, 3), 18, np.uint8)
phone[pad:pad + scr_h, pad:pad + scr_w] = screen
ph, pw = phone.shape[:2]
src_pts = np.float32([[0, 0], [pw, 0], [pw, ph], [0, ph]])
dx, dy = (W - pw) // 2, (H - ph) // 2
dst_pts = np.float32([[dx + 30, dy + 10], [dx + pw - 20, dy - 5], [dx + pw - 40, dy + ph + 15], [dx + 10, dy + ph - 20]])
M = cv2.getPerspectiveTransform(src_pts, dst_pts)
warped = cv2.warpPerspective(phone, M, (W, H), borderValue=(0, 0, 0))
mask = cv2.warpPerspective(np.full(phone.shape[:2], 255, np.uint8), M, (W, H))
canvas = np.where(mask[..., None] > 0, warped.astype(np.float32), canvas)
out = cv2.GaussianBlur(canvas, (0, 0), 1.1)
out += np.random.default_rng(1).normal(0, 4, out.shape)
cv2.imwrite(sys.argv[2], np.clip(out, 0, 255).astype(np.uint8), [cv2.IMWRITE_JPEG_QUALITY, 82])
