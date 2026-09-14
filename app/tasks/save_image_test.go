package tasks

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	vips.LoggingSettings(nil, vips.LogLevelWarning)
	if err := vips.Startup(nil); err != nil {
		log.Fatal(err)
	}
	code := m.Run()
	vips.Shutdown()
	os.Exit(code)
}

func TestProcessImagePreservesOriginal(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		wantWidth     int
		wantHeight    int
		orientation   byte
	}{
		{"landscape", 2000, 1000, 1000, 500, 0},
		{"portrait", 1000, 2000, 600, 1200, 0},
		{"both limits", 2000, 3000, 800, 1200, 0},
		{"small", 100, 200, 100, 200, 0},
		{"resize after EXIF rotation", 2000, 1000, 600, 1200, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))
			var upload bytes.Buffer
			err := jpeg.Encode(&upload, img, nil)
			require.NoError(t, err)
			if tc.orientation != 0 {
				// Insert an EXIF APP1 segment with one TIFF orientation tag after JPEG SOI.
				exif := []byte{
					0xff, 0xe1, 0x00, 0x22,
					'E', 'x', 'i', 'f', 0, 0,
					'I', 'I', 42, 0, 8, 0, 0, 0,
					1, 0,
					0x12, 0x01, 3, 0, 1, 0, 0, 0, tc.orientation, 0, 0, 0,
					0, 0, 0, 0,
				}
				encoded := bytes.Clone(upload.Bytes())
				upload.Reset()
				upload.Write(encoded[:2])
				upload.Write(exif)
				upload.Write(encoded[2:])
			}
			original := bytes.Clone(upload.Bytes())

			size, preview, err := processImage(upload.Bytes())
			require.NoError(t, err)
			assert.Equal(t, original, upload.Bytes())
			assert.Equal(t, tc.wantWidth, size.Width)
			assert.Equal(t, tc.wantHeight, size.Height)
			decoded, err := jpeg.DecodeConfig(bytes.NewReader(preview))
			require.NoError(t, err)
			assert.Equal(t, size.Width, decoded.Width)
			assert.Equal(t, size.Height, decoded.Height)
			output, err := vips.NewImageFromBuffer(preview)
			require.NoError(t, err)
			defer output.Close()
			assert.LessOrEqual(t, output.Orientation(), 1)
			assert.True(t, output.HasICCProfile())
		})
	}
}

func TestProcessImageGrayscale(t *testing.T) {
	var upload bytes.Buffer
	img := image.NewGray(image.Rect(0, 0, 100, 200))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.Gray{Y: 128}), image.Point{}, draw.Src)
	require.NoError(t, jpeg.Encode(&upload, img, nil))

	_, preview, err := processImage(upload.Bytes())
	require.NoError(t, err)
	decoded, err := jpeg.Decode(bytes.NewReader(preview))
	require.NoError(t, err)
	got := color.RGBAModel.Convert(decoded.At(50, 100)).(color.RGBA)
	assert.InDelta(t, 128, got.R, 5)
	assert.InDelta(t, got.R, got.G, 1)
	assert.InDelta(t, got.R, got.B, 1)
}

func TestProcessImageInvalidUpload(t *testing.T) {
	_, _, err := processImage([]byte("not an image"))
	require.Error(t, err)
}
