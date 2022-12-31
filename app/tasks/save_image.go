package tasks

import (
	"cloud.google.com/go/storage"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/m-butterfield/mattbutterfield.com/app/data"
	"github.com/m-butterfield/mattbutterfield.com/app/lib"
	"image"
	_ "image/jpeg"
	"io"
	"log"
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

	bucket := client.Bucket(lib.FilesBucket)
	upload := bucket.Object(lib.UploadsPrefix + body.ImageFileName)

	width, height, err := getDimensions(ctx, upload)
	if err != nil {
		lib.InternalError(err, c)
		return
	}

	hash, err := getHash(ctx, upload)
	if err != nil {
		lib.InternalError(err, c)
		return
	}
	fileName := hash + ".jpg"

	if err = copyAndDeleteUpload(ctx, client, upload, fileName); err != nil {
		lib.InternalError(err, c)
		return
	}
	var imageTypes []data.ImageType
	if body.ImageType != "" {
		imageTypes = append(imageTypes, data.ImageType{
			Type: body.ImageType,
		})
	}

	if err = ds.SaveImage(&data.Image{
		ID:         fileName,
		Caption:    body.Caption,
		Location:   body.Location,
		Width:      width,
		Height:     height,
		ImageTypes: imageTypes,
		CreatedAt:  body.CreatedDate.Time,
	}); err != nil {
		lib.InternalError(err, c)
		return
	}
}

func getDimensions(ctx context.Context, obj *storage.ObjectHandle) (int, int, error) {
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func(reader *storage.Reader) {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}(reader)
	imgConf, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, err
	}
	return imgConf.Width, imgConf.Height, nil
}

func getHash(ctx context.Context, obj *storage.ObjectHandle) (string, error) {
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer func(reader *storage.Reader) {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}(reader)
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func copyAndDeleteUpload(ctx context.Context, client *storage.Client, upload *storage.ObjectHandle, fileName string) error {
	result := client.Bucket(lib.ImagesBucket).Object(fileName)
	if _, err := result.CopierFrom(upload).Run(ctx); err != nil {
		return err
	}
	if err := upload.Delete(ctx); err != nil {
		return err
	}
	return nil
}
