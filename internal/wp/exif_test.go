package wp_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestExifFrac2Dec(t *testing.T) {
	if got := wp.ExifFrac2Dec("1/2"); got != 0.5 {
		t.Fatalf("1/2=%v", got)
	}
	if got := wp.ExifFrac2Dec("5.5"); got != 5.5 {
		t.Fatalf("5.5=%v", got)
	}
	if got := wp.ExifFrac2Dec("1/0"); got != 0 {
		t.Fatalf("1/0=%v", got)
	}
	if got := wp.ExifFrac2Dec("nope"); got != 0 {
		t.Fatalf("nope=%v", got)
	}
}

func TestExifDate2TS(t *testing.T) {
	ts := wp.ExifDate2TS("2024:06:15 12:00:00")
	if ts <= 0 {
		t.Fatalf("timestamp=%d", ts)
	}
	if wp.ExifDate2TS("bad") != 0 {
		t.Fatal("bad date should be 0")
	}
}

func TestReadImageMetadataJPEGExif(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "with-exif.jpg")
	if err := os.WriteFile(path, buildJPEGWithExif(), 0o644); err != nil {
		t.Fatal(err)
	}
	meta, ok := wp.ReadImageMetadata(nil, path)
	if !ok {
		t.Fatal("expected metadata")
	}
	if meta["orientation"] != 6 {
		t.Fatalf("orientation=%v", meta["orientation"])
	}
	if meta["camera"] != "GoPressCam" {
		t.Fatalf("camera=%v", meta["camera"])
	}
	if meta["aperture"] != 2.8 {
		t.Fatalf("aperture=%v", meta["aperture"])
	}
	if meta["iso"] != float64(200) {
		t.Fatalf("iso=%v", meta["iso"])
	}
	if meta["title"] != "Short title" {
		t.Fatalf("title=%v", meta["title"])
	}
	if meta["credit"] != "Ada" {
		t.Fatalf("credit=%v", meta["credit"])
	}
	if meta["created_timestamp"] == int64(0) || meta["created_timestamp"] == 0 {
		t.Fatalf("created_timestamp=%v", meta["created_timestamp"])
	}
}

// buildJPEGWithExif encodes a tiny JPEG then injects an APP1 Exif segment after SOI.
func buildJPEGWithExif() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 20), B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic(err)
	}
	jpegBytes := buf.Bytes()
	if len(jpegBytes) < 4 || jpegBytes[0] != 0xFF || jpegBytes[1] != 0xD8 {
		panic("bad jpeg")
	}
	tiff := buildExifTIFF()
	app1Body := append([]byte("Exif\x00\x00"), tiff...)
	seg := make([]byte, 2+len(app1Body))
	binary.BigEndian.PutUint16(seg[:2], uint16(len(seg)))
	copy(seg[2:], app1Body)
	var out []byte
	out = append(out, 0xFF, 0xD8)
	out = append(out, 0xFF, 0xE1)
	out = append(out, seg...)
	out = append(out, jpegBytes[2:]...)
	return out
}

func buildExifTIFF() []byte {
	// Little-endian TIFF with IFD0 (desc, model, orientation, artist, exif ptr)
	// and Exif IFD (FNumber, ISO, DateTimeDigitized).
	order := binary.LittleEndian
	buf := make([]byte, 0, 512)
	buf = append(buf, 'I', 'I', 0x2A, 0x00) // magic
	ifd0Off := 8
	off := make([]byte, 4)
	order.PutUint32(off, uint32(ifd0Off))
	buf = append(buf, off...)

	// We'll assemble IFDs with values stored after both IFDs.
	type entry struct {
		tag  uint16
		typ  uint16
		cnt  uint32
		data []byte // already encoded inline or as pointed blob
		blob []byte // optional overflow
	}

	desc := append([]byte("Short title"), 0)
	model := append([]byte("GoPressCam"), 0)
	artist := append([]byte("Ada"), 0)
	dt := append([]byte("2024:06:15 12:00:00"), 0)
	fnum := make([]byte, 8)
	order.PutUint32(fnum[0:4], 14)
	order.PutUint32(fnum[4:8], 5) // 2.8

	// Placeholder exif IFD offset filled later.
	exifPtr := make([]byte, 4)

	ifd0Entries := []entry{
		{0x010E, 2, uint32(len(desc)), nil, desc},
		{0x0110, 2, uint32(len(model)), nil, model},
		{0x0112, 3, 1, putU16LE(6), nil},
		{0x013B, 2, uint32(len(artist)), nil, artist},
		{0x8769, 4, 1, exifPtr, nil},
	}
	exifEntries := []entry{
		{0x829D, 5, 1, nil, fnum},
		{0x8827, 3, 1, putU16LE(200), nil},
		{0x9004, 2, uint32(len(dt)), nil, dt},
	}

	// Layout: header(8) | IFD0 | next0(4) | ExifIFD | next1(4) | blobs
	ifd0Size := 2 + len(ifd0Entries)*12 + 4
	exifOff := ifd0Off + ifd0Size
	exifSize := 2 + len(exifEntries)*12 + 4
	blobOff := exifOff + exifSize
	order.PutUint32(exifPtr, uint32(exifOff))

	// Assign blob offsets for overflow values.
	cur := blobOff
	assign := func(e *entry) {
		if e.blob == nil {
			if e.data == nil {
				e.data = []byte{0, 0, 0, 0}
			}
			for len(e.data) < 4 {
				e.data = append(e.data, 0)
			}
			return
		}
		if len(e.blob) <= 4 {
			e.data = make([]byte, 4)
			copy(e.data, e.blob)
			return
		}
		e.data = make([]byte, 4)
		order.PutUint32(e.data, uint32(cur))
		cur += len(e.blob)
	}
	for i := range ifd0Entries {
		assign(&ifd0Entries[i])
	}
	for i := range exifEntries {
		assign(&exifEntries[i])
	}

	writeIFD := func(entries []entry) {
		cnt := make([]byte, 2)
		order.PutUint16(cnt, uint16(len(entries)))
		buf = append(buf, cnt...)
		for _, e := range entries {
			rec := make([]byte, 12)
			order.PutUint16(rec[0:2], e.tag)
			order.PutUint16(rec[2:4], e.typ)
			order.PutUint32(rec[4:8], e.cnt)
			copy(rec[8:12], e.data[:4])
			buf = append(buf, rec...)
		}
		buf = append(buf, 0, 0, 0, 0) // next IFD
	}
	writeIFD(ifd0Entries)
	writeIFD(exifEntries)
	for _, e := range ifd0Entries {
		if len(e.blob) > 4 {
			buf = append(buf, e.blob...)
		}
	}
	for _, e := range exifEntries {
		if len(e.blob) > 4 {
			buf = append(buf, e.blob...)
		}
	}
	return buf
}

func putU16LE(v uint16) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint16(b[0:2], v)
	return b
}
