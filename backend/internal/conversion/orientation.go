package conversion

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"

	"github.com/rwcarlsen/goexif/exif"
)

const (
	orientationNormal     = 1
	orientationFlipH      = 2
	orientationRotate180  = 3
	orientationFlipV      = 4
	orientationTranspose  = 5
	orientationRotate90CW = 6
	orientationTransverse = 7
	orientationRotate90CC = 8
)

type jpegMetadata struct {
	Width       int
	Height      int
	Orientation int
	InputBytes  int64
}

func (m jpegMetadata) normalizedDimensions() (int, int) {
	if orientationSwapsAxes(m.Orientation) {
		return m.Height, m.Width
	}

	return m.Width, m.Height
}

func readJPEGMetadata(inputPath string) (jpegMetadata, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return jpegMetadata{}, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return jpegMetadata{}, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return jpegMetadata{}, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	orientation := readOrientation(bytes.NewReader(data))

	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return jpegMetadata{}, fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	return jpegMetadata{
		Width:       config.Width,
		Height:      config.Height,
		Orientation: orientation,
		InputBytes:  fileInfo.Size(),
	}, nil
}

func decodeNormalizedJPEG(inputPath string) (image.Image, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	orientation := readOrientation(bytes.NewReader(data))

	sourceImage, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	return applyOrientation(sourceImage, orientation), nil
}

func readOrientation(reader io.Reader) int {
	metadata, err := exif.Decode(reader)
	if err != nil {
		return orientationNormal
	}

	tag, err := metadata.Get(exif.Orientation)
	if err != nil || tag == nil {
		return orientationNormal
	}

	orientation, err := tag.Int(0)
	if err != nil {
		return orientationNormal
	}

	if orientation < orientationNormal || orientation > orientationRotate90CC {
		return orientationNormal
	}

	return orientation
}

func applyOrientation(source image.Image, orientation int) image.Image {
	normalizedOrientation := normalizeOrientation(orientation)
	sourceBounds := source.Bounds()
	width := sourceBounds.Dx()
	height := sourceBounds.Dy()

	destinationBounds := image.Rect(0, 0, width, height)
	if orientationSwapsAxes(normalizedOrientation) {
		destinationBounds = image.Rect(0, 0, height, width)
	}

	destination := image.NewNRGBA(destinationBounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			destinationX, destinationY := mapOrientationCoordinates(x, y, width, height, normalizedOrientation)
			pixel := color.NRGBAModel.Convert(source.At(sourceBounds.Min.X+x, sourceBounds.Min.Y+y)).(color.NRGBA)
			destination.SetNRGBA(destinationX, destinationY, pixel)
		}
	}

	return destination
}

func normalizeOrientation(orientation int) int {
	if orientation < orientationNormal || orientation > orientationRotate90CC {
		return orientationNormal
	}

	return orientation
}

func orientationSwapsAxes(orientation int) bool {
	switch normalizeOrientation(orientation) {
	case orientationTranspose, orientationRotate90CW, orientationTransverse, orientationRotate90CC:
		return true
	default:
		return false
	}
}

func mapOrientationCoordinates(x int, y int, width int, height int, orientation int) (int, int) {
	switch normalizeOrientation(orientation) {
	case orientationFlipH:
		return width - 1 - x, y
	case orientationRotate180:
		return width - 1 - x, height - 1 - y
	case orientationFlipV:
		return x, height - 1 - y
	case orientationTranspose:
		return y, x
	case orientationRotate90CW:
		return height - 1 - y, x
	case orientationTransverse:
		return height - 1 - y, width - 1 - x
	case orientationRotate90CC:
		return y, width - 1 - x
	default:
		return x, y
	}
}
