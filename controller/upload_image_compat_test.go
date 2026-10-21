package controller

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func TestUploadImageFormatCompatibility(t *testing.T) {
	var jpg, pngData bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	if err := jpeg.Encode(&jpg, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"image", "attachment"} {
		for _, tc := range []struct {
			name string
			data []byte
			want string
		}{
			{"clipboard.png", jpg.Bytes(), ".jpg"},
			{"photo.JPG", pngData.Bytes(), ".png"},
			{"photo.jpeg", jpg.Bytes(), ".jpeg"},
			{"photo.png", pngData.Bytes(), ".png"},
			{"script.png", []byte("<?php echo 'not an image'; ?>"), ""},
			{"script.svg", []byte("<svg onload='alert(1)'/>"), ""},
			{"script.php", jpg.Bytes(), ""},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var body bytes.Buffer
				w := multipart.NewWriter(&body)
				part, e := w.CreateFormFile("file", tc.name)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = part.Write(tc.data); e != nil {
					t.Fatal(e)
				}
				if e = w.Close(); e != nil {
					t.Fatal(e)
				}
				req := httptest.NewRequest("POST", "/", &body)
				req.Header.Set("Content-Type", w.FormDataContentType())
				if e = req.ParseMultipartForm(1024 * 1024); e != nil {
					t.Fatal(e)
				}
				defer req.MultipartForm.RemoveAll()
				got, e := validateUpload(req.MultipartForm.File["file"][0], kind)
				if tc.want == "" {
					if e == nil {
						t.Fatal("unsafe file accepted")
					}
					return
				}
				if e != nil || got != tc.want {
					t.Fatalf("got %q, %v; want %q", got, e, tc.want)
				}
			})
		}
	}
}
