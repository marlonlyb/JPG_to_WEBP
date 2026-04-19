Tiny 2x3 JPEG fixtures for EXIF orientation regression coverage.

- `orientation-1.jpg`, `orientation-3.jpg`, `orientation-6.jpg`, `orientation-8.jpg` carry EXIF Orientation values matching their filenames.
- `no-exif.jpg` has no APP1 EXIF segment.
- `malformed-exif.jpg` includes a non-decodable EXIF-looking APP1 payload to verify fail-open behavior.

Service and app tests copy these committed fixtures into temp directories so EXIF parsing is exercised against real JPEG files, while `orientation_test.go` uses in-memory helpers for exhaustive 1..8 pixel remapping assertions.
