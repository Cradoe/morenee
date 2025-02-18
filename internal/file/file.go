package file

import (
	"context"
	"fmt"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type FileUploader struct {
	cloud_name string
	api_key    string
	api_secret string
}

func New(cloud_name, api_key, api_secret string) *FileUploader {
	return &FileUploader{
		cloud_name: cloud_name,
		api_key:    api_key,
		api_secret: api_secret,
	}
}

func (f *FileUploader) UploadFile(fileName string) (string, error) {

	cld, err := cloudinary.NewFromParams(f.cloud_name, f.api_key, f.api_secret)
	if err != nil {
		return "", err
	}

	// Upload the file to Cloudinary
	ctx := context.Background()
	uploadResult, err := cld.Upload.Upload(ctx, fileName, uploader.UploadParams{})

	if err != nil {
		return "", err
	}

	// Return the uploaded file URL as a response
	fmt.Printf("File uploaded successfully: %s\n", uploadResult.SecureURL)
	return uploadResult.SecureURL, nil
}
