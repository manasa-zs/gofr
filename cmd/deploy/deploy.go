package deploy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"gofr.dev/pkg/gofr"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
)

func Run(ctx *gofr.Context) (interface{}, error) {
	err := os.Mkdir("app", os.ModePerm)
	if err != nil {
		ctx.Errorf("Failed to create app directory:", err)

		return nil, err
	}

	fmt.Println("Directory app created successfully")

	cmd := exec.Command("sh", "-c", "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app/main .")
	_, err = cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll("app")
		ctx.Errorf("Error executing command:", err)

		return nil, err
	}

	fmt.Println("Binary creation successful")

	err = copyDir("configs", "app/configs")
	if err != nil {
		os.RemoveAll("app")
		ctx.Errorf("Failed to copy configs directory to app:", err)

		return nil, err
	}

	fmt.Println("Copied configs to app directory successful")

	err = CreateDockerfile()
	if err != nil {
		os.RemoveAll("app")

		ctx.Errorf("Failed to create Dockerfile :%v", err)
	}

	err = zipSource("app", "app.zip")
	if err != nil {
		os.RemoveAll("app")
		ctx.Errorf("Failed to zip directory:", err)

		return nil, err
	}

	fmt.Println("Zipped Successfully")

	os.RemoveAll("app")

	service := ctx.GetHTTPService("deployment")

	var writerBody bytes.Buffer
	writer := multipart.NewWriter(&writerBody)

	file, err := os.Open("app.zip")
	if err != nil {
		os.RemoveAll("app.zip")

		return nil, err
	}

	// Add the file as a form data field
	fileWriter, err := writer.CreateFormFile("file", "app.zip")
	if err != nil {
		os.RemoveAll("app.zip")

		return nil, err
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		os.RemoveAll("app.zip")

		return nil, err
	}

	file.Close()

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		os.RemoveAll("app.zip")

		return nil, err
	}

	resp, err := service.PostWithHeaders(ctx, "deploy", nil, writerBody.Bytes(), map[string]string{"Content-Type": writer.FormDataContentType()})
	if err != nil {
		os.RemoveAll("app.zip")
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		os.RemoveAll("app.zip")
		return nil, err
	}

	fmt.Println(string(body))

	os.RemoveAll("app.zip")

	return nil, nil
}

// Function to copy a directory recursively
func copyDir(src, dst string) error {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dst, fileInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(src, entry.Name())
		destinationPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(sourcePath, destinationPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(sourcePath, destinationPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}
	return nil
}

func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// CreateDockerfile creates a Dockerfile with the specified content.
func CreateDockerfile() error {
	// Define the content of the Dockerfile
	dockerfileContent := `FROM alpine:latest

RUN apk add --no-cache tzdata ca-certificates

COPY ./main /main
COPY /configs /configs

RUN chmod +x /main

EXPOSE 8000
CMD ["/main"]
`

	// Create a new file named Dockerfile
	file, err := os.Create("Dockerfile")
	if err != nil {
		return err
	}

	// Write the content to the Dockerfile
	_, err = file.WriteString(dockerfileContent)
	if err != nil {
		return err
	}

	file.Close()

	// Create a new file named Dockerfile
	err = os.Rename("Dockerfile", "app/Dockerfile")
	if err != nil {
		return err
	}

	defer file.Close()

	fmt.Println("Dockerfile created successfully!")
	return nil
}
