package validator

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ky0ryu/video-upload-service/internal/domain"
)

func TestVideoValidator_Validate(t *testing.T) {
	// MP4 magic bytes: 'ftypmp42' or similar. Mock the minimal header for the mime detector
	validMP4Header := append([]byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6d, 0x70, 0x34, 0x32}, []byte("mp4")...)
	invalidHeader := []byte("non-video plain text file contents")

	tests := []struct {
		name          string
		sizeLimitMB   int64
		videoFile     domain.VideoFile
		wantErr       bool
		errSubstrings []string
	}{
		{
			name:        "Valid MP4 upload",
			sizeLimitMB: 10,
			videoFile: domain.VideoFile{
				Video: domain.Video{
					OriginalFilename: "test_video.mp4",
				},
				File: bytes.NewReader(validMP4Header),
				Size: int64(len(validMP4Header)),
			},
			wantErr: false,
		},
		{
			name:        "File exceeds size limit",
			sizeLimitMB: 1, // 1 MB
			videoFile: domain.VideoFile{
				Video: domain.Video{
					OriginalFilename: "large_video.mp4",
				},
				File: bytes.NewReader(validMP4Header),
				Size: 2 * 1024 * 1024, // 2MB
			},
			wantErr:       true,
			errSubstrings: []string{"exceeds"},
		},
		{
			name:        "Unsupported file extensions",
			sizeLimitMB: 10,
			videoFile: domain.VideoFile{
				Video: domain.Video{
					OriginalFilename: "malicious_script.sh",
				},
				File: bytes.NewReader(validMP4Header),
				Size: int64(len(validMP4Header)),
			},
			wantErr:       true,
			errSubstrings: []string{"unsupported format"},
		},
		{
			name:        "Spoofed file format (MP4 extension but TXT content)",
			sizeLimitMB: 10,
			videoFile: domain.VideoFile{
				Video: domain.Video{
					OriginalFilename: "spoofed_extension.mp4",
				},
				File: bytes.NewReader(invalidHeader),
				Size: int64(len(invalidHeader)),
			},
			wantErr:       true,
			errSubstrings: []string{"unsupported video format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vv := NewVideoValidator(tt.sizeLimitMB)
			err := vv.Validate(&tt.videoFile)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && len(tt.errSubstrings) > 0 {
				matched := false
				for _, sub := range tt.errSubstrings {
					if strings.Contains(err.Error(), sub) {
						matched = true
						break
					}
				}
				if !matched {
					t.Errorf("expected error containing one of %v, got: %v", tt.errSubstrings, err)
				}
			}
		})
	}
}
