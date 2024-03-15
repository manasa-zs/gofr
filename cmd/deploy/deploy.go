package deploy

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gofr.dev/pkg/gofr"
)

// Constants
const (
	destinationDir  = "app"
	sourceDir       = "configs"
	zipFileName     = "app.zip"
	deployEndpoint  = "deploy"
	executableName  = "main"
	dockerfileDir   = "app"
	dockerfileName  = "Dockerfile"
	dockerfilePort  = 8000
	dockerfileCMD   = "/main"
	linuxBuildFlags = "CGO_ENABLED=0 GOOS=linux GOARCH=amd64"
)

// Run orchestrates the deployment process.
func Run(ctx *gofr.Context) (interface{}, error) {
	// Clean up previous artifacts
	defer cleanup()

	// Create necessary directories
	if err := os.Mkdir(destinationDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create %s directory: %v", destinationDir, err)
	}

	// Build binary
	if err := buildBinary(ctx); err != nil {
		return nil, fmt.Errorf("failed to build binary: %v", err)
	}

	// Copy configurations
	if err := copyDirectory(sourceDir, filepath.Join(destinationDir, sourceDir)); err != nil {
		return nil, fmt.Errorf("failed to copy configurations: %v", err)
	}

	// Create Dockerfile
	if err := createDockerfile(); err != nil {
		return nil, fmt.Errorf("failed to create Dockerfile: %v", err)
	}

	// Zip the source directory
	if err := zipSource(destinationDir, zipFileName); err != nil {
		return nil, fmt.Errorf("failed to zip directory: %v", err)
	}

	// Deploy the zip file
	if err := deployZip(ctx); err != nil {
		return nil, fmt.Errorf("failed to deploy: %v", err)
	}

	return nil, nil
}

// buildBinary compiles the Go code into an executable binary.
func buildBinary(ctx *gofr.Context) error {
	cmd := exec.Command("sh", "-c", linuxBuildFlags+" go build -o "+destinationDir+"/"+executableName+" .")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing command: %s, %v", output, err)
	}
	fmt.Println("Binary created successfully")
	return nil
}

// copyDirectory copies a directory recursively.
func copyDirectory(src, dst string) error {
	err := os.MkdirAll(dst, os.ModePerm)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		srcFile := filepath.Join(src, file.Name())
		destFile := filepath.Join(dst, file.Name())

		if file.IsDir() {
			if err := copyDirectory(srcFile, destFile); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcFile, destFile); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a file from source to destination.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// createDockerfile creates a Dockerfile in the specified directory.
func createDockerfile() error {
	content := fmt.Sprintf(`FROM alpine:latest
RUN apk add --no-cache tzdata ca-certificates
COPY /%s /%s
RUN chmod +x /%s
EXPOSE %d
CMD ["%s"]`,
		executableName, executableName, executableName, dockerfilePort, dockerfileCMD)

	file, err := os.Create(filepath.Join(dockerfileDir, dockerfileName))
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return err
	}

	fmt.Println("Dockerfile created successfully!")
	return nil
}

// zipSource zips the source directory.
func zipSource(source, target string) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(strings.TrimPrefix(path, source), string(filepath.Separator))
		if info.IsDir() {
			header.Name += "/"
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// deployZip deploys the zip file.
func deployZip(ctx *gofr.Context) error {
	service := ctx.GetHTTPService("deployment")

	file, err := os.Open(zipFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", zipFileName)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	fmt.Println("Deployment Started")

	resp, err := service.PostWithHeaders(ctx, deployEndpoint, nil, body.Bytes(), map[string]string{"Content-Type": writer.FormDataContentType()})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Deployd Successfully : %v", string(responseBody))

	return nil
}

// cleanup removes temporary files and directories.
func cleanup() {
	os.RemoveAll(destinationDir)
	os.RemoveAll(zipFileName)
}
