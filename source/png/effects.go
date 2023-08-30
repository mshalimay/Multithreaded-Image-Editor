// Package png allows for loading png images and applying
// image flitering effects on them.
package png

import (
	"image/color"
	"math"
	"image"
	"sync"
)

// hashmap of effects and their corresponding kernels in this project
var effects = map[string][]float64{
	"S": {0, -1, 0, -1, 5, -1, 0, -1, 0},
	"E": {-1, -1, -1, -1, 8, -1, -1, -1, -1},
	"B": {1/9.0, 1/9.0, 1/9.0, 1/9.0, 1/9.0, 1/9.0, 1/9.0, 1/9.0, 1/9.0},
}

//=============================================================================
// Kernel struct and methods
//=============================================================================

// Kernel struct represents a kernel to be applied to an image
// @values: array of kernel values
// @size: number of elements in the kernel
// @dim: dimension of the kernel (i.e., dim x dim)
// @center: index of the center element of the kernel
// obs: all kernels in this project are assumed to be square matrices
type Kernel struct{
	values []float64
	size int
	dim int
	center int
}

// Creates a Kernel struct given a string representing an effect string and returns a pointer to it.
func NewKernel(effect string) *Kernel{
	if effect == "G"{
		return nil
	}
	var kernel Kernel
	kernel.values = effects[effect]
	kernel.size = len(kernel.values)
	kernel.dim = int(math.Sqrt(float64(kernel.size)))
	kernel.center = kernel.dim / 2
	return &kernel
}

// Creates a slice of Kernel structs given a slice of strings representing effects and returns a pointer to it.
func CreateKernels(effects []string) []*Kernel{
	kernels := make([]*Kernel, len(effects))
	for i, effect := range effects {
		kernels[i] = NewKernel(effect)
	}
	return kernels
}

//=============================================================================
// Effect application methods
//=============================================================================

// Apply effect represented by 'kernel' to the 'img'. Used by 'parfiles' implementation.
func (img *Image) ApplyEffect(kernel *Kernel) {
	inputPixels, outputPixels := img.GetInputOutputPixels()
	bounds := inputPixels.Bounds()
	if kernel == nil{
		img.Grayscale(inputPixels, outputPixels, bounds.Min.Y, bounds.Max.Y, bounds.Min.X, bounds.Max.X)
	} else{
		img.ConvolveFlat(kernel, inputPixels, outputPixels, bounds.Min.Y, bounds.Max.Y, bounds.Min.X, bounds.Max.X)
	}
}

// Apply effect represented by 'kernel' to a slice of 'img'. Used by 'parslices' implementation.
func (img *Image) ApplyEffectSlice(kernel *Kernel, YStart, YEnd, XStart, XEnd int, wgEffect *sync.WaitGroup) {
	inputPixels, outputPixels := img.GetInputOutputPixels()
	if kernel == nil{
		img.Grayscale(inputPixels, outputPixels, YStart, YEnd, XStart, XEnd)
	} else{
		img.ConvolveFlat(kernel, inputPixels, outputPixels, YStart, YEnd, XStart, XEnd)
	}
	// signal effect application complete
	wgEffect.Done()
}

// Apply effect represented by 'kernel' to a slice of 'img'. Used by 'parslices2' implementation.
func (img *Image) ApplyEffectSlice2(kernel *Kernel, YStart, YEnd, XStart, XEnd int) {
	inputPixels, outputPixels := img.GetInputOutputPixels()
	if kernel == nil{
		img.Grayscale(inputPixels, outputPixels, YStart, YEnd, XStart, XEnd)
	} else{
		img.ConvolveFlat(kernel, inputPixels, outputPixels, YStart, YEnd, XStart, XEnd)
	}
}

// Grayscale applies a grayscale filtering effect to the image
// @inputPixels: pointer to the pixels of image to be filtered
// @outputPixels: pointer to the pixels of image to be written to
// @YStart, YEnd, XStart, XEnd: indexes delimiting the slice of the image pixels to be filtered
func (img *Image) Grayscale(inputPixels *image.RGBA64, 
	outputPixels *image.RGBA64, YStart int, YEnd int, XStart int, XEnd int) {
	for y := YStart; y < YEnd; y++ {
		for x := XStart; x < XEnd; x++ {
			//Returns the pixel (i.e., RGBA) value at a (x,y) position
			r, g, b, a := inputPixels.At(x, y).RGBA()

			// convert to grayscale and clamp to [0, 65535]
			greyC := clamp(float64(r+g+b) / 3)

			// set new pixel color
			outputPixels.Set(x, y, color.RGBA64{greyC, greyC, greyC, uint16(a)})
		}
	}
}

// ConvolveFlat applies a convolution filtering effect to the image using a flat kernel
// @kernel: pointer to the kernel to be applied
// @inputPixels: pointer to the pixels of image to be filtered
// @outputPixels: pointer to the pixels of image to be written to
// @YStart, YEnd, XStart, XEnd: indexes delimiting the slice of the image pixels to be filtered
// references:
// 1) http://www.songho.ca/dsp/convolution/convolution2d_example.html
// 2) https://www.allaboutcircuits.com/technical-articles/two-dimensional-convolution-in-image-processing/
func (img *Image) ConvolveFlat(kernel *Kernel, inputPixels *image.RGBA64, 
	outputPixels *image.RGBA64, YStart int, YEnd int, XStart int, XEnd int){
	
	bounds := inputPixels.Bounds()
	// iterate over image rows
	for y := YStart; y < YEnd; y++ {
		// iterave over image columns
		for x := XStart; x < XEnd; x++ {
			// new pixel colors
			var rNew, gNew, bNew float64

			// iterate over kernel "rows" and "columns"
			for i:=0; i < kernel.size; i++ {
				m := i / kernel.dim // row index in the kernel
				n := i % kernel.dim // column index in the kernel
				
				// invert kernel indexes 
				mm := kernel.dim - 1 - m
				nn := kernel.dim - 1 - n
				
				// adjusted indices to access image pixels
				yy := y + (kernel.center - mm)
				xx := x + (kernel.center - nn)

				// if inbounds, set new values (i.e. zero-padding for out of bounds elements)
				if xx >= bounds.Min.X && xx < bounds.Max.X && yy >= bounds.Min.Y &&  yy < bounds.Max.Y {
					r, g , b , _ := inputPixels.At(xx, yy).RGBA()
					rNew += float64(r) * kernel.values[i]
					gNew += float64(g) * kernel.values[i]
					bNew += float64(b) * kernel.values[i]
				}
			}
			// obs: keeping 'a' channel constant; changing it sometimes gave results different from the 'expected' images
			outputPixels.Set(x, y, color.RGBA64{clamp(rNew), clamp(gNew), clamp(bNew), 65535})
		}
	}
}

//=============================================================================
// Methods for debugging and testing
//=============================================================================

// ConvolveFlat applies a convolution filtering effect to the image using a matrix kernel.
// Not used in the project, kept for reference.
func (img *Image) Convolve(kernel [][]float64){
	
	inputPixels, outputPixels := img.GetInputOutputPixels()

	bounds := inputPixels.Bounds()
	kernelNRows := len(kernel)
	kernelNCols := len(kernel[0])

	kCenterX := kernelNCols / 2
	kCenterY := kernelNRows / 2

	// iterate over image rows
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		// iterave over image columns
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			
			// new pixel colors
			var rNew, gNew, bNew float64

			// iterate over kernel rows
			for m:=0; m < len(kernel); m++ {
				mm := kernelNRows - 1 - m

				// iterate over kernel columns
				for n:=0; n < len(kernel[0]); n++ {
					// adjusted indices to access image pixels 
					nn := kernelNCols - 1 - n
					
					yy := y + (kCenterY - mm);
					xx := x + (kCenterX - nn);

					if xx >= bounds.Min.X && xx < bounds.Max.X && yy >= bounds.Min.Y &&  yy < bounds.Max.Y {
						r, g , b , _ := inputPixels.At(xx, yy).RGBA()

						rNew += float64(r) * kernel[m][n]
						gNew += float64(g) * kernel[m][n]
						bNew += float64(b) * kernel[m][n]
						//aNew += float64(a) 
					}
				}
			}
			outputPixels.Set(x, y, color.RGBA64{clamp(rNew), clamp(gNew), clamp(bNew), 65535})
		}
	}
	// invert the image buffer containing the Final pixels
	img.Final = 1 - img.Final
}
