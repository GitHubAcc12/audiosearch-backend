package worker

import(
	"net/http"
	"os/exec"
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
	operations		[]string
	responses		[]response.Response
	FileUri			string	// Name of the file without extension
	FileUrl			string	// Where to download the file
	wavFilePath		string
}

func NewWorker(uri string, url string) Worker {
	return Worker{
		id: uuid.NewString(),
		operations: []string{},
		responses: []response.Response{},
		FileUri: uri,
		FileUrl: url,
		wavFilePath: "",
	}
}

func (w Worker) IsFinished() bool {
	return len(w.responses) > 0
}

func (w Worker) downloadFile(filePath string) {
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

func (w Worker) DownloadAndExtractAudioConcurrent(vidFilePath string, audFilePath string) string {
	go w.downloadAndExtractAudio(vidFilePath, audFilePath)
	return w.id
}

func (w Worker) Id() string {
	return w.id
}

func (w Worker) downloadAndExtractAudio(vFile string, aFile string) {
	log.Print("Will download into: " + vFile + " and " + aFile)
	w.downloadFile(vFile)
	w.splitFile(vFile, aFile)
}

func (w Worker) splitFile(inputFilePath string, outputFilePath string) {
	cmd := getCommandAudioFromVideofile(inputFilePath, outputFilePath)
	err := cmd.Run()

	if err != nil {
		log.Print("Error converting to wav file:")
		log.Fatal(err)
	}

	w.wavFilePath = outputFilePath
	filePathSplitFile := filepath.FromSlash("./"+constants.FILES_FOLDER_PATH+w.FileUri+"/")

	w.createDirectory(filePathSplitFile)

	cmd = getCommandSplitAudio(outputFilePath, filePathSplitFile)
	err = cmd.Run()

	if err != nil {
		log.Print("Error splitting wav file:")
		log.Fatal(err)
	}
}

func (w Worker) createDirectory(tmpFilePath string) (string, error) {
	outFilePath := filepath.FromSlash(tmpFilePath)
	return outFilePath, os.MkdirAll(outFilePath, os.ModePerm)
}

func (w Worker) DeleteBigFiles() (error, error) {
	err := os.Remove(w.wavFilePath)
	err2 := os.Remove(strings.TrimSuffix(w.wavFilePath, ".wav")+".mp4")

	return err, err2
}

func (w Worker) getNewSpeechClient(ctx context.Context) *speech.Client {
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func (w Worker) sendSingleApiRequest(ctx context.Context, client *speech.Client, filePath string) (*speechpb.RecognizeResponse, error){
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

func (w Worker) AnalyzeFiles(ctx context.Context, splitFilesFolder string) string {
	go w.analyzeFilesConcurrently(ctx, splitFilesFolder)
	return w.id
}

func (w Worker) analyzeFilesConcurrently(ctx context.Context, splitFilesFolder string) {
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
			log.Print("Received response")
			resp := response.Response{
				TimeStamps: []int64{},
				Message: "",
				Response: result,
				Index: fileIndex,
			}
			operationResults <- resp
			log.Print("Done with routine " + fileUri)
		}(ctx, client, filepath.FromSlash(splitFilesFolder + "/" + f.Name()))
	}

	waitGroup.Wait()

	close(operationResults)
	log.Print("Waitgroup finished and Channel closed!")
	client.Close()

	w.responses = tools.ToSlice(operationResults)
}


func getCommandAudioFromVideofile(inputFile string, outputFile string) *exec.Cmd{
	// ffmpeg -i {video_name} -ab 160k -ac 2 -ar 44100 -vn audio.wav
	return exec.Command("ffmpeg",
	"-i", inputFile,
	"-ab", "160k",
	"-ac", "2",
	"-ar", "44100",
	"-vn", outputFile)
}

func getCommandSplitAudio(inputFile string, outputPath string) *exec.Cmd {
	// ffmpeg -i precalcday1.wav -f segment -segment_time 600 -c copy out%03d.wav
	// Segment audio file into 5 minute long pieces
	return  exec.Command("ffmpeg",
	"-i", inputFile,
	"-f", "segment",
	"-segment_time", "55",
	"-c", "copy",
	outputPath+"%03d.wav")
}