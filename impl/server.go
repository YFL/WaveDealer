package impl

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/YFL/WaveDealer/api"
	"github.com/ebitengine/oto/v3"
	"github.com/gin-gonic/gin"
	"github.com/hajimehoshi/go-mp3"
	"go.senan.xyz/taglib"
)

type Server struct {
}

func (s *Server) RequestYoutubeSong(c *gin.Context) {
	var bodyJson api.RequestYoutubeSongJSONRequestBody
	if err := c.BindJSON(&bodyJson); err != nil {
		fmt.Printf("Couldn't bind the request body to api.RequestYoutubeSongJSONRequestBody: %s\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("./yt-dlp.exe", "--print", "filename", "-o", "%(title)s.%(ext)s", "-x", "--audio-format", "mp3", "--audio-quality", "0", "--no-keep-video", bodyJson.Url)
	if cmd.Err != nil {
		fmt.Printf("There was some problem with creating the command for filename printing: %s\n", cmd.Err)
		c.Status(http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer

	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		fmt.Printf("Couldn't get filename")
		c.Status(http.StatusInternalServerError)
		return
	}

	filename := buf.String()
	filename = fmt.Sprintf("%s.mp3", strings.Split(filename, ".")[0])
	fmt.Printf("Downloaded file name: %s\n", filename)

	cmd = exec.Command("./yt-dlp.exe", "-o", "%(title)s.%(ext)s", "-x", "--audio-format", "mp3", "--audio-quality", "0", "--no-keep-video", bodyJson.Url)
	if cmd.Err != nil {
		fmt.Printf("There was some problem with creating the yt-dlp command: %s\n", cmd.Err)
		c.Status(http.StatusInternalServerError)
		return
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		fmt.Printf("Couldn't run yt-dpl: %s\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	file, err := os.Open(fmt.Sprintf("./%s", filename))
	if err != nil {
		panic("opening my-file.mp3 failed: " + err.Error())
	}

	// Decode file. This process is done as the file plays so it won't
	// load the whole thing into memory.
	decodedMp3, err := mp3.NewDecoder(file)
	if err != nil {
		panic("mp3.NewDecoder failed: " + err.Error())
	}

	// Prepare an Oto context (this will use your default audio device) that will
	// play all our sounds. Its configuration can't be changed later.

	op := &oto.NewContextOptions{}
	properties, err := taglib.ReadProperties(fmt.Sprintf("./%s", filename))

	// Usually 44100 or 48000. Other values might cause distortions in Oto
	op.SampleRate = int(properties.SampleRate)

	// Number of channels (aka locations) to play sounds from. Either 1 or 2.
	// 1 is mono sound, and 2 is stereo (most speakers are stereo).
	op.ChannelCount = int(properties.Channels)

	// Format of the source. go-mp3's format is signed 16bit integers.
	op.Format = oto.FormatSignedInt16LE

	// Remember that you should **not** create more than one context
	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		panic("oto.NewContext failed: " + err.Error())
	}
	// It might take a bit for the hardware audio devices to be ready, so we wait on the channel.
	<-readyChan

	// Create a new 'player' that will handle our sound. Paused by default.
	player := otoCtx.NewPlayer(decodedMp3)

	// Play starts playing the sound and returns without waiting for it (Play() is async).
	player.Play()

	// We can wait for the sound to finish playing using something like this
	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}
	// Close file only after you finish playing
	file.Close()

}
