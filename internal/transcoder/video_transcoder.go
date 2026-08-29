package transcoder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/ky0ryu/video-upload-service/internal/domain"
	"github.com/ky0ryu/video-upload-service/internal/task"
)

type VideoTranscoder struct {
	Repo domain.VideoRepository
}

func (vt *VideoTranscoder) TranscodeVideo(ctx context.Context, t *asynq.Task) error {
	var p task.TranscodeVideoPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// malformed payload
		return fmt.Errorf("invalid payload: %w", asynq.SkipRetry)
	}

	// update video status to "transcoding"
	if updErr := vt.Repo.UpdateVideoState(ctx, p.VideoID, domain.VideoTranscoding); updErr != nil {
		fmt.Printf("failed to update state: %v", updErr)
	}

	path := strings.TrimSuffix(p.SaveDir, filepath.Ext(p.SaveDir)) + ".m3u8"

	fmt.Printf("transcode video: %v : %v", p.VideoID, path)
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", p.SaveDir,
		"-vf", "scale=-2:720",
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "aac",
		"-hls_time", "4", "-hls_list_size", "0",
		"-f", "hls",
		path,
	)

	// capture the cmd error thru a buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// update video status to "failed"
		if updErr := vt.Repo.UpdateVideoState(ctx, p.VideoID, domain.VideoFailed); updErr != nil {
			fmt.Printf("failed to update state: %v", updErr)
		}
		return fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}

	// update video status to "ready"
	if updErr := vt.Repo.UpdateVideoState(ctx, p.VideoID, domain.VideoReady); updErr != nil {
		fmt.Printf("failed to update state: %v", updErr)
	}
	return nil
}
