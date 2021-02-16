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

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func main () {
	r := gin.Default()
	r.GET("/ping", processGET)
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

func processGET(c *gin.Context) {
	fileLocation := "./files/"
	vidFileLocation := fileLocation + "videofile.mp4"
	audFileLocation := fileLocation + "audio.wav"

	fileUrl := c.Request.Header["Url"][0]
	lookingFor := c.Request.Header["Lookingfor"][0]
	log.Print("Filelocation: " + fileLocation)

	// Create google cloud client
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Download video file from given URL
	err = downloadFile(vidFileLocation, fileUrl)

	cmd := getCommandAudioFromVideofile(vidFileLocation)
	cmd.Run()

	result, err := sendRequest(os.Stdout, client, audFileLocation)

	timeStamps := findWordTimestamp(lookingFor, result)

	if err != nil {
		log.Fatal(err)
	}

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

	c.JSON(200, gin.H{
		"message": string(jsonResult),
	})
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

func sendRequest(w io.Writer, client *speech.Client, filename string) (*speechpb.SpeechRecognitionAlternative, error) {
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
	if err != nil {
		return nil, err
	}

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
}