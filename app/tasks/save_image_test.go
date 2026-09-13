package tasks

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessImagePreservesOriginal(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		wantWidth     int
		wantHeight    int
	}{
		{"landscape", 2000, 1000, 1000, 500},
		{"portrait", 1000, 2000, 600, 1200},
		{"both limits", 2000, 3000, 800, 1200},
		{"small", 100, 200, 100, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var upload bytes.Buffer
			err := jpeg.Encode(&upload, image.NewRGBA(image.Rect(0, 0, tc.width, tc.height)), nil)
			require.NoError(t, err)
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
		})
	}
}

func TestProcessImageInvalidUpload(t *testing.T) {
	_, _, err := processImage([]byte("not an image"))
	require.Error(t, err)
}
