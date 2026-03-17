package wd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

type WaveDealer struct {
	ctx       *oto.Context
	player    *oto.Player
	requests  chan string
	playQueue chan string
}

func NewWaveDealer() (*WaveDealer, error) {
	var opts oto.NewContextOptions
	opts.ChannelCount = 2
	opts.SampleRate = 48000
	ctx, ready, err := oto.NewContext(&opts)
	if err != nil {
		return nil, fmt.Errorf("oto context couldn't be created: %+v", err)
	}

	// Wait for the device to become ready
	<-ready
	fmt.Println("Audio device ready")

	return &WaveDealer{
		ctx:       ctx,
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
	var currentFilePlaying *os.File
	var mp3Decoder *mp3.Decoder
	var err error
	for {
		select {
		case file := <-wd.playQueue:
			fmt.Printf("New file in queue: %s\n", file)
			queue = append(queue, file)
		default:
			fmt.Println("Nothing new in the queue, sleeping")
			time.Sleep(250 * time.Millisecond)
		}

		if wd.player == nil || !wd.player.IsPlaying() {
			// Clean up previously played file
			if currentFilePlaying != nil {
				fmt.Println("Cleaning up after last played song")
				currentFilePlaying.Close()
			}

			fmt.Printf("Queue len: %d\n", len(queue))
			if len(queue) == 0 {
				fmt.Println("Nothing to play")
				continue
			}
			// Pick the new file to play
			toPlay := queue[0]
			// Remove the file to be played from the queue
			queue = queue[1:]

			fmt.Printf("Playing %s next\n", toPlay)
			currentFilePlaying, err = os.Open(fmt.Sprintf("./%s", toPlay))
			if err != nil {
				fmt.Printf("Opening %s failed: %s\n", toPlay, err)
				continue
			}
			fmt.Printf("%s opened successfully\n", toPlay)

			// Decode file. This process is done as the file plays so it won't
			// load the whole thing into memory.
			mp3Decoder, err = mp3.NewDecoder(currentFilePlaying)
			if err != nil {
				fmt.Printf("mp3.NewDecoder for %s failed: %s\n", toPlay, err)
				continue
			}
			fmt.Printf("mp3Decoder successfully created for %s\n", toPlay)

			// Create a new 'player' that will handle our sound. Paused by default.
			wd.player = wd.ctx.NewPlayer(mp3Decoder)

			// Play starts playing the sound and returns without waiting for it (Play() is async).
			wd.player.Play()
		}
	}
}
