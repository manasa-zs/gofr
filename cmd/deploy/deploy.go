package deploy

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"

	"gofr.dev/pkg/gofr"
)

func Run(ctx *gofr.Context) (interface{}, error) {
	cmd := exec.Command("sh", "-c", "GOOS=linux go build -o main .")
	_, err := cmd.CombinedOutput()
	if err != nil {
		ctx.Errorf("Error executing command:", err)

		return nil, err
	}

	ctx.Infof("Binary creation successful")

	// Create a new zip file
	zipFile, err := os.Create("app.zip")
	if err != nil {
		ctx.Errorf("Error creating zip file:", err)

		return nil, err
	}
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add the binary file to the zip archive
	if err := addFileToZip(zipWriter, "main"); err != nil {
		ctx.Errorf("Error adding binary to zip:", err)

		return nil, err
	}

	// Add the configs directory to the zip archive if present
	if _, err := os.Stat("configs"); err == nil {
		if err := addDirectoryToZip(zipWriter, "configs"); err != nil {
			ctx.Errorf("Error adding configs directory to zip:", err)

			return nil, err
		}
	}

	ctx.Info("Zip creation Successful")

	body, multipartWriter, err := createMultipartForm("app.zip")
	if err != nil {
		ctx.Errorf("Error creating multipart form object:", err)

		return nil, err
	}

	_, _ = ctx.GetHTTPService("deployment").PostWithHeaders(ctx, "", nil, body.Bytes(), map[string]string{"Content-Type": multipartWriter.FormDataContentType()})

	return nil, nil
}

func createMultipartForm(zipFileName string) (*bytes.Buffer, *multipart.Writer, error) {
	// Create a buffer to store the multipart form data
	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)

	// Open the zip file
	zipFile, err := os.Open(zipFileName)
	if err != nil {
		return nil, nil, err
	}
	defer zipFile.Close()

	// Create a form file part for the zip file
	zipPart, err := multipartWriter.CreateFormFile("zip_file", zipFileName)
	if err != nil {
		return nil, nil, err
	}

	// Copy the zip file content to the form file part
	_, err = io.Copy(zipPart, zipFile)
	if err != nil {
		return nil, nil, err
	}

	// Close the multipart writer to finalize the form data
	err = multipartWriter.Close()
	if err != nil {
		return nil, nil, err
	}

	return &requestBody, multipartWriter, nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the file information
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a new file header
	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}

	// Set the file name as the file header name
	header.Name = filename

	// Create a new zip file entry
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy the file contents to the zip entry
	_, err = io.Copy(writer, file)
	return err
}

func addDirectoryToZip(zipWriter *zip.Writer, directory string) error {
	// Walk through all files and directories within the specified directory
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the header name to the relative path within the zip archive
		header.Name, err = filepath.Rel(filepath.Dir(directory), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// If the file is a directory, simply create the directory header in the zip archive
			header.Name += "/"
			_, err := zipWriter.CreateHeader(header)
			return err
		}

		// If the file is a regular file, add it to the zip archive
		fileToZip, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		// Create a writer for the file in the zip archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Copy the file contents to the zip archive
		if _, err := io.Copy(writer, fileToZip); err != nil {
			return err
		}

		return nil
	})
}
