// Fix common problems in MKV files:
//
// - Convert EAC3 audio to AAC to avoid issues with players.
// - If the file has equivalent EAC3/AAC tracks, remove the EAC3 version.
// - Set all "eng" tracks to be the default tracks.
// - All other tracks and metadata is copied from the original file.
//
// (C) Jul/2025 by Marco Paganini <paganini@paganini.net>

package main

import (
	"encoding/json"
	"flag"
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
	mkvAudioType = "audio"
	mkvSubType   = "subtitles"
)

var defaultLang = flag.String("lang", "eng", "Default language for audio and subtitle tracks")

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

// filterTracks returns a list of tracks filtered by type, codec and language. If any of the
// parameters is blank, ignore it during comparison.
func filterTracks(tracks []trackInfo, ttype string, codec string, lang string) []trackInfo {
	var ret []trackInfo
	for _, track := range tracks {
		if ttype != "" && track.Type != ttype {
			continue
		}
		if codec != "" && track.CodecID != codec {
			continue
		}
		if lang != "" && track.Properties.Language != lang {
			continue
		}
		ret = append(ret, track)
	}
	return ret
}

func langAndDisposition(track trackInfo) (string, string) {
	lang := "und"
	disposition := "-default"

	if track.Properties.Language != "" {
		lang = track.Properties.Language
	}
	if lang == *defaultLang {
		disposition = "default"
	}
	return lang, disposition
}

// printHeader prints a header using the passed string. The string is broken down by
// newlines and a separator is printed before the first line and after the first line
// to match the longest line in the string.
func printHeader(header string) {
	lines := strings.Split(header, "\n")

	var maxlen int
	for _, line := range lines {
		maxlen = max(maxlen, len(line))
	}

	fmt.Println(strings.Repeat("=", maxlen))
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println(strings.Repeat("=", maxlen))
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

	printHeader("Processing AUDIO tracks")

	for _, track := range tracks {
		trackAction := ""

		if track.Type != mkvAudioType {
			continue
		}
		trackData := fmt.Sprintf("%d: codec=%s lang=%s", track.ID, track.CodecID, track.Properties.Language)

		lang, disposition := langAndDisposition(track)

		// Transcode or copy.
		if track.CodecID == eac3Codec {
			// If we have an equivalent AAC track with the same language and
			// language is not "und", ignore that the EAC3 track.
			if lang != "und" {
				equivalent := filterTracks(tracks, mkvAudioType, aacCodec, lang)
				if len(equivalent) > 0 {
					trackAction = fmt.Sprintf("found %d AAC equivalent audio track(s). Skipping.", len(equivalent))
					log.Println("  " + trackData + ": " + trackAction)
					continue
				}
			}
			trackAction = "selected for EAC3 --> AAC conversion"
			args = append(args,
				fmt.Sprintf("-c:a:%d", audiotrack), "aac",
				fmt.Sprintf("-b:a:%d", audiotrack), aacBitrate,
				fmt.Sprintf("-metadata:s:a:%d", audiotrack), fmt.Sprintf("title=AAC Audio (%s)", lang))
		} else {
			trackAction = "selected for COPY."
			args = append(args, fmt.Sprintf("-c:a:%d", audiotrack), "copy")
		}
		args = append(args,
			"-map", fmt.Sprintf("0:%d", track.ID),
			fmt.Sprintf("-disposition:a:%d", audiotrack), disposition)

		log.Println("  " + trackData + ": " + trackAction)
		audiotrack++
	}

	printHeader("Processing SUBTITLES tracks")

	for _, track := range tracks {
		trackAction := ""

		if track.Type != mkvSubType {
			continue
		}

		trackData := fmt.Sprintf("%d: codec=%s lang=%s", track.ID, track.CodecID, track.Properties.Language)

		_, disposition := langAndDisposition(track)

		// Map track for output, copy and set disposition.
		args = append(args,
			"-map", fmt.Sprintf("0:%d", track.ID),
			fmt.Sprintf("-c:s:%d", subtrack), "copy",
			fmt.Sprintf("-disposition:s:%d", subtrack), disposition)

		trackAction = "selected for COPY."
		log.Println("  " + trackData + ": " + trackAction)
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

	printHeader(fmt.Sprintf("File: %s\nList of input tracks", mkvfile))

	for _, track := range tracks {
		log.Printf("  - ID: %d (%s), Codec: %s, Language: %s", track.ID, track.Type, track.CodecID, track.Properties.Language)
	}

	tcmd := TranscoderCmd(mkvfile, outputFile, tracks)
	printHeader("Executing command")
	log.Println("'" + strings.Join(tcmd, "' '") + "'")

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

// usage prints a customized usage message.
func usage() {
	progname := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage: %s [options] <input_file.mkv>...\n\n", progname)
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func main() {
	// No date & time on logs.
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := checkRequirements(); err != nil {
		progname := filepath.Base(os.Args[0])
		log.Fatalf("%s:missing requirements: %v", progname, err)
	}

	for _, f := range flag.Args() {
		if err := TranscodeEAC3(f); err != nil {
			progname := filepath.Base(os.Args[0])
			log.Printf("%s: ERROR(%s): %v\n", progname, f, err)
			continue
		}
		log.Printf("%s: Operation successful.\n", f)
	}
	os.Exit(0)
}