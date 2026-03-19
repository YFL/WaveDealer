package wd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

type WaveDealer struct {
	requests  chan string
	playQueue chan string
	isPlaying atomic.Bool
}

func NewWaveDealer() (*WaveDealer, error) {
	fmt.Println("Audio device ready")

	return &WaveDealer{
		requests:  make(chan string),
		playQueue: make(chan string),
	}, nil
}

func (wd *WaveDealer) RequestYoutubeSong(url string) {
	wd.requests <- url
}

func (wd *WaveDealer) RunRequestWorker() {
	for req := range wd.requests {
		fmt.Printf("Handling request %s\n", req)
		go func() {
			fmt.Printf("Getting filename for %s\n", req)
			cmd := exec.Command("./yt-dlp.exe", "--print", "filename", "-o", "%(title)s.%(ext)s", "-x",
				"--audio-format", "mp3", "--audio-quality", "0", "--no-keep-video", req)
			if cmd.Err != nil {
				fmt.Printf("There was some problem with creating the command for filename printing: %s\n", cmd.Err)
				return
			}

			var buf bytes.Buffer

			cmd.Stdout = &buf

			if err := cmd.Run(); err != nil {
				fmt.Printf("Couldn't get filename for request %s: %s\n", req, err)
				return
			}

			// buf now containse the name of the video file that has been dowloaded
			filename := buf.String()
			// We turn the video file's name into the MP3 file's name
			filename = fmt.Sprintf("%s.mp3", strings.Split(filename, ".")[0])
			fmt.Printf("Downloaded file name: %s\n", filename)

			fmt.Printf("Downloading %s\n", req)
			cmd = exec.Command("./yt-dlp.exe", "-o", "%(title)s.%(ext)s", "-x", "--audio-format", "mp3",
				"--audio-quality", "0", "--no-keep-video", req)
			if cmd.Err != nil {
				fmt.Printf("There was some problem with creating the yt-dlp command: %s\n", cmd.Err)
				return
			}

			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			if err := cmd.Run(); err != nil {
				fmt.Printf("Couldn't run yt-dpl for %s: %s\n", req, err)
				return
			}

			fmt.Printf("%s downloaded, adding %s to play queue.\n", req, filename)
			wd.playQueue <- filename
		}()
	}
}

func (wd *WaveDealer) RunPlayWorker() {
	queue := []string{}
	for {
		select {
		case file := <-wd.playQueue:
			fmt.Printf("New file in queue: %s\n", file)
			queue = append(queue, file)
		default:
			fmt.Println("Nothing new in the queue, sleeping")
			time.Sleep(250 * time.Millisecond)
		}

		if wd.isPlaying.Load() {
			continue
		}

		fmt.Printf("Previous file finished playing, trying to play the next in queue\n")
		fmt.Printf("Queue len: %d\n", len(queue))
		if len(queue) == 0 {
			fmt.Println("Nothing to play")
			continue
		}
		// Pick the new file to play
		toPlay := "./" + queue[0]
		// Remove the file to be played from the queue
		queue = queue[1:]

		go wd.playSong(toPlay)
	}
}

func (wd *WaveDealer) playSong(filePath string) {
	fmt.Printf("Playing %s next\n", filePath)
	cmd := exec.Command("./ffplay.exe", "-nodisp", "-autoexit", filePath)
	if cmd.Err != nil {
		fmt.Printf("Couldn't create command for ffplay: %v\n", cmd.Err)
		return
	}

	wd.isPlaying.Store(true)
	defer wd.isPlaying.Store(false)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Couldn't run ffplay: %v\n", err)
		return
	}
	fmt.Println("Returned from calling cmd.Run (ffplay)")
}
