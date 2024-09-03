package storage

import (
	"errors"
	"fmt"
	"io/ioutil"

	// "log"
	"os"
	"path"

	"github.com/colinmarc/hdfs/v2"
	log "github.com/sirupsen/logrus"
)

// NoSQL: FileStorage struct encapsulating HDFS client
type FileStorage struct {
	client *hdfs.Client
	logger *log.Logger
}

const (
	hdfsRoot     = "/hdfs"
	hdfsCopyDir  = "/hdfs/copied-files/"
	hdfsWriteDir = "/hdfs/created/"
)

func New(logger *log.Logger) (*FileStorage, error) {
	// Local instance
	hdfsUri := os.Getenv("HDFS_URI")

	client, err := hdfs.New(hdfsUri)
	if err != nil {
		logger.Panic(err)
		return nil, err
	}

	// client.SetInt("dfs.replication", 1)

	// Return storage handler with logger and HDFS client
	return &FileStorage{
		client: client,
		logger: logger,
	}, nil
}

func (fs *FileStorage) Close() error {
	if err := fs.client.Close(); err != nil {
		// Log the error or handle it in a way that makes sense for your application
		log.Println("error closing HDFS client: %w", err)
		return errors.New("error closing HDFS client")
	}
	return nil
}

func (fs *FileStorage) CreateDirectories() error {
	// Default permissions
	// 0644 Only the owner can read and write. Everyone else can only read. No one can execute the file.
	err := fs.client.MkdirAll(hdfsCopyDir, 0644)
	if err != nil {
		fs.logger.Println(err)
		return err
	}

	// NoSQL TODO: What is the difference between MkdirAll and Mkdir?
	err = fs.client.Mkdir(hdfsWriteDir, 0644)
	if err != nil {
		fs.logger.Println(err)
		return err
	}

	return nil
}

func (fs *FileStorage) WalkDirectories() []string {
	// Walk all files in HDFS root directory and all subdirectories
	var paths []string
	callbackFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			fs.logger.Printf("Directory: %s\n", path)
			path = fmt.Sprintf("Directory: %s\n", path)
			paths = append(paths, path)
		} else {
			fs.logger.Printf("File: %s\n", path)
			path = fmt.Sprintf("File: %s\n", path)
			paths = append(paths, path)
		}
		return nil
	}
	fs.client.Walk(hdfsRoot, callbackFunc)
	return paths
}

func (fs *FileStorage) CopyLocalFile(localFilePath, fileName string) error {
	// Create local file
	file, err := os.Create(localFilePath)
	if err != nil {
		fs.logger.Println("Error in creating local file:", err)
		return err
	}
	fileContent := "Hello World!"
	_, err = file.WriteString(fileContent)
	if err != nil {
		fs.logger.Println("Error in writing local file:", err)
		return err
	}
	file.Close()

	// Copy file to HDFS
	_ = fs.client.CopyToRemote(localFilePath, hdfsCopyDir+fileName)
	return nil
}

func (fs *FileStorage) WriteFile(fileContent string, fileName string) error {
	filePath := hdfsWriteDir + fileName

	// Create file on HDFS with default replication and block size
	file, err := fs.client.Create(filePath)
	if err != nil {
		fs.logger.Println("Error in creating file on HDFS:", err)
		return err
	}

	defer file.Close() // Ensure file is closed when the function exits

	// Write content
	// Create byte array from string file content
	fileContentByteArray := []byte(fileContent)

	// IMPORTANT: writes content to local buffer, content is pushed to HDFS only when Close is called!
	_, err = file.Write(fileContentByteArray)
	if err != nil {
		fs.logger.Println("Error in writing file on HDFS:", err)
		return err
	}

	// Ensuring all changes are flushed to HDFS
	if err := file.Flush(); err != nil {
		fs.logger.Println("Error flushing file on HDFS:", err)
		return err
	}

	return nil
}

// func (fs *FileStorage) ReadFile(fileName string, isCopied bool) (string, error) {
// 	var filePath string
// 	if isCopied {
// 		filePath = hdfsCopyDir + fileName
// 	} else {
// 		filePath = hdfsWriteDir + fileName
// 	}

// 	// Open file for reading
// 	file, err := fs.client.Open(filePath)
// 	if err != nil {
// 		fs.logger.Println("Error in opening file for reding on HDFS:", err)
// 		return "", err
// 	}

// 	// Read file content
// 	buffer := make([]byte, 1024)
// 	n, err := file.Read(buffer)
// 	if err != nil {
// 		fs.logger.Println("Error in reading file on HDFS:", err)
// 		return "", err
// 	}

// 	// Convert number of read bytes into string
// 	fileContent := string(buffer[:n])
// 	return fileContent, nil
// }

func (fs *FileStorage) ReadFile(fileName string, isCopied bool) (string, error) {
	var dirPath string
	if isCopied {
		dirPath = hdfsCopyDir
	} else {
		dirPath = hdfsWriteDir
	}

	filePath := path.Join(dirPath, fileName)

	// log.Println("File Path:", filePath)

	file, err := fs.client.Open(filePath)
	if err != nil {
		fs.logger.Printf("Error opening file %s for reading on HDFS: %v\n", fileName, err)
		return "", err
	}
	defer file.Close()
	// log.Println("File:", file)
	// fileStat := file.Stat()
	// if err != nil {
	// 	fs.logger.Printf("Error getting file stat for %s: %v\n", fileName, err)
	// 	return "", err
	// }

	// fs.logger.Printf("File size for %s: %d\n", fileName, fileStat.Size())

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		fs.logger.Printf("Error reading file %s on HDFS: %v\n", fileName, err)
		return "", err
	}

	log.Println("FileContent:", "valjda nije prazan")

	return string(fileContent), nil
}

// TODO NoSQL: add method that returns file content as byte array when content is not human readable (images, video,...)
