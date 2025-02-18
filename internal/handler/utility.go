package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cradoe/morenee/internal/response"
)

func (util *RouteHandler) HandleUploadFile(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		message := errors.New("invalid request data")
		util.ErrHandler.BadRequest(w, r, message)
		return
	}

	// Get the uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		message := errors.New("error retrieving the file")
		util.ErrHandler.BadRequest(w, r, message)
		return
	}
	defer file.Close()

	fileExtension := filepath.Ext(handler.Filename)

	// Save the file temporarily to the server
	tempFile, err := os.CreateTemp("", fmt.Sprintf("upload-*%s", fileExtension))
	if err != nil {
		util.ErrHandler.ServerError(w, r, err)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Write the uploaded content to the temporary file
	_, err = tempFile.ReadFrom(file)
	if err != nil {
		util.ErrHandler.ServerError(w, r, err)
		return
	}

	// upload to cloud storage
	file_url, err := util.FileUploader.UploadFile(tempFile.Name())

	if err != nil {
		util.ErrHandler.ServerError(w, r, err)
		return
	}

	message := "File uploaded successfully"
	err = response.JSONOkResponse(w, file_url, message, nil)

	if err != nil {
		util.ErrHandler.ServerError(w, r, err)
	}
}
