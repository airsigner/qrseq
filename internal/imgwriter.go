package internal

import (
	"image"
	"image/color"

	"github.com/yeqown/go-qrcode/v2"
)

type Option struct {
	Padding   int
	BlockSize int
}

type imgWriter struct {
	img      image.Image
	option   *Option
	callback func(image.Image)
}

var (
	backgroundColor = color.White
	foregroundColor = color.Black
)

// NewImageWriter creates a new instance of the imgWriter struct and returns
// it as a qrcode.Writer.
//
// The function takes two parameters:
//   - callback: a function that takes an image.Image as input and does not return
//     anything.
//   - opt: a pointer to an Option struct.
//
// The function returns a qrcode.Writer.
func NewImageWriter(callback func(image.Image), opt *Option) qrcode.Writer {
	return &imgWriter{
		img:      nil,
		option:   opt,
		callback: callback,
	}
}

// Write writes the QR code matrix to an image and sets it in the imgWriter
// struct.
//
// It takes a qrcode.Matrix as input and returns an error.
// The function calculates the width and height of the image based on the matrix
// size and padding.
// It creates a new image.Paletted with the calculated dimensions and a palette
// containing the background and foreground colors.
// It calculates the indices of the background and foreground colors in the
// image's palette.
// It defines a helper function rectangle that sets the color of a rectangular
// area in the image.
// It sets the background color of the entire image.
// It iterates over the matrix and sets the foreground color of the non-zero
// values.
// It sets the image in the imgWriter struct and returns nil.
func (w *imgWriter) Write(mat qrcode.Matrix) error {
	padding := w.option.Padding
	blockWidth := w.option.BlockSize
	width := mat.Width()*blockWidth + 2*padding
	height := width

	img := image.NewPaletted(
		image.Rect(0, 0, width, height),
		[]color.Color{backgroundColor, foregroundColor},
	)
	bgColor := uint8(img.Palette.Index(backgroundColor))
	fgColor := uint8(img.Palette.Index(foregroundColor))

	rectangle := func(x1, y1 int, x2, y2 int, img *image.Paletted, color uint8) {
		for x := x1; x < x2; x++ {
			for y := y1; y < y2; y++ {
				pos := img.PixOffset(x, y)
				img.Pix[pos] = color
			}
		}
	}

	// background
	rectangle(0, 0, width, height, img, bgColor)

	mat.Iterate(qrcode.IterDirection_COLUMN, func(x int, y int, v qrcode.QRValue) {
		if v.IsSet() {
			sx := x*blockWidth + padding
			sy := y*blockWidth + padding
			ex := (x+1)*blockWidth + padding
			ey := (y+1)*blockWidth + padding
			rectangle(sx, sy, ex, ey, img, fgColor)
		}
	})

	w.img = img
	return nil
}

// Close closes the imgWriter and invokes the callback function with the image.
//
// It does not return any value.
func (w imgWriter) Close() error {
	w.callback(w.img)
	return nil
}

// Image returns the image stored in the imgWriter struct.
//
// It takes no parameters.
// It returns an image.Image.
func (w imgWriter) Image() image.Image {
	return w.img
}
