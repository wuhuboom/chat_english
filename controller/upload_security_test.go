package controller

import "testing"

func TestUploadSignaturesRejectDisguisedScripts(t *testing.T) {
	if validUploadSignature(".jpg", []byte("<?php system($_GET['x']); ?>")) {
		t.Fatal("PHP disguised as JPG was accepted")
	}
	if validUploadSignature(".txt", []byte("<script>alert(1)</script>")) {
		t.Fatal("active HTML disguised as TXT was accepted")
	}
	if !validUploadSignature(".png", []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatal("valid PNG signature was rejected")
	}
}
