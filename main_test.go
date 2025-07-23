// Unit tests for videofix

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func TestFilterTracks(t *testing.T) {
	tracks := []trackInfo{
		{ID: 1, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 2, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 3, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "spa"}},
		{ID: 4, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
			Language string `json:"language"`
		}{Language: ""}},
		{ID: 5, Type: "subtitles", CodecID: "S_TEXT/UTF8", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
	}

	testCases := []struct {
		name     string
		ttype    string
		codec    string
		lang     string
		expected []trackInfo
	}{
		{
			name:     "Filter by type audio",
			ttype:    "audio",
			codec:    "",
			lang:     "",
			expected: []trackInfo{tracks[0], tracks[1], tracks[2]},
		},
		{
			name:     "Filter by codec E-AC-3",
			ttype:    "",
			codec:    "E-AC-3",
			lang:     "",
			expected: []trackInfo{tracks[0], tracks[2]},
		},
		{
			name:     "Filter by language eng",
			ttype:    "",
			codec:    "",
			lang:     "eng",
			expected: []trackInfo{tracks[0], tracks[1], tracks[4]},
		},
		{
			name:     "Filter by type audio and lang eng",
			ttype:    "audio",
			codec:    "",
			lang:     "eng",
			expected: []trackInfo{tracks[0], tracks[1]},
		},
		{
			name:     "No match",
			ttype:    "video",
			codec:    "AAC",
			lang:     "",
			expected: []trackInfo{},
		},
		{
			name:     "Empty filters",
			ttype:    "",
			codec:    "",
			lang:     "",
			expected: tracks,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterTracks(tracks, tc.ttype, tc.codec, tc.lang)
			if len(result) == 0 && len(tc.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestLangAndDisposition(t *testing.T) {
	testCases := []struct {
		name                string
		track               trackInfo
		expectedLang        string
		expectedDisposition string
	}{
		{
			name: "Default language",
			track: trackInfo{
				Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"},
			},
			expectedLang:        "eng",
			expectedDisposition: "default",
		},
		{
			name: "Non-default language",
			track: trackInfo{
				Properties: struct {
					Language string `json:"language"`
				}{Language: "spa"},
			},
			expectedLang:        "spa",
			expectedDisposition: "-default",
		},
		{
			name: "Empty language",
			track: trackInfo{
				Properties: struct {
					Language string `json:"language"`
				}{Language: ""},
			},
			expectedLang:        "und",
			expectedDisposition: "-default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lang, disposition := langAndDisposition(tc.track)
			if lang != tc.expectedLang {
				t.Errorf("expected lang %s, got %s", tc.expectedLang, lang)
			}
			if disposition != tc.expectedDisposition {
				t.Errorf("expected disposition %s, got %s", tc.expectedDisposition, disposition)
			}
		})
	}
}

func TestTranscoderCmd(t *testing.T) {
	// Create a temporary file to simulate the input file
	tempFile, err := os.CreateTemp("", "test*.mkv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	inputFile := tempFile.Name()
	outputFile := "output.mkv"

	tracks := []trackInfo{
		{ID: 0, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
			Language string `json:"language"`
		}{Language: ""}},
		{ID: 1, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 2, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 3, Type: "subtitles", CodecID: "S_TEXT/UTF8", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
	}

	expectedArgs := []string{
		"ffmpeg",
		"-loglevel", "error",
		"-stats",
		"-i", inputFile,
		"-c:v", "copy",
		"-map", "0:v",
		"-map_chapters", "0",
		"-map_metadata", "0",
		"-map", "0:2",
		"-disposition:a:0", "default",
		"-c:a:0", "copy",
		"-map", "0:3",
		"-c:s:0", "copy",
		"-disposition:s:0", "default",
		"-max_interleave_delta", "0",
		"-y",
		"-f", "matroska",
		outputFile,
	}

	args := TranscoderCmd(inputFile, outputFile, tracks)

	// Create a map of the expected arguments for easier lookup
	expectedArgsMap := make(map[string]bool)
	for _, arg := range expectedArgs {
		expectedArgsMap[arg] = true
	}

	// Check if all expected arguments are present in the generated arguments
	for _, expected := range expectedArgs {
		found := false
		for _, actual := range args {
			if expected == actual {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected argument '%s' not found in the command", expected)
		}
	}

	// Check for extra arguments
	if len(args) != len(expectedArgs) {
		// Find the extra arguments
		extraArgs := []string{}
		for _, actual := range args {
			if !expectedArgsMap[actual] {
				extraArgs = append(extraArgs, actual)
			}
		}
		t.Errorf("Got extra arguments: %v", extraArgs)
	}

	fmt.Println(args)
}

func TestPrintHeader(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	header := "This is a test header\nWith multiple lines"
	printHeader(header)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	got := buf.String()

	expected := `=====================
This is a test header
With multiple lines
=====================
`
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
