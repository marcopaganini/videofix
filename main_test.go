package main

import (
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

func TestLangAndDisposition(t *testing.T) {
	// Helper function to set the value of a string pointer
	setStringPtr := func(s string) *string {
		return &s
	}

	testCases := []struct {
		name                string
		track               trackInfo
		optLang             *string
		expectedLang        string
		expectedDisposition string
	}{
		{
			name: "Language matches optLang",
			track: trackInfo{Properties: struct {
				Language string `json:"language"`
			}{Language: "eng"}},
			optLang:             setStringPtr("eng"),
			expectedLang:        "eng",
			expectedDisposition: "default",
		},
		{
			name: "Language does not match optLang",
			track: trackInfo{Properties: struct {
				Language string `json:"language"`
			}{Language: "spa"}},
			optLang:             setStringPtr("eng"),
			expectedLang:        "spa",
			expectedDisposition: "-default",
		},
		{
			name: "Empty language property",
			track: trackInfo{Properties: struct {
				Language string `json:"language"`
			}{Language: ""}},
			optLang:             setStringPtr("eng"),
			expectedLang:        "und",
			expectedDisposition: "-default",
		},
		{
			name: "Language is und",
			track: trackInfo{Properties: struct {
				Language string `json:"language"`
			}{Language: "und"}},
			optLang:             setStringPtr("eng"),
			expectedLang:        "und",
			expectedDisposition: "-default",
		},
		{
			name: "optLang is not default",
			track: trackInfo{Properties: struct {
				Language string `json:"language"`
			}{Language: "por"}},
			optLang:             setStringPtr("por"),
			expectedLang:        "por",
			expectedDisposition: "default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the global optLang for the duration of this test case
			originalOptLang := optLang
			optLang = tc.optLang
			defer func() { optLang = originalOptLang }()

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
