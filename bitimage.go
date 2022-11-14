// stolen and modified from https://github.com/mugli/png2escpos
package escpos

import (
	"fmt"
	"image"
)

func closestNDivisibleBy8(n int) int {
	q := n / 8
	n1 := q * 8

	return n1
}

func printImage(img image.Image) (xL byte, xH byte, yL byte, yH byte, data []byte) {
	width, height, pixels := getPixels(img)

	removeTransparency(&pixels)
	makeGrayscale(&pixels)

	printWidth := closestNDivisibleBy8(width)
	printHeight := closestNDivisibleBy8(height)
	bytes, _ := rasterize(printWidth, printHeight, &pixels)

	return byte((printWidth >> 3) & 0xff), byte(((printWidth >> 3) >> 8) & 0xff), byte(printHeight & 0xff), byte((printHeight >> 8) & 0xff), bytes
}

func makeGrayscale(pixels *[][]pixel) {
	height := len(*pixels)
	width := len((*pixels)[0])

	for y := 0; y < height; y++ {
		row := (*pixels)[y]
		for x := 0; x < width; x++ {
			pixel := row[x]

			luminance := (float64(pixel.R) * 0.299) + (float64(pixel.G) * 0.587) + (float64(pixel.B) * 0.114)
			var value int
			if luminance < 128 {
				value = 0
			} else {
				value = 255
			}

			pixel.R = value
			pixel.G = value
			pixel.B = value

			row[x] = pixel
		}
	}
}

func removeTransparency(pixels *[][]pixel) {
	height := len(*pixels)
	width := len((*pixels)[0])

	for y := 0; y < height; y++ {
		row := (*pixels)[y]
		for x := 0; x < width; x++ {
			pixel := row[x]

			alpha := pixel.A
			invAlpha := 255 - alpha

			pixel.R = (alpha*pixel.R + invAlpha*255) / 255
			pixel.G = (alpha*pixel.G + invAlpha*255) / 255
			pixel.B = (alpha*pixel.B + invAlpha*255) / 255
			pixel.A = 255

			row[x] = pixel
		}
	}
}

func rasterize(printWidth int, printHeight int, pixels *[][]pixel) ([]byte, error) {
	if printWidth%8 != 0 {
		return nil, fmt.Errorf("printWidth must be a multiple of 8")
	}

	if printHeight%8 != 0 {
		return nil, fmt.Errorf("printHeight must be a multiple of 8")
	}

	bytes := make([]byte, (printWidth*printHeight)>>3)

	for y := 0; y < printHeight; y++ {
		for x := 0; x < printWidth; x = x + 8 {
			i := y*(printWidth>>3) + (x >> 3)
			bytes[i] =
				byte((getPixelValue(x+0, y, pixels) << 7) |
					(getPixelValue(x+1, y, pixels) << 6) |
					(getPixelValue(x+2, y, pixels) << 5) |
					(getPixelValue(x+3, y, pixels) << 4) |
					(getPixelValue(x+4, y, pixels) << 3) |
					(getPixelValue(x+5, y, pixels) << 2) |
					(getPixelValue(x+6, y, pixels) << 1) |
					getPixelValue(x+7, y, pixels))
		}
	}

	return bytes, nil
}

func getPixelValue(x int, y int, pixels *[][]pixel) int {
	row := (*pixels)[y]
	pixel := row[x]

	if pixel.R > 0 {
		return 0
	}

	return 1
}

func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) pixel {
	return pixel{int(r >> 8), int(g >> 8), int(b >> 8), int(a >> 8)}
}

type pixel struct {
	R int
	G int
	B int
	A int
}

func getPixels(img image.Image) (int, int, [][]pixel) {

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]pixel
	for y := 0; y < height; y++ {
		var row []pixel
		for x := 0; x < width; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return width, height, pixels
}
