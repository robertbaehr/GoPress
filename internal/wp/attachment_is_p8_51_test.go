package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P8.51: AttachmentIs image/audio/video via mime prefix; AttachmentIsImage wrapper.
func TestAttachmentIsP851(t *testing.T) {
	c := mediaBoot(t)
	if !wp.AttachmentIs(c, "image", 5) {
		t.Fatal("fixture attachment 5 should be image")
	}
	if !wp.AttachmentIsImage(c, 5) {
		t.Fatal("AttachmentIsImage")
	}
	if wp.AttachmentIs(c, "audio", 5) || wp.AttachmentIs(c, "video", 5) {
		t.Fatal("image is not audio/video")
	}
	if wp.AttachmentIs(c, "image", 0) || wp.AttachmentIs(nil, "image", 5) {
		t.Fatal("zero/nil guards")
	}
	if wp.AttachmentIs(c, "image", 1) {
		t.Fatal("non-attachment post")
	}
}

// P8.51: AudioExtensions / VideoExtensions filters.
func TestAudioVideoExtensionsP851(t *testing.T) {
	c := mediaBoot(t)
	audio := wp.AudioExtensions(c)
	if len(audio) < 3 || audio[0] != "mp3" {
		t.Fatalf("audio=%v", audio)
	}
	video := wp.VideoExtensions(c)
	if len(video) < 3 || video[0] != "mp4" {
		t.Fatalf("video=%v", video)
	}
	c.Hooks.AddFilter("wp_audio_extensions", func(value any, _ ...any) any {
		return []string{"mp3", "opus"}
	}, 10, 1)
	got := wp.AudioExtensions(c)
	if len(got) != 2 || got[1] != "opus" {
		t.Fatalf("filtered audio=%v", got)
	}
}
