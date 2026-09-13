package tasks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"math"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/h2non/bimg"
	"github.com/m-butterfield/mattbutterfield.com/app/data"
	"github.com/m-butterfield/mattbutterfield.com/app/lib"
)

const (
	maxWidth  = 1000
	maxHeight = 1200
)

func saveImage(c *gin.Context) {
	body := &lib.SaveImageRequest{}
	err := c.Bind(body)
	if err != nil {
		lib.InternalError(err, c)
		return
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		lib.InternalError(err, c)
		return
	}
	defer func(client *storage.Client) {
		if err := client.Close(); err != nil {
			log.Println(err)
		}
	}(client)

	upload := client.Bucket(lib.FilesBucket).Object(lib.UploadsPrefix + body.ImageFileName)

	originalData, err := readImage(ctx, upload)
	if err != nil {
		lib.InternalError(err, c)
		return
	}

	size, previewData, err := processImage(originalData)
	if err != nil {
		lib.InternalError(err, c)
		return
	}

	images := client.Bucket(lib.ImagesBucket)
	fileName, err := saveImageFile(ctx, images, originalData)
	if err != nil {
		lib.InternalError(err, c)
		return
	}
	previewID, err := saveImageFile(ctx, images, previewData)
	if err != nil {
		lib.InternalError(err, c)
		return
	}

	var imageTypes []data.ImageType
	if body.ImageType != "" {
		imageTypes = append(imageTypes, data.ImageType{
			Type: body.ImageType,
		})
	}

	var tags []data.Tag
	for _, tagName := range body.Tags {
		if tagName != "" {
			tags = append(tags, data.Tag{Name: tagName})
		}
	}

	if err = ds.SaveImage(&data.Image{
		ID:         fileName,
		PreviewID:  previewID,
		Caption:    body.Caption,
		Location:   body.Location,
		Width:      size.Width,
		Height:     size.Height,
		ImageTypes: imageTypes,
		Tags:       tags,
		CreatedAt:  body.CreatedDate.Time,
		Camera:     body.Camera,
		Lens:       body.Lens,
		Film:       body.Film,
	}); err != nil {
		lib.InternalError(err, c)
		return
	}
	if err := upload.Delete(ctx); err != nil {
		// The image is saved; a cleanup failure must not retry the database insert.
		log.Println(err)
	}
}

func saveImageFile(ctx context.Context, bucket *storage.BucketHandle, imgData []byte) (string, error) {
	hash, err := getHash(imgData)
	if err != nil {
		return "", err
	}
	fileName := hash + ".jpg"
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	w := bucket.Object(fileName).NewWriter(ctx)
	w.ContentType = "image/jpeg"
	if _, err := w.Write(imgData); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return fileName, nil
}

func readImage(ctx context.Context, obj *storage.ObjectHandle) ([]byte, error) {
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func(reader *storage.Reader) {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}(reader)
	return io.ReadAll(reader)
}

func processImage(buffer []byte) (*bimg.ImageSize, []byte, error) {
	// Keep the uploaded bytes intact, including metadata, for original storage.
	img := bimg.NewImage(bytes.Clone(buffer))
	if _, err := img.AutoRotate(); err != nil {
		return nil, nil, err
	}

	size, err := img.Size()
	if err != nil {
		return nil, nil, err
	}
	width := size.Width
	height := size.Height

	if width > maxWidth {
		ratio := float64(height) / float64(width)
		width = maxWidth
		height = int(math.Round(float64(width) * ratio))
	}
	if height > maxHeight {
		ratio := float64(width) / float64(height)
		height = maxHeight
		width = int(math.Round(float64(height) * ratio))
	}

	imgData, err := img.Process(bimg.Options{
		Width:   width,
		Height:  height,
		Quality: 98,
	})
	if err != nil {
		return nil, nil, err
	}
	return &bimg.ImageSize{
		Width:  width,
		Height: height,
	}, imgData, nil
}

func getHash(data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
