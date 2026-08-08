package task

const TypeTranscodeVideoType = "video:transcode"

type TranscodeVideoPayload struct {
	VideoID string `json:"video_id"`
	SaveDir string `json:"save_dir"`
}
