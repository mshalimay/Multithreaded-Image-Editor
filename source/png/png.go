// Package png allows for loading png images and applying
// image flitering effects on them
package png

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"fmt"
)

//=============================================================================
// Image struct and methods
//=============================================================================

// 'Image' is a structure for working with PNG images.
// 'int' and 'out' are buffers for the image's pixels, that are swapped after each effect is applied
// 'Final' controls which of the buffers contains the last modified image. Used to apply effects sequentially.
type Image struct {
	in     *image.RGBA64   // Buffer 1 for pixels; equals the original image at initialization
	out    *image.RGBA64   // Buffer 2 for pixels
	Bounds image.Rectangle // The size of the image
	Final int			   // 0 if in is the last modified image, 1 if out is the last modified image
}


// GetInputOutputPixels returns image buffers that should act as input and output for next modifications.
// if Final == 1 ==> out = last modified image = input for next modifications.
func (im *Image) GetInputOutputPixels() (*image.RGBA64, *image.RGBA64) {
	// Final == 0 => in = last modified image = input for next modifications
	if im.Final == 0 {
		return im.in, im.out
	} else{
	// Final == 1 => out = last modified image = input for next modifications	
		return im.out, im.in
	}
}

// Set color of pixel in 'x' 'y' position to 'c'
func (im *Image) Set(x, y int, c color.Color) {
	im.out.Set(x, y, c)
}

// Load returns a Image that was loaded based on the filePath parameter
func Load(filePath string) (*Image, error) {

	inReader, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}
	defer inReader.Close()

	inOrig, err := png.Decode(inReader)

	if err != nil {
		return nil, err
	}

	bounds := inOrig.Bounds()

	outImg := image.NewRGBA64(bounds)
	inImg := image.NewRGBA64(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := inOrig.At(x, y).RGBA()
			inImg.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(a)})
		}
	}
	task := &Image{}
	task.in = inImg
	task.out = outImg
	task.Bounds = bounds
	task.Final = 0
	return task, nil
}

// Save saves the image Final state to the given file
func (img *Image) Save(filePath string) error {

	outWriter, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outWriter.Close()

	// save the image with the last modified buffer
	if Final := img.Final; Final == 0 {
		err = png.Encode(outWriter, img.in)
	}else{
		err = png.Encode(outWriter, img.out)
	}

	if err != nil {
		return err
	}
	return nil
}

//clamp will clamp the 'comp' parameter to zero if 'comp'<0 or 65535 if 'comp'>65535
func clamp(comp float64) uint16 {
	return uint16(math.Min(65535, math.Max(0, comp)))
}

//============================================================================
// functions for debugging
//============================================================================

// PrintPixel prints the pixel of 'img' at (x,y) position
func (img *Image) PrintPixel(x int, y int, in_out string){
	if in_out == "in" {
		r, g, b, a := img.in.At(x, y).RGBA()
		fmt.Println("(", r, ",", g, ",", b, ",", a, ")")
	}else {
		r, g, b, a := img.out.At(x, y).RGBA()
		fmt.Println("(", r, ",", g, ",", b, ",", a, ")")
	}
	
}

// PrintPixels prints all pixels of the 'img'
func (img *Image) PrintPixels(){
	bounds := img.out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.out.At(x, y).RGBA()
			print("(", r, ",", g, ",", b, ",", a, ")")
		}
		print("\n")
	}
}


// CompareImages compares two images pixel by pixel and returns true if they are equal, false otherwise
func CompareImages(img1 *Image, img2 *Image) bool {
	equal := true
	for y := 0; y < img1.out.Bounds().Max.Y; y++ {
		for x := 0; x < img1.out.Bounds().Max.X; x++ {
			r1, g1, b1, a1 := img1.out.At(x, y).RGBA()
			var r2, g2, b2, a2 uint32
			
			if img2.Final == 0 {
				r2, g2, b2, a2 = img2.in.At(x, y).RGBA()
			}else {
				r2, g2, b2, a2 = img2.out.At(x, y).RGBA()
			}

			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				// print the pixel values
				fmt.Println("Pixel (", x, ",", y, ") is different")
				fmt.Println("Image 1: (", r1, ",", g1, ",", b1, ",", a1, ")")
				fmt.Println("Image 2: (", r2, ",", g2, ",", b2, ",", a2, ")")
				equal = false
			}
		}
	}
	return equal
}

// WritePixelsToFile writes all pixels of the 'img' to a file
func (img *Image) WritePixelsToFile(filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Unable to create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	bounds := img.out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.out.At(x, y).RGBA()
			fmt.Fprintf(file, "(%v, %v, %v, %v)", r, g, b, a)
		}
		fmt.Fprint(file, "\n")
	}
}


