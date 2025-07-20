// Fix common problems in MKV files:
//
// - Convert EAC3 audio to AAC to avoid issues with players.
// - All other tracks and metadata is copied from the original file.
//
// (C) Jul/2025 by Marco Paganini <paganini@paganini.net>

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	outputSuffix = "_with_aac"
	eac3Codec    = "E-AC-3"
	aacCodec     = "AAC"
	aacBitrate   = "256k"
	defaultLang  = "eng"
	mkvAudioType = "audio"
	mkvSubType   = "subtitles"
)

// trackInfo holds information about a track from mkvmerge.
type trackInfo struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	CodecID    string `json:"codec"`
	Properties struct {
		Language string `json:"language"`
	} `json:"properties"`
}

// mkvInfo holds the top-level JSON structure from mkvmerge.
type mkvInfo struct {
	Tracks []trackInfo `json:"tracks"`
}

// checkRequirements returns an error if any of the required programs
// are not installed in the system.
func checkRequirements() error {
	if _, err := exec.LookPath("mkvmerge"); err != nil {
		return fmt.Errorf("mkvmerge not found. Please install the mkvtoolnix package")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found. Please install the ffmpeg package")
	}
	return nil
}

// readTracks returns a map of all tracks in the input file using
// mkvmerge --identify.
func readTracks(inputFile string) ([]trackInfo, error) {
	// Get track information using mkvmerge.
	cmd := exec.Command("mkvmerge", "--identify", "-F", "json", inputFile)
	output, err := cmd.Output()
	if err != nil {
		return []trackInfo{}, fmt.Errorf("error running mkvmerge: %w", err)
	}

	var info mkvInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return []trackInfo{}, fmt.Errorf("error parsing mkvmerge JSON output: %w", err)
	}

	tracks := []trackInfo{}
	for _, track := range info.Tracks {
		t := trackInfo{
			ID:         track.ID,
			Type:       track.Type,
			CodecID:    track.CodecID,
			Properties: track.Properties,
		}
		tracks = append(tracks, t)
	}

	return tracks, nil
}

func filterAudioTracks(tracks []trackInfo, codec string) []trackInfo {
	var ret []trackInfo
	for _, track := range tracks {
		if track.Type == mkvAudioType && track.CodecID == codec {
			ret = append(ret, track)
		}
	}
	return ret
}

func langAndDisposition(track trackInfo) (string, string) {
	lang := "und"
	disposition := "-default"

	if track.Properties.Language != "" {
		lang = track.Properties.Language
	}
	if lang == defaultLang {
		disposition = "default"
	}
	return lang, disposition
}

// transcoderCmd creates an ffmpeg command to transcode EAC3 tracks to AAC
// and copy the remaining data.
func TranscoderCmd(inputFile string, outputFile string, tracks []trackInfo) []string {
	// Create the ffmpeg command line.
	args := []string{
		"ffmpeg",
		"-loglevel", "error",
		"-stats",
		"-i", inputFile,
		"-c:v", "copy", // Default codec for video = copy.
		"-map", "0:v", // Copy all video tracks first.
		"-map_chapters", "0", // Copy all chapters
		"-map_metadata", "0", // Copy all metadata
	}

	// Add AAC conversion for each EAC3 track.
	// Copy non-EAC3 audio tracks directly.
	// Copy subtitle tracks directly.
	// Set the default flag on "eng" tracks.

	// IMPORTANT: The -map command uses the INPUT track number while the
	// -c:a:TRACK command uses the relative OUTPUT track number.
	audiotrack := 0
	subtrack := 0

	// Run first for audio tracks, then subtitle tracks so we maintain the
	// A/V/S order in the output file.
	for _, track := range tracks {
		if track.Type != mkvAudioType {
			continue
		}

		lang, disposition := langAndDisposition(track)

		// Map track for output.
		args = append(args, "-map", fmt.Sprintf("0:%d", track.ID))

		// Transcode or copy.
		if track.CodecID == eac3Codec {
			args = append(args,
				fmt.Sprintf("-c:a:%d", audiotrack), "aac",
				fmt.Sprintf("-b:a:%d", audiotrack), aacBitrate,
				fmt.Sprintf("-metadata:s:a:%d", audiotrack), fmt.Sprintf("title=AAC Audio (%s)", lang))
		} else {
			args = append(args, fmt.Sprintf("-c:a:%d", audiotrack), "copy")
		}
		args = append(args, fmt.Sprintf("-disposition:a:%d", audiotrack), disposition)
		audiotrack++
	}

	for _, track := range tracks {
		if track.Type != mkvSubType {
			continue
		}

		_, disposition := langAndDisposition(track)

		// Map track for output, copy and set disposition.
		args = append(args,
			"-map", fmt.Sprintf("0:%d", track.ID),
			fmt.Sprintf("-c:s:%d", subtrack), "copy",
			fmt.Sprintf("-disposition:s:%d", subtrack), disposition)
		subtrack++
	}

	// Final arguments.
	args = append(args,
		"-max_interleave_delta", "0",
		"-y",
		"-f", "matroska",
		outputFile)

	return args
}

// TranscodeEAC3 converts EAC3 audio to AAC audio in the input file.
func TranscodeEAC3(mkvfile string) error {
	// Check if the input file exists
	if _, err := os.Stat(mkvfile); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", mkvfile)
	}

	// Generate the output filename
	filename := filepath.Base(mkvfile)
	dirname := filepath.Dir(mkvfile)
	extension := strings.ToLower(filepath.Ext(filename))
	filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	if extension != ".mkv" {
		return fmt.Errorf("not an MKV file: %s", mkvfile)
	}

	outputFile := filepath.Join(dirname, fmt.Sprintf("%s%s%s.TMP", filenameNoExt, outputSuffix, extension))

	// Do not proceed if our temp file already exists.  This may mean another
	// instance running or some other condition that needs to be investigated.
	if _, err := os.Stat(outputFile); err == nil {
		return fmt.Errorf("output file '%s' already exists. Skipping", outputFile)
	}

	tracks, err := readTracks(mkvfile)
	if err != nil {
		return err
	}
	log.Println("=== List of input tracks ===")
	for _, track := range tracks {
		log.Printf("  - ID: %d (%s), Codec: %s, Language: %s", track.ID, track.Type, track.CodecID, track.Properties.Language)
	}

	// If no EAC3 tracks found, nothing to do.
	if len(filterAudioTracks(tracks, eac3Codec)) == 0 {
		return fmt.Errorf("no EAC3 audio tracks found in %s. No conversion needed", mkvfile)
	}

	// If AAC tracks found, we assume we already transcoded this file.
	if len(filterAudioTracks(tracks, aacCodec)) > 0 {
		return fmt.Errorf("found AAC audio tracks in %s. No conversion needed", mkvfile)
	}

	tcmd := TranscoderCmd(mkvfile, outputFile, tracks)
	log.Printf("Executing command:\n%s\n", "'"+strings.Join(tcmd, "' '")+"'")

	// Execute the ffmpeg command, send all output to stderr.
	cmd := exec.Command(tcmd[0], tcmd[1:]...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(outputFile)
		return fmt.Errorf("ffmpeg conversion failed for %s: %v", mkvfile, err)
	}

	// Move the output file to the input file
	if _, err := os.Stat(mkvfile); os.IsNotExist(err) {
		return fmt.Errorf("original file (%s) no longer exists after transcoding", mkvfile)
	}
	if err := os.Rename(outputFile, mkvfile); err != nil {
		return fmt.Errorf("failed to move '%s' to '%s': %v", outputFile, mkvfile, err)
	}
	return nil
}

func main() {
	// No date & time on logs.
	log.SetFlags(0)
	progname := filepath.Base(os.Args[0])

	if len(os.Args) < 2 {
		log.Fatalf("use: %s input_file.mkv...", progname)
	}

	if err := checkRequirements(); err != nil {
		log.Fatalf("%s:missing requirements: %v", progname, err)
	}

	for _, f := range os.Args[1:] {
		if err := TranscodeEAC3(f); err != nil {
			log.Printf("%s: ERROR(%s): %v\n", progname, f, err)
			continue
		}
		log.Printf("%s: Operation successful.\n", f)
	}
	os.Exit(0)
}
