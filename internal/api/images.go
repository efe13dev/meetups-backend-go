package api

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// processAndSaveImage decodifica la imagen subida, la redimensiona a ancho 1000px si lo supera,
// y la guarda con un nombre UUID v4 en uploadsDir sin depender de CGo/libvips (sharp).
func (s *Server) processAndSaveImage(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, 10<<20)) // máximo 10 MB
	if err != nil {
		return "", err
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("invalid image format: %w", err)
	}

	if err := os.MkdirAll(s.uploadsDir, 0o755); err != nil {
		return "", err
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	resized := false
	if w > 1000 {
		newH := (h * 1000) / w
		img = resizeImage(img, 1000, newH)
		resized = true
	}

	ext := format
	if ext == "" {
		ext = "png"
	}
	filename := newUUID() + "." + ext
	outPath := filepath.Join(s.uploadsDir, filename)

	// Si no hacía falta redimensionar, volcamos los bytes originales directamente sin re-encodear.
	if !resized {
		return filename, os.WriteFile(outPath, data, 0o644)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	switch format {
	case "jpeg":
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
	default:
		err = png.Encode(out, img)
	}
	return filename, err
}

func resizeImage(src image.Image, dstW, dstH int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	for y := 0; y < dstH; y++ {
		sy := b.Min.Y + (y*srcH)/dstH
		for x := 0; x < dstW; x++ {
			sx := b.Min.X + (x*srcW)/dstW
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
