package main

import (
	"github.com/gin-gonic/gin"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	//"os/exec"
	"strings"
	"encoding/json"
	"net/http"
	"sync"
	"strconv"
	"path/filepath"

	"backend/response"
	"backend/constants"
	"backend/tools"
	"backend/worker"

	// uuid "github.com/google/uuid"
	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var workerMap map[string]worker.Worker

func main () {
	workerMap = make(map[string]worker.Worker)
	// TODO workers will have to be deleted eventually
	r := gin.Default()
	r.POST("/search", searchAudioTimestampsPOST)
	r.GET("/status", statusGET)
	r.GET("/load", loadFileGET)
	r.GET("/check", checkFileGET)
	r.Run()
}

func arrayToString(a []int64, delim string) string {
    return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}



func downloadFile(fileUrl string, filePath string) {
	resp, err := http.Get(fileUrl)
	
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	out, err := os.Create(filePath)

	if err != nil {
		log.Fatal(err)
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)

	if err != nil {
		log.Fatal(err)
	}
}

func getNewSpeechClient() *speech.Client {
	// Create google cloud client
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

/*func loadFileAndExtractAudio(fileUrl string, vidFileName string, audFileName string) {
	log.Print("Starting to download file " + vidFileName)
	downloadFile(fileUrl, vidFileName)
	log.Print("Download finished.")
	cmd := getCommandAudioFromVideofile(filepath.FromSlash(vidFileName), filepath.FromSlash(audFileName))
	err := cmd.Run()
	if err != nil {
		log.Print("Error converting to wav file:")
		log.Fatal(err)
	}
}*/

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
	
	

	
	vidFileLocation := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH + reqWorker.FileUri + ".mp4")
	audFileLocation := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH + reqWorker.FileUri + ".wav")

	/*
	go loadFileAndExtractAudio(fileUrl, vidFileLocation, audFileLocation)

	*/

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
	
	err := os.MkdirAll("./"+constants.FILES_FOLDER_PATH+request.URI, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	/*
	fileUri := "./"+constants.FILES_FOLDER_PATH + request.URI + ".wav"

	outputFilePath := "./"+constants.FILES_FOLDER_PATH+request.URI+"/"

	// Split file into 5 minute sequences
	cmd := getCommandSplitAudio(strings.TrimSuffix(fileUri, ".wav"), outputFilePath)
	cmd.Run()

	
	// Delete big file after small files obtained
	err = os.Remove(fileUri)
	err2 := os.Remove(strings.TrimSuffix(fileUri, ".wav")+".mp4")

	if err != nil || err2 != nil{
		log.Fatal(err)
		log.Fatal(err2)
	}*/


	files, err := ioutil.ReadDir(filepath.FromSlash("./"+constants.FILES_FOLDER_PATH+request.URI))

	if err != nil {
		log.Fatal(err)
	}
	client := getNewSpeechClient()

	var waitGroup sync.WaitGroup

	operationResults := make(chan response.Response, len(files)) 

	log.Print("Starting loop")
	for _, f := range files {
		log.Print("Iteration: " + f.Name())
		// Save index of file to later add index*5min, or index*300 to the word timestamps
		fName := strings.TrimSuffix(f.Name(), ".wav")
		fileIndex, err := strconv.Atoi(fName)
		
		if err != nil {
			log.Fatal(err)
		}
		// Start goroutines to send all the requests to cloud api
		waitGroup.Add(1)
		go func(client *speech.Client, fileUri string) {
			defer waitGroup.Done()
			// Do I need a new client every time to make it fast?
			log.Print("In goroutine! Fileuri: " + fileUri)
			_, err := sendLongRunningRequest(os.Stdout, client, fileUri) // Getting there!!!
			if err != nil {
				log.Fatal(err)
			}
			log.Print("Received response from google!")
			// Save names of the operations to check on them later
			resp := response.Response{
				TimeStamps: []int64{},
				Message: "",
				Response: nil, // Getting there
				Index: fileIndex,
			}
			operationResults <- resp
			log.Print("Done with routine " + fileUri)
		}(client, filepath.FromSlash("./"+constants.FILES_FOLDER_PATH+request.URI + "/" + f.Name()))
	}

	log.Print("Waiting for waitgroup")
	
	waitGroup.Wait() // Wait for all goroutines to finish
	close(operationResults)
	log.Print("Waitgroup finished!")
	client.Close()

	resArray := tools.ToSlice(operationResults)
	log.Print("Created array")

	/*result, err := sendLongRunningRequest(os.Stdout, client, fileUri)

	if err != nil {
		log.Print("Error from sendlongrunningrequest")
		log.Fatal(err)
	}

	resultToSend := response.Response{
		TimeStamps: []int64{},
		OperationNames: resArray,
		Response: "",
	}*/

	jsonResult, err := json.Marshal(resArray)
	log.Print("Marshaled")

	if err != nil {
		log.Fatal(err)
	}

	c.String(200, "{ \"result\": " + string(jsonResult) + " }")
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
}



func pollOperation(client *speech.Client, operationName string) (*speechpb.LongRunningRecognizeResponse, error) {
	ctx := context.Background()
	op := client.LongRunningRecognizeOperation(operationName)
	resp, err := op.Poll(ctx)
	return resp, err
}

func sendLongRunningRequest(w io.Writer, client *speech.Client, filename string) (*speech.LongRunningRecognizeOperation, error) {
	ctx := context.Background()
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	req := &speechpb.LongRunningRecognizeRequest {
		Config: &speechpb.RecognitionConfig {
			Encoding:			speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:	44100,
			LanguageCode:		"en-US",
			AudioChannelCount:	2,
			EnableWordTimeOffsets: true,
		},
		Audio: &speechpb.RecognitionAudio {
			AudioSource: &speechpb.RecognitionAudio_Content{Content: data},
		},
	}

	op, err := client.LongRunningRecognize(ctx, req)

	return op, err

}