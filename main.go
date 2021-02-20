package main

import (
	"github.com/gin-gonic/gin"
	"context"
	"log"
	"os"
	"strings"
	"encoding/json"
	"path/filepath"

	"backend/response"
	"backend/constants"
	"backend/worker"

	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var workerMap map[string]worker.Worker

func main () {
	workerMap = make(map[string]worker.Worker)
	// TODO workers will have to be deleted eventually
	r := gin.Default()
	r.POST("/search", searchAudioTimestampsPOST)
	//r.GET("/status", statusGET)
	r.GET("/load", loadFileGET)
	r.GET("/check", checkFileGET)
	r.Run()
}

func loadFileGET(c *gin.Context) {

	var reqWorker worker.Worker
	if c.Request.Header["Workerid"] == nil {
		fileUrl := c.Request.Header["Url"][0]
		fileUri := c.Request.Header["Uri"][0]
		reqWorker = worker.NewWorker(fileUri, fileUrl)
		workerMap[reqWorker.Id()] = reqWorker
	} else {
		reqWorker = workerMap[c.Request.Header["Workerid"][0]]
	}
	
	log.Print("Worker Uri after creation: " + reqWorker.FileUri)

	
	vidFileLocation := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH + reqWorker.FileUri + ".mp4")
	audFileLocation := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH + reqWorker.FileUri + ".wav")

	reqWorker.DownloadAndExtractAudioConcurrent(vidFileLocation, audFileLocation)


	resultToSend := response.Response{
		TimeStamps: []int64{},
		Response: nil,
		Message: "Download initiated",
		Index: -1,
		WorkerId: reqWorker.Id(),
	}

	jsonResult, err := json.Marshal(resultToSend)

	if err != nil {
		log.Fatal(err)
	}

	c.String(200, string(jsonResult))
}

func checkFileGET(c *gin.Context) {
	fileUri := c.Request.Header["Fileuri"][0]
	fileName := "./"+constants.FILES_FOLDER_PATH + fileUri + ".wav"
	status := "false"
	if _, err := os.Stat(fileName); err == nil {
		status = "true" // File exists
	} else if os.IsNotExist(err) {
		status = "false" // File doesn't exist
	} else {
		log.Fatal(err) // Both possible, something went wrong
	}

	resp := response.Response{
		TimeStamps: []int64{},
		Message: status,
		Response: nil,
		Index: -1,
		WorkerId: "", // Getting there
	}

	respJson, err := json.Marshal(resp)

	if err != nil {
		log.Fatal(err)
	}

	c.String(200, string(respJson))
}

func searchAudioTimestampsPOST(c *gin.Context) {
	var request response.REQUEST
	c.BindJSON(&request)

	var reqWorker worker.Worker

	if len(request.WORKER_ID) == 0 {
		resp := response.Response{
			TimeStamps: []int64{},
			Response: nil,
			Message: "Operation failed: No video/audio associated",
			Index: -1,
			WorkerId: "",
		}

		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Print("Error marshaling response, would have failed anyway")
			log.Print(err)
		}
		c.String(404, string(jsonResp))
		return
	} else {
		reqWorker = workerMap[request.WORKER_ID]
	}

	err1, err2 := reqWorker.DeleteBigFiles()

	if err1 != nil {
		log.Print("Error deleting mp4 file:")
		log.Print(err1)
	}
	if err2 != nil {
		log.Print("Error deleting wav file:")
		log.Print(err2)
	}
	log.Print("Worker fileuri: " + reqWorker.FileUri)
	splitFilePath := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH+reqWorker.FileUri)
	reqWorker.AnalyzeFiles(context.Background(), splitFilePath)


	resultToSend := response.Response{
		TimeStamps: []int64{},
		Message: "Speech evaluation initiated",
		Response: nil,
		Index: -1,
		WorkerId: reqWorker.Id(),
	}

	jsonResult, err := json.Marshal(resultToSend)

	if err != nil {
		log.Fatal(err)
	}

	c.String(200, string(jsonResult))
}



func findWordTimestamp(wordToFind string, audioContent *speechpb.SpeechRecognitionAlternative) []int64 {
	results := make([]int64, 0)
	// Think of smart way to reduce words here
	for _, word := range audioContent.Words {
		if strings.Contains(word.Word, wordToFind) {
			results = append(results, word.StartTime.Seconds)
		}
	}
	return results
}

/*
func statusGET(c *gin.Context) {
	operationName := c.Request.Header["Operationname"][0]
	client := getNewSpeechClient()
	resp, err := pollOperation(client, operationName)

	respJson, err := json.Marshal(resp)

	if err != nil {
		log.Fatal(err)
	}

	c.String(200, string(respJson))

	client.Close()
}*/