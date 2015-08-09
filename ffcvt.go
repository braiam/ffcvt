////////////////////////////////////////////////////////////////////////////
// Porgram: FfCvt
// Purpose: ffmpeg convert wrapper tool
// Authors: Tong Sun (c) 2015, All rights reserved
////////////////////////////////////////////////////////////////////////////

/*

Transcodes all videos in the given directory and all of it's subdirectories
using ffmpeg.

*/

////////////////////////////////////////////////////////////////////////////
// Program start

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

////////////////////////////////////////////////////////////////////////////
// Global variables definitions

var (
	sprintf = fmt.Sprintf
	videos  []string
)

////////////////////////////////////////////////////////////////////////////
// Main

func main() {
	flag.Usage = Usage
	flag.Parse()

	// One mandatory arguments, either -d or -f
	if len(Opts.Directory)+len(Opts.File) < 1 {
		Usage()
	}
	getDefault()

	startTime := time.Now()
	if Opts.Directory != "" {
		filepath.Walk(Opts.Directory, visit)
		transcodeVideos()
	} else if Opts.File != "" {
		fmt.Printf("\n== Transcoding: %s\n", Opts.File)
		transcodeFile(Opts.File)
	}
	fmt.Printf("\nTranscoding completed in %s\n", time.Since(startTime))
}

////////////////////////////////////////////////////////////////////////////
// Function definitions

//==========================================================================
// Directory & files handling

func visit(path string, f os.FileInfo, err error) error {
	if f.IsDir() {
		return nil
	}

	appendVideo(path)
	return nil
}

// Append the video file to the list, unless it's encoded already
func appendVideo(fname string) {
	if fname[len(fname)-5:] == "_.mkv" {
		return
	}

	fext := strings.ToUpper(fname[len(fname)-4:])
	if strings.Index(Opts.Exts, fext) < 0 {
		return
	}

	if Opts.NoClobber && fileExist(getOutputName(fname)) {
		return
	}

	videos = append(videos, fname)
}

//==========================================================================
// Transcode handling

// Transcode videos in the global videos array
func transcodeVideos() {
	for i, inputName := range videos {
		fmt.Printf("\n== Transcoding [%d/%d]: '%s'\n   under %s\n",
			i+1, len(videos), filepath.Base(inputName), filepath.Dir(inputName))
		transcodeFile(inputName)
	}
}

func transcodeFile(inputName string) {
	startTime := time.Now()
	outputName := getOutputName(inputName)

	args := encodeParametersV(encodeParametersA(
		[]string{"-i", inputName}))
	if Opts.Force {
		args = append(args, "-y")
	}
	args = append(args, strings.Fields(Opts.OptExtra)...)
	args = append(args, flag.Args()...)
	args = append(args, outputName)
	debug(Opts.FFMpeg, 2)
	debug(strings.Join(args, " "), 1)

	if Opts.NoExec {
		fmt.Printf("%s: to execute -\n  %s %s\n",
			progname, Opts.FFMpeg, strings.Join(args, " "))
	} else {
		cmd := exec.Command(Opts.FFMpeg, args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			log.Printf("%s: Exec error - %s", progname, err.Error())
		}
		fmt.Printf("%s\n", out.String())
		time := time.Since(startTime)

		if err != nil {
			fmt.Println("Failed.")
		} else {
			originalSize := fileSize(inputName)
			transcodedSize := fileSize(outputName)
			sizeDifference := originalSize - transcodedSize

			fmt.Println("Done.")
			fmt.Printf("Org Size: %d KB\n", originalSize)
			fmt.Printf("New Size: %d KB\n", transcodedSize)
			fmt.Printf("Saved:    %d%% with %d KB\n",
				sizeDifference*100/originalSize, sizeDifference)
			fmt.Printf("Time: %v\n\n", time)
		}
	}

	return
}

// Returns the encode parameters for Audio
func encodeParametersA(args []string) []string {
	if Opts.AC {
		args = append(args, "-c:a", "copy")
		return args
	}
	if Opts.AN {
		args = append(args, "-an")
		return args
	}
	if Opts.A2Opus {
		Opts.AES = "libopus"
	}
	if Opts.AES != "" {
		args = append(args, "-c:a", Opts.AES)
	}
	if Opts.ABR != "" {
		args = append(args, "-b:a", Opts.ABR)
	}
	if Opts.AEA != "" {
		args = append(args, strings.Fields(Opts.AEA)...)
	}
	return args
}

// Returns the encode parameters for Video
func encodeParametersV(args []string) []string {
	if Opts.VC {
		args = append(args, "-c:v", "copy")
		return args
	}
	if Opts.VN {
		args = append(args, "-vn")
		return args
	}
	if Opts.V2X265 {
		Opts.VES = "libx265"
	}
	if Opts.VES != "" {
		args = append(args, "-c:v", Opts.VES)
	}
	if Opts.CRF != "" {
		if Opts.VES[:6] == "libx26" {
			args = append(args, "-"+Opts.VES[3:]+"-params", "crf="+Opts.CRF)
		}
	}
	if Opts.VEA != "" {
		args = append(args, strings.Fields(Opts.VEA)...)
	}
	return args
}

//==========================================================================
// Dealing with Files

// Returns true if the file exist
func fileExist(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil
}

// Returns the file size
func fileSize(fname string) int64 {
	stat, err := os.Stat(fname)
	checkError(err)

	return stat.Size() / 1024
}

// Replaces the file extension from the input string with _.mkv, and optionally Opts.Suffix
func getOutputName(input string) string {
	index := strings.LastIndex(input, ".")
	if index > 0 {
		input = input[:index]
	}
	return input + Opts.Suffix + "_.mkv"
}

func debug(input string, threshold int) {
	if !(Opts.Debug >= threshold) {
		return
	}
	print("] ")
	print(input)
	print("\n")
}

func checkError(err error) {
	if err != nil {
		log.Printf("%s: Fatal error - %s", progname, err.Error())
		os.Exit(1)
	}
}
