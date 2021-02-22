package worker

import(
	"net/http"
	"os"
	"io"
	"path/filepath"
	"context"
	"io/ioutil"
	"strings"
	"strconv"
	"sync"
	"log"

	"backend/response"
	"backend/tools"
	"backend/constants"

	uuid "github.com/google/uuid"
	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)


type Worker struct {
	id				string
	Responses		[]response.Response
	FileUri			string	// Name of the file without extension
	FileUrl			string	// Where to download the file
	wavFilePath		string
}

func NewWorker(uri string, url string) *Worker {
	return &Worker{
		id: uuid.NewString(),
		Responses: make([]response.Response, 0),
		FileUri: uri,
		FileUrl: url,
		wavFilePath: "",
	}
}

func (w *Worker) IsFinished() bool {
	log.Print(len(w.Responses))
	return len(w.Responses) > 0
}

func (w *Worker) downloadFile(filePath string) {
	resp, err := http.Get(w.FileUrl)
	
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

func (w *Worker) DownloadAndExtractAudioConcurrent(vidFilePath string, audFilePath string) string {
	go w.downloadAndExtractAudio(vidFilePath, audFilePath)
	return w.id
}

func (w *Worker) Id() string {
	return w.id
}

func (w *Worker) downloadAndExtractAudio(vFile string, aFile string) {
	log.Print("Will download into: " + vFile + " and " + aFile)
	w.downloadFile(vFile)
	w.splitFile(vFile, aFile)
}

func (w *Worker) splitFile(inputFilePath string, outputFilePath string) {
	cmd := tools.GetCommandAudioFromVideofile(inputFilePath, outputFilePath)
	err := cmd.Run()

	if err != nil {
		log.Print("Error converting to wav file:")
		log.Fatal(err)
	}

	w.wavFilePath = outputFilePath
	filePathSplitFile := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH+w.FileUri+"/")

	w.createDirectory(filePathSplitFile)

	cmd = tools.GetCommandSplitAudio(outputFilePath, filePathSplitFile)
	err = cmd.Run()

	if err != nil {
		log.Print("Error splitting wav file:")
		log.Fatal(err)
	}
}

func (w *Worker) createDirectory(tmpFilePath string) (string, error) {
	outFilePath := filepath.FromSlash(tmpFilePath)
	return outFilePath, os.MkdirAll(outFilePath, os.ModePerm)
}

func (w *Worker) DeleteBigFiles() (error, error) {
	err := os.Remove(w.wavFilePath)
	err2 := os.Remove(strings.TrimSuffix(w.wavFilePath, ".wav")+".mp4")

	return err, err2
}

func (w *Worker) getNewSpeechClient(ctx context.Context) *speech.Client {
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func (w *Worker) sendSingleApiRequest(ctx context.Context, client *speech.Client, filePath string) (*speechpb.RecognizeResponse, error){
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	req := &speechpb.RecognizeRequest {
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

	return client.Recognize(ctx, req)
	// returns response, error
}

func (w *Worker) AnalyzeFiles(ctx context.Context, splitFilesFolder string) string {
	go w.analyzeFilesConcurrently(ctx, splitFilesFolder)
	return w.id
}

func (w *Worker) analyzeFilesConcurrently(ctx context.Context, splitFilesFolder string) {
	log.Print("Splitfiledfolder: " + splitFilesFolder)
	files, err := ioutil.ReadDir(splitFilesFolder)

	if err != nil {
		log.Fatal(err)
	}

	client := w.getNewSpeechClient(ctx)

	var waitGroup sync.WaitGroup
	operationResults := make(chan response.Response, len(files))
	log.Print("In Worker: starting loop")
	for _, f := range files {
		log.Print("Iteration " + f.Name())
		fName := strings.TrimSuffix(f.Name(), ".wav")
		fileIndex, err := strconv.Atoi(fName)

		if err != nil {
			log.Fatal(err)
		}

		// Start goroutines
		waitGroup.Add(1)
		go func(ctx context.Context, client *speech.Client, fileUri string) {
			defer waitGroup.Done()

			log.Print("In worker goroutine! Fileuri: " + fileUri)
			result, err := w.sendSingleApiRequest(ctx, client, fileUri)
			if err != nil {
				log.Print("Error in goroutine:")
				log.Fatal(err)
			}
			if len(result.Results) > 0 {
				log.Print(result.Results[0].Alternatives[0].Transcript)
			}
			
			//log.Print("Received response")
			resp := response.Response{
				TimeStamps: []int64{},
				Message: "",
				GoogleResponse: result,
				Index: int64(fileIndex),
			}
			operationResults <- resp
			log.Print("Done with routine " + fileUri)
		}(ctx, client, filepath.FromSlash(splitFilesFolder + "/" + f.Name()))
	}

	waitGroup.Wait()

	close(operationResults)
	// TODO are the right things in here?
	log.Print("Waitgroup finished and Channel closed!")
	client.Close()

	w.Responses = tools.ToSlice(operationResults)


	// TODO this is where it fails. Why???
	log.Print("REsponse iteration double triple safe:")
	for _, r := range w.Responses {
		if len(r.GoogleResponse.Results) > 0 {
			log.Print(r.GoogleResponse.Results[0].Alternatives[0].Transcript)
		}
		
	}
}

func (w *Worker) findWordTimestampsInResponses(word string) {
	var waitGroup sync.WaitGroup
	operationResults := make(chan response.Response, len(w.Responses))
	for _, resp := range w.Responses {
		log.Print("Outside of loop Response: " + strconv.FormatInt(resp.Index, 10))
		waitGroup.Add(1)
		go func(iResp response.Response) {
			defer waitGroup.Done()
			log.Print("Response: " + strconv.FormatInt(iResp.Index, 10))
			iResp.FindWordTimestamps(word)
			operationResults <- iResp
		}(resp)
	}
	waitGroup.Wait()
	close(operationResults)
	log.Print("Waitgroup find closed!")
	w.Responses = tools.ToSlice(operationResults)
	log.Print("Waitgroup find closed and array assigned!")
}

func (w *Worker) FindWordTimestamps(word string) []int64 {
	w.findWordTimestampsInResponses(word)
	tStamps := make([]int64, 0)
	for _, resp := range w.Responses {
		tStamps = append(tStamps, resp.TimeStamps...)
	}
	return tStamps
}

