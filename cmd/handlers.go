package main

import (
	"abakhytzh/doodocs/internal/info"
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		HandleError(w, http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		HandleError(w, http.StatusNotFound)
		return
	}
	tmpl, err := template.ParseFiles("./ui/html/index.html")
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}
}

func UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HandleError(w, http.StatusMethodNotAllowed)
		return
	}
	r.ParseMultipartForm(10 * 1024 * 1024)
	file, handler, err := r.FormFile("myfile")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	fmt.Println("file info")
	fmt.Println("File Name:", handler.Filename)
	fmt.Println("File Sixe:", handler.Size)
	fmt.Println("File Type:", handler.Header.Get("Content-Type"))
	fmt.Println("File Header:", handler.Header)

}
func HandleError(w http.ResponseWriter, num int) {
	errorData := struct {
		ErrorNum     int
		ErrorMessage string
	}{
		ErrorNum:     num,
		ErrorMessage: http.StatusText(num),
	}
	w.WriteHeader(num)
	tmpl, err := template.ParseFiles("./ui/html/error.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, errorData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func ArchiveInformationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HandleError(w, http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		HandleError(w, http.StatusBadRequest)
		return
	}
	defer file.Close()

	if handler.Header.Get("Content-Type") != "application/zip" {
		HandleError(w, http.StatusBadRequest)
		return
	}

	archive, err := zip.NewReader(file, handler.Size)
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}

	// Расчет общего размера и информации о файлах
	totalSize := float64(0)
	for _, file := range archive.File {
		totalSize += float64(file.UncompressedSize64)
	}
	if len(archive.File) == 0 {
		HandleError(w, http.StatusBadRequest)
		return
	}

	response := info.ArchiveInfo{
		Filename:    handler.Filename,
		ArchiveSize: float64(handler.Size),
		TotalSize:   totalSize,
		TotalFiles:  len(archive.File),
		Files:       []info.FileInfo{},
	}

	for _, file := range archive.File {
		fileInfo := info.FileInfo{
			FilePath: file.Name,
			Size:     float64(file.UncompressedSize64),
			MimeType: foundMimeType(file.Name), // Определите тип MIME на основе имени файла
		}

		response.Files = append(response.Files, fileInfo)
		response.TotalSize += fileInfo.Size
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func foundMimeType(filename string) string {
	switch filepath.Ext(filename) {
	case ".jpg":
		return "image/jpeg"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xml":
		return "application/xml"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
func ArchiveFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		HandleError(w, http.StatusMethodNotAllowed)
		return
	}
	// получаем список файлов их запроса
	reader, err := r.MultipartReader()
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}
	// Создайте временный файл для архива
	tempFile, err := os.CreateTemp("", "temp-archive-*.zip")
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()
	//создаем новый зип архив
	archive := zip.NewWriter(w)
	defer archive.Close()
	// Флаг для определения, был ли добавлен хотя бы один допустимый файл в архив
	hasValidFile := false
	// Перебираем все файлы в запросе
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			HandleError(w, http.StatusInternalServerError)
			return
		}
		// Проверяем mime-тип файла
		mimeType := foundMimeType(part.FileName())
		if mimeType == "application/octet-stream" {
			// Файл с недопустимым mime-типом
			HandleError(w, http.StatusBadRequest)
			return
		}
		// Если mime-тип файла допустимый, добавляем его в архив
		fileWriter, err := archive.Create(part.FileName())
		if err != nil {
			HandleError(w, http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(fileWriter, part)
		if err != nil {
			HandleError(w, http.StatusInternalServerError)
			return
		}
		// Установливаем флаг, что был добавлен хотя бы один допустимый файл
		hasValidFile = true
	}

	if !hasValidFile {
		// Не было ни одного допустимого файла
		HandleError(w, http.StatusBadRequest)
		return
	}
	// Откройте временный файл
	file, err := os.Open(tempFile.Name())
	if err != nil {
		HandleError(w, http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Установите правильные заголовки и отправьте файл клиенту
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")
	http.ServeContent(w, r, "archive.zip", time.Time{}, file)
}

// func SendFileByEmailHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		HandleError(w, http.StatusMethodNotAllowed)
// 		return
// 	}

// 	err := r.ParseMultipartForm(10 * 1024 * 1024)
// 	if err != nil {
// 		HandleError(w, http.StatusInternalServerError)
// 		return
// 	}

// 	file, handler, err := r.FormFile("file")
// 	if err != nil {
// 		HandleError(w, http.StatusBadRequest)
// 		return
// 	}
// 	defer file.Close()

// 	// Проверьте тип файла
// 	mimeType := foundMimeType(handler.Filename)
// 	if mimeType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && mimeType != "application/pdf" {
// 		HandleError(w, http.StatusBadRequest)
// 		return
// 	}

// 	emailList := r.FormValue("emails")
// 	emails := strings.Split(emailList, ",")

// 	if len(emails) == 0 {
// 		HandleError(w, http.StatusBadRequest)
// 		return
// 	}

// 	// Настройте SMTP-клиент и отправьте файл на каждый из указанных почтовых адресов
// 	for _, email := range emails {
// 		// Ваш код для отправки файла по электронной почте на адрес "email"
// 	}

// 	// Верните успешный HTTP-ответ
// 	w.WriteHeader(http.StatusOK)
// }

// func HandleArchiveInfo(c *gin.Context) {
// 	file, header, err := c.FormFile("file")
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось получить файл"})
// 		return
// 	}
// 	defer file.Close()

// 	// Добавьте здесь код для проверки, что файл является архивом, и анализа его структуры
// 	// Затем подготовьте информацию о файле и его содержимом

// 	// Пример подготовки информации о файле и его содержимом
// 	filename := header.Filename
// 	archiveSize := header.Size
// 	totalSize := 0
// 	totalFiles := 0

// 	// Создайте структуру для хранения информации о файлах в архиве
// 	var files []map[string]interface{}

// 	// Верните информацию о файле и его содержимом в формате JSON
// 	response := gin.H{
// 		"filename":     filename,
// 		"archive_size": archiveSize,
// 		"total_size":   totalSize,
// 		"total_files":  totalFiles,
// 		"files":        files,
// 	}

// 	c.JSON(http.StatusOK, response)
// }
