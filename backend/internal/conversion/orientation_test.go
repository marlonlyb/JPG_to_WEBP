package conversion

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestReadOrientationFallsBackToNormal(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int
	}{
		{
			name: "missing exif",
			data: createFixtureJPEGBytes(t, orientationNormal, false),
			want: orientationNormal,
		},
		{
			name: "malformed exif",
			data: createFixtureJPEGBytes(t, orientationNormal, true),
			want: orientationNormal,
		},
		{
			name: "out of range orientation",
			data: createFixtureJPEGBytes(t, 9, false),
			want: orientationNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readOrientation(bytes.NewReader(tt.data))
			if got != tt.want {
				t.Fatalf("readOrientation() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestApplyOrientationRemapsPixelsForAllExifValues(t *testing.T) {
	tests := []struct {
		name        string
		orientation int
		wantRows    [][]color.NRGBA
	}{
		{name: "orientation 1", orientation: 1, wantRows: [][]color.NRGBA{{colorA, colorB}, {colorC, colorD}, {colorE, colorF}}},
		{name: "orientation 2", orientation: 2, wantRows: [][]color.NRGBA{{colorB, colorA}, {colorD, colorC}, {colorF, colorE}}},
		{name: "orientation 3", orientation: 3, wantRows: [][]color.NRGBA{{colorF, colorE}, {colorD, colorC}, {colorB, colorA}}},
		{name: "orientation 4", orientation: 4, wantRows: [][]color.NRGBA{{colorE, colorF}, {colorC, colorD}, {colorA, colorB}}},
		{name: "orientation 5", orientation: 5, wantRows: [][]color.NRGBA{{colorA, colorC, colorE}, {colorB, colorD, colorF}}},
		{name: "orientation 6", orientation: 6, wantRows: [][]color.NRGBA{{colorE, colorC, colorA}, {colorF, colorD, colorB}}},
		{name: "orientation 7", orientation: 7, wantRows: [][]color.NRGBA{{colorF, colorD, colorB}, {colorE, colorC, colorA}}},
		{name: "orientation 8", orientation: 8, wantRows: [][]color.NRGBA{{colorB, colorD, colorF}, {colorA, colorC, colorE}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := createOrientationSourceImage()
			result := applyOrientation(source, tt.orientation)

			if result.Bounds().Dx() != len(tt.wantRows[0]) || result.Bounds().Dy() != len(tt.wantRows) {
				t.Fatalf("applyOrientation() bounds = %v, want %dx%d", result.Bounds(), len(tt.wantRows[0]), len(tt.wantRows))
			}

			for y, row := range tt.wantRows {
				for x, want := range row {
					got := color.NRGBAModel.Convert(result.At(x, y)).(color.NRGBA)
					if got != want {
						t.Fatalf("applyOrientation() pixel (%d,%d) = %#v, want %#v", x, y, got, want)
					}
				}
			}
		})
	}
}

var (
	colorA = color.NRGBA{R: 255, A: 255}
	colorB = color.NRGBA{G: 255, A: 255}
	colorC = color.NRGBA{B: 255, A: 255}
	colorD = color.NRGBA{R: 255, G: 255, A: 255}
	colorE = color.NRGBA{R: 255, B: 255, A: 255}
	colorF = color.NRGBA{G: 255, B: 255, A: 255}
)

func createOrientationSourceImage() *image.NRGBA {
	imageData := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	imageData.SetNRGBA(0, 0, colorA)
	imageData.SetNRGBA(1, 0, colorB)
	imageData.SetNRGBA(0, 1, colorC)
	imageData.SetNRGBA(1, 1, colorD)
	imageData.SetNRGBA(0, 2, colorE)
	imageData.SetNRGBA(1, 2, colorF)
	return imageData
}

func createFixtureJPEGBytes(t *testing.T, orientation int, malformedEXIF bool) []byte {
	t.Helper()

	buffer := bytes.NewBuffer(nil)
	if err := jpeg.Encode(buffer, createOrientationSourceImage(), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("jpeg.Encode() error = %v", err)
	}

	if malformedEXIF {
		return insertAPP1Segment(t, buffer.Bytes(), []byte{'E', 'x', 'i', 'f', 0x00, 0x00, 'b', 'a', 'd'})
	}

	if orientation == orientationNormal {
		return buffer.Bytes()
	}

	return insertAPP1Segment(t, buffer.Bytes(), createOrientationEXIFSegment(orientation))
}

func createOrientationEXIFSegment(orientation int) []byte {
	return []byte{
		'E', 'x', 'i', 'f', 0x00, 0x00,
		'M', 'M', 0x00, 0x2a, 0x00, 0x00, 0x00, 0x08,
		0x00, 0x01,
		0x01, 0x12,
		0x00, 0x03,
		0x00, 0x00, 0x00, 0x01,
		0x00, byte(orientation), 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}

func insertAPP1Segment(t *testing.T, jpegBytes []byte, payload []byte) []byte {
	t.Helper()

	if len(jpegBytes) < 2 || jpegBytes[0] != 0xff || jpegBytes[1] != 0xd8 {
		t.Fatalf("invalid jpeg header")
	}

	segmentLength := len(payload) + 2
	segment := []byte{0xff, 0xe1, byte(segmentLength >> 8), byte(segmentLength)}
	segment = append(segment, payload...)

	result := make([]byte, 0, len(jpegBytes)+len(segment))
	result = append(result, jpegBytes[:2]...)
	result = append(result, segment...)
	result = append(result, jpegBytes[2:]...)
	return result
}
