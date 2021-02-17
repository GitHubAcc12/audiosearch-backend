package main

import (
	"github.com/gin-gonic/gin"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"encoding/json"
	"backend/response"
	"net/http"
	//"strconv"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func main () {
	r := gin.Default()
	r.POST("/search", searchAudioTimestamps)
	r.GET("/status", statusGET)
	r.Run()
}

func arrayToString(a []int64, delim string) string {
    return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

func getCommandAudioFromVideofile(inputFile string) *exec.Cmd{
	//'ffmpeg -i {video_name} -ab 160k -ac 2 -ar 44100 -vn audio.wav'
	return exec.Command("ffmpeg",
	"-i", inputFile,
	"-ab", "160k",
	"-ac", "2",
	"-ar", "44100",
	"-vn", "./files/audio.wav")
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(filepath)

	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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

func searchAudioTimestamps(c *gin.Context) {
	fileLocation := "./files/"
	vidFileLocation := fileLocation + "videofile.mp4"
	audFileLocation := fileLocation + "audio.wav"


	var request response.REQUEST
	c.BindJSON(&request)
	

	fileUrl := request.URL

	client := getNewSpeechClient()


	// Download video file from given URL
	err := downloadFile(vidFileLocation, fileUrl)

	cmd := getCommandAudioFromVideofile(vidFileLocation)
	cmd.Run()

	result, err := sendLongRunningRequest(os.Stdout, client, audFileLocation)

	if err != nil {
		log.Print("Error from sendlongrunningrequest")
		log.Fatal(err)
	}


	resultToSend := response.Response{
		TimeStamps: []int64{},
		OperationName: result.Name(),
		Response: nil,
	}

	jsonResult, err := json.Marshal(resultToSend)
		
	if err != nil {
		log.Fatal(err)
	}

	//stringResult/*, err2*/ := string(resultToSend)
	//log.Print(stringResult)
	/*if err2 != nil {
		log.Print("err2")
		log.Print(string(jsonResult))
		log.Fatal(err2)
	}*/

	c.String(200, string(jsonResult))

	client.Close()

	/*
	timeStamps := findWordTimestamp(lookingFor, result)



	// jsonTimeStamps, err := json.Marshal(timeStamps)

	if err != nil {
		log.Fatal(err)
	}

	resultToSend := response.Response {
		TimeStamps: timeStamps,
	}

	jsonResult, err := json.Marshal(resultToSend)
		
	if err != nil {
		log.Fatal(err)
	}
	*/

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

	/*
	// Here is where progress bar might be good?
	resp, err := op.Wait(ctx)
	if err != nil {
		return nil, err
	}

	for _, result := range resp.Results {
		log.Print("in result loop")
		for _, alt := range result.Alternatives {
			fmt.Fprintf(w, "\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}
	return resp.Results[0].Alternatives[0], nil
	*/
}