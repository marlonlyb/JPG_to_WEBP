package conversion

import (
	"image"
	"io"

	"github.com/chai2010/webp"
)

type EncodeOptions struct {
	Quality float32
}

type Encoder interface {
	Encode(writer io.Writer, source image.Image, options EncodeOptions) error
}

type WebPEncoder struct{}

func NewWebPEncoder() WebPEncoder {
	return WebPEncoder{}
}

func (WebPEncoder) Encode(writer io.Writer, source image.Image, options EncodeOptions) error {
	return webp.Encode(writer, source, &webp.Options{
		Lossless: false,
		Quality:  options.Quality,
	})
}
