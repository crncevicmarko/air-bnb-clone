package handlers

import (
	"encoding/json"
	"io/ioutil"

	// "log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vukasinc25/fst-airbnb/cache"
	"github.com/vukasinc25/fst-airbnb/storage"
)

type KeyProduct struct{}

type StorageHandler struct {
	logger *log.Logger
	// NoSQL: injecting file storage
	store *storage.FileStorage
	redis *cache.ProductCache
}

func NewStorageHandler(l *log.Logger, s *storage.FileStorage, r *cache.ProductCache) *StorageHandler {
	return &StorageHandler{l, s, r}
}

func (s *StorageHandler) CopyFileToStorage(rw http.ResponseWriter, h *http.Request) {
	fileName := h.FormValue("fileName")

	err := s.store.CopyLocalFile(fileName, fileName)

	if err != nil {
		http.Error(rw, "File storage exception", http.StatusInternalServerError)
		s.logger.Println("File storage exception: ", err)
		return
	}
}

func (s *StorageHandler) WriteFileToStorage(rw http.ResponseWriter, h *http.Request) {
	log.Println("Usli u WriteFilesToStorage")
	// Parse the form data, including the uploaded files
	err := h.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		sendErrorWithMessage(rw, "Unable to parse form", http.StatusBadRequest)
		log.Println("Error parsing form:", err)
		return
	}

	// Get the files from the form data
	files := h.MultipartForm.File["files"]

	log.Println("Files", files)
	for _, fileHeader := range files {
		// Open the file from the form data
		file, err := fileHeader.Open()
		if err != nil {
			sendErrorWithMessage(rw, "Unable to open file", http.StatusInternalServerError)
			log.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		log.Println("File", file)

		// Use the file name from the form or generate one
		fileName := fileHeader.Filename

		log.Println("File name", fileName)
		// Read the file content
		fileContent, err := ioutil.ReadAll(file)
		if err != nil {
			sendErrorWithMessage(rw, "Unable to read file content", http.StatusInternalServerError)
			log.Println("Error reading file content:", err)
			return
		}

		log.Println("FileContent", "valjda nije prazno")

		// Write the file to HDFS
		err = s.store.WriteFile(string(fileContent), fileName)
		if err != nil {
			if strings.Contains(err.Error(), "file already exists") {
				sendErrorWithMessage(rw, "File already exists", http.StatusInternalServerError)
				return
			}
			sendErrorWithMessage(rw, "File storage exception", http.StatusInternalServerError)
			log.Println("File storage exception:", err)
			return
		}
	}

	rw.WriteHeader(http.StatusCreated)
}

func (s *StorageHandler) ReadFileFromStorage(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	fileName := vars["fileName"]
	copied := h.FormValue("isCopied")
	isCopied := false
	if copied != "" {
		isCopied = true
	}

	fileContent, err := s.store.ReadFile(fileName, isCopied)

	if err != nil {
		sendErrorWithMessage(rw, "File storage exception", http.StatusInternalServerError)
		log.Println("File storage exception: ", err)
		return
	}

	err = s.redis.Post([]byte(fileContent), fileName)
	if err != nil {
		sendErrorWithMessage(rw, "Error caching file content", http.StatusInternalServerError)
		log.Println("Error caching file content:", err)
		return
	}

	// Write content to response
	// io.WriteString(rw, fileContent)
	rw.Header().Set("Content-Type", "image/jpeg")
	rw.Write([]byte(fileContent))
	// log.Printf("Content of file %s: %s\n", fileName, fileContent)

}

func sendErrorWithMessage(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := map[string]string{"message": message}
	json.NewEncoder(w).Encode(errorResponse)
}

func (sh *StorageHandler) MiddlewareCacheHit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		log.Println("U func")
		vars := mux.Vars(h)
		id := "images:" + vars["fileName"]
		product, err := sh.redis.GetImage(id)
		if err != nil {
			log.Println("nema slike")
			next.ServeHTTP(rw, h)
		} else {
			log.Println("ima slike")
			rw.Header().Set("Content-Type", "image/jpeg")
			rw.Write([]byte(product))
		}
	})
}
