package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFilterTracks(t *testing.T) {
	tracks := []trackInfo{
		{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 2, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 3, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
			Language string `json:"language"`
		}{Language: "und"}},
		{ID: 4, Type: "subtitles", CodecID: "S_HDMV/PGS", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 5, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "spa"}},
	}

	testCases := []struct {
		name     string
		ttype    string
		codec    string
		lang     string
		expected []trackInfo
	}{
		{
			name:  "Filter by ttype audio",
			ttype: "audio",
			expected: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 2, Type: "audio", CodecID: "E-AC-3", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 5, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "spa"}},
			},
		},
		{
			name:  "Filter by codec AAC",
			codec: "AAC",
			expected: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 5, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "spa"}},
			},
		},
		{
			name: "Filter by lang eng",
			lang: "eng",
			expected: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 2, Type: "audio", CodecID: "E-AC-3", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 4, Type: "subtitles", CodecID: "S_HDMV/PGS", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
			},
		},
		{
			name:  "Filter by ttype audio and lang eng",
			ttype: "audio",
			lang:  "eng",
			expected: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
				{ID: 2, Type: "audio", CodecID: "E-AC-3", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
			},
		},
		{
			name:  "Filter by ttype audio, codec AAC, and lang eng",
			ttype: "audio",
			codec: "AAC",
			lang:  "eng",
			expected: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "eng"}},
			},
		},
		{
			name:     "No matching tracks",
			ttype:    "video",
			lang:     "spa",
			expected: []trackInfo{},
		},
		{
			name:     "Empty filters",
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
				t.Errorf("expected:\n%v\ngot:\n%v", tc.expected, result)
			}
		})
	}
}

func TestPruneOK(t *testing.T) {
	tracks := []trackInfo{
		{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 2, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "por"}},
		{ID: 3, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
			Language string `json:"language"`
		}{Language: "und"}},
		{ID: 4, Type: "subtitles", CodecID: "S_HDMV/PGS", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 5, Type: "subtitles", CodecID: "S_HDMV/PGS", Properties: struct {
			Language string `json:"language"`
		}{Language: "por"}},
	}

	testCases := []struct {
		name          string
		tracks        []trackInfo
		defaultLang   string
		expectErr     bool
		expectedError string
	}{
		{
			name:          "successful pruning",
			tracks:        tracks,
			defaultLang:   "eng",
			expectedError: "",
		},
		{
			name: "Pruning would remove all audio tracks",
			tracks: []trackInfo{
				{ID: 1, Type: "audio", CodecID: "AAC", Properties: struct {
					Language string `json:"language"`
				}{Language: "spa"}},
				{ID: 2, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
					Language string `json:"language"`
				}{Language: "und"}},
			},
			defaultLang:   "eng",
			expectErr:     true,
			expectedError: "pruning would remove all audio tracks from the output",
		},
		{
			name:        "No tracks pruned",
			tracks:      tracks,
			defaultLang: "por",
			expectErr:   false,
		},
		{
			name:        "Empty track list",
			tracks:      []trackInfo{},
			defaultLang: "eng",
			expectErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pruneOK(tc.tracks, tc.defaultLang)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, but got none")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error '%s', but got '%s'", tc.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTranscoderCmd(t *testing.T) {
	tracks := []trackInfo{
		{ID: 1, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 2, Type: "audio", CodecID: "AAC", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 3, Type: "video", CodecID: "V_MPEG4/ISO/AVC", Properties: struct {
			Language string `json:"language"`
		}{Language: ""}},
		{ID: 4, Type: "subtitles", CodecID: "S_HDMV/PGS", Properties: struct {
			Language string `json:"language"`
		}{Language: "eng"}},
		{ID: 5, Type: "audio", CodecID: "E-AC-3", Properties: struct {
			Language string `json:"language"`
		}{Language: "spa"}},
	}

	testCases := []struct {
		name       string
		tracks     []trackInfo
		doPrune    bool
		optlang    string
		inputFile  string
		outputFile string
		expected   []string
	}{
		{
			name:       "EAC3 to AAC conversion",
			tracks:     tracks,
			doPrune:    false,
			optlang:    "eng",
			inputFile:  "input.mkv",
			outputFile: "output.mkv",
			expected: []string{
				"ffmpeg", "-loglevel", "error", "-stats", "-i", "input.mkv",
				"-c:v", "copy", "-map", "0:v", "-map_chapters", "0", "-map_metadata", "0",
				"-c:a:0", "copy", "-map", "0:2", "-disposition:a:0", "default",
				"-c:a:1", "aac", "-b:a:1", "256k", "-metadata:s:a:1", "title=AAC Audio (spa)", "-map", "0:5", "-disposition:a:1", "-default",
				"-map", "0:4", "-c:s:0", "copy", "-disposition:s:0", "default",
				"-max_interleave_delta", "0", "-y", "-f", "matroska", "output.mkv",
			},
		},
		{
			name:       "Pruning enabled",
			tracks:     tracks,
			doPrune:    true,
			optlang:    "eng",
			inputFile:  "input.mkv",
			outputFile: "output.mkv",
			expected: []string{
				"ffmpeg", "-loglevel", "error", "-stats", "-i", "input.mkv",
				"-c:v", "copy", "-map", "0:v", "-map_chapters", "0", "-map_metadata", "0",
				"-c:a:0", "copy", "-map", "0:2", "-disposition:a:0", "default",
				"-map", "0:4", "-c:s:0", "copy", "-disposition:s:0", "default",
				"-max_interleave_delta", "0", "-y", "-f", "matroska", "output.mkv",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := transcoderCmd(tc.inputFile, tc.outputFile, tc.tracks, tc.doPrune, tc.optlang)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected:\n%v\ngot:\n%v", tc.expected, result)
			}
		})
	}
}

func TestFindVideoFiles(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"test1.mkv",
		"test2.mkv",
		"test3.mp4",
		"other.txt",
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f, err)
		}
	}

	expected := []string{
		filepath.Join(tempDir, "test1.mkv"),
		filepath.Join(tempDir, "test2.mkv"),
	}

	result, err := findVideoFiles(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(result))
	}

	// filepath.Walk order is lexicographical
	for i, f := range expected {
		if result[i] != f {
			t.Errorf("expected file %s, got %s", f, result[i])
		}
	}
}
